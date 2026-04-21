package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sudobytemebaby/efir/services/gateway/internal/config"
	gatewayhandler "github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/auth"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/message"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/room"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/user"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler/wsauth"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	roomv1 "github.com/sudobytemebaby/efir/services/shared/gen/room"
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	vk "github.com/valkey-io/valkey-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("service error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		slog.Warn("invalid log level in config, falling back to info", "value", cfg.LogLevel)
		logLevel = logger.LevelInfo
	}

	l := logger.New(logger.Options{Level: logLevel})
	slog.SetDefault(l)

	valkeyClient, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{cfg.Cache.Addr},
		Password:    cfg.Cache.Pass,
	})
	if err != nil {
		return err
	}
	defer valkeyClient.Close()

	if err := valkeyClient.Do(ctx, valkeyClient.B().Ping().Build()).Error(); err != nil {
		return err
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(timeoutInterceptor(cfg.Timeouts.GRPC)),
	}

	authConn, err := grpc.NewClient(cfg.Services.Auth, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = authConn.Close() }()

	userConn, err := grpc.NewClient(cfg.Services.User, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = userConn.Close() }()

	roomConn, err := grpc.NewClient(cfg.Services.Room, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = roomConn.Close() }()

	messageConn, err := grpc.NewClient(cfg.Services.Message, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = messageConn.Close() }()

	jwtMiddleware := middleware.JWTAuth(cfg.Auth.Secret)
	userRateLimiter := middleware.UserRateLimiter(valkeyClient, cfg.RateLimit.Requests, cfg.RateLimit.Window)

	healthHandler := healthcheck.New()

	gatewayhandler.SetMaxBodySize(cfg.Server.MaxBodySize)

	authHandler := auth.NewHandler(
		authv1.NewAuthServiceClient(authConn),
		cfg.Env == sharedcfg.EnvProduction,
	)
	userHandler := user.NewHandler(userv1.NewUserServiceClient(userConn))
	roomHandler := room.NewHandler(roomv1.NewRoomServiceClient(roomConn))
	messageHandler := message.NewHandler(messagev1.NewMessageServiceClient(messageConn))
	wsAuthHandler := wsauth.NewHandler(valkeyClient, cfg.Auth.WSTicketTTL)

	healthHandler.SetReady(true)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)

	// Public — no auth required
	r.Group(func(r chi.Router) {
		authHandler.RegisterPublic(r)
		authHandler.RegisterSession(r)
	})

	// Protected — JWT + rate limit
	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Use(userRateLimiter)
		authHandler.RegisterProtected(r)
		wsAuthHandler.RegisterProtected(r)
		userHandler.Register(r)
		roomHandler.Register(r)
		messageHandler.Register(r)
	})

	// Internal — service-to-service
	r.Group(func(r chi.Router) {
		wsAuthHandler.RegisterInternal(r)
	})

	// Health
	r.HandleFunc("/health", healthHandler.Health)
	r.HandleFunc("/ready", healthHandler.Ready)

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           r,
		ReadHeaderTimeout: cfg.Timeouts.ReadHeader,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("gateway service started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.Shutdown)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down gateway service", "error", err)
	}

	slog.Info("gateway service stopped gracefully")
	return nil
}

func timeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
