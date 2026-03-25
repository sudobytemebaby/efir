package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sudobytemebaby/efir/services/gateway/internal/config"
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
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logLevel, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		slog.Warn("invalid log level in config, falling back to info", "value", cfg.LogLevel)
		logLevel = logger.LevelInfo
	}

	l := logger.New(logger.Options{Level: logLevel})
	slog.SetDefault(l)

	valkeyClient, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{cfg.ValkeyAddr},
		Password:    cfg.ValkeyPass,
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
		grpc.WithUnaryInterceptor(timeoutInterceptor(cfg.GRPCTimeout)),
	}

	authConn, err := grpc.NewClient(cfg.AuthServiceAddr, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = authConn.Close() }()

	userConn, err := grpc.NewClient(cfg.UserServiceAddr, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = userConn.Close() }()

	roomConn, err := grpc.NewClient(cfg.RoomServiceAddr, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = roomConn.Close() }()

	messageConn, err := grpc.NewClient(cfg.MessageServiceAddr, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = messageConn.Close() }()

	jwtMiddleware := middleware.JWTAuth(cfg.JWTSecret)
	ipRateLimiter := middleware.IPRateLimiter(valkeyClient, cfg.RateLimitRequests, cfg.RateLimitWindow)
	userRateLimiter := middleware.UserRateLimiter(valkeyClient, cfg.RateLimitRequests, cfg.RateLimitWindow)

	healthHandler := healthcheck.New()

	authHandler := auth.NewHandler(authv1.NewAuthServiceClient(authConn))
	userHandler := user.NewHandler(userv1.NewUserServiceClient(userConn))
	roomHandler := room.NewHandler(roomv1.NewRoomServiceClient(roomConn))
	messageHandler := message.NewHandler(messagev1.NewMessageServiceClient(messageConn))
	wsAuthHandler := wsauth.NewHandler(valkeyClient, cfg.WSTicketTTL)

	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(ipRateLimiter)
		authHandler.Register(r)
	})

	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware)
		r.Use(userRateLimiter)
		userHandler.Register(r)
		roomHandler.Register(r)
		messageHandler.Register(r)
	})

	wsAuthHandler.Register(r)

	r.HandleFunc("/health", healthHandler.Health)
	r.HandleFunc("/ready", healthHandler.Ready)

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("gateway"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
