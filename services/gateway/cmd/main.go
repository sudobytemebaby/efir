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
	"github.com/sudobytemebaby/efir/services/gateway/internal/client"
	"github.com/sudobytemebaby/efir/services/gateway/internal/config"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/gateway/internal/middleware"
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	vk "github.com/valkey-io/valkey-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logLevel, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		slog.Warn("invalid log level in config, falling back to info", "value", cfg.LogLevel)
		logLevel = logger.LevelInfo
	}

	l := logger.New(logger.Options{Level: logLevel})
	slog.SetDefault(l)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	valkeyClient, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{cfg.ValkeyAddr},
		Password:    cfg.ValkeyPass,
	})
	if err != nil {
		slog.Error("failed to connect to valkey", "error", err)
		os.Exit(1)
	}
	defer valkeyClient.Close()

	if err := valkeyClient.Do(ctx, valkeyClient.B().Ping().Build()).Error(); err != nil {
		slog.Error("failed to ping valkey", "error", err)
		os.Exit(1)
	}

	grpcTimeout, err := cfg.ParseGRPCTimeout()
	if err != nil {
		slog.Error("failed to parse grpc timeout", "error", err)
		os.Exit(1)
	}

	rateLimitWindow, err := cfg.ParseRateLimitWindow()
	if err != nil {
		slog.Error("failed to parse rate limit window", "error", err)
		os.Exit(1)
	}

	authClient, err := client.NewAuthClient(cfg.AuthServiceAddr, grpcTimeout)
	if err != nil {
		slog.Error("failed to create auth client", "error", err)
		os.Exit(1)
	}

	userClient, err := client.NewUserClient(cfg.UserServiceAddr, grpcTimeout)
	if err != nil {
		slog.Error("failed to create user client", "error", err)
		os.Exit(1)
	}

	roomClient, err := client.NewRoomClient(cfg.RoomServiceAddr, grpcTimeout)
	if err != nil {
		slog.Error("failed to create room client", "error", err)
		os.Exit(1)
	}

	messageClient, err := client.NewMessageClient(cfg.MessageServiceAddr, grpcTimeout)
	if err != nil {
		slog.Error("failed to create message client", "error", err)
		os.Exit(1)
	}

	ticketTTL, err := cfg.ParseWSTicketTTL()
	if err != nil {
		slog.Error("failed to parse ticket TTL", "error", err)
		os.Exit(1)
	}

	jwtMiddleware := middleware.JWTAuth(cfg.JWTSecret)
	ipRateLimiter := middleware.IPRateLimiter(valkeyClient, cfg.RateLimitRequests, rateLimitWindow)
	userRateLimiter := middleware.UserRateLimiter(valkeyClient, cfg.RateLimitRequests, rateLimitWindow)

	healthHandler := healthcheck.New()

	httpHandler := handler.NewHTTPHandler(authClient)
	userHandler := handler.NewUserHandler(userClient)
	roomHandler := handler.NewRoomHandler(roomClient)
	messageHandler := handler.NewMessageHandler(messageClient)
	wsAuthHandler := handler.NewWSAuthHandler(valkeyClient, ticketTTL)

	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(ipRateLimiter)
		httpHandler.Register(r)
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
		slog.Error("gateway service stopped unexpectedly", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down gateway service", "error", err)
		os.Exit(1)
	}

	slog.Info("gateway service stopped gracefully")
}
