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

	authClient, err := client.NewAuthClient(cfg.AuthServiceAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer func() {
		if err := authClient.Close(); err != nil {
			slog.Warn("failed to close auth client", "error", err)
		}
	}()

	userClient, err := client.NewUserClient(cfg.UserServiceAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer func() {
		if err := userClient.Close(); err != nil {
			slog.Warn("failed to close user client", "error", err)
		}
	}()

	roomClient, err := client.NewRoomClient(cfg.RoomServiceAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer func() {
		if err := roomClient.Close(); err != nil {
			slog.Warn("failed to close room client", "error", err)
		}
	}()

	messageClient, err := client.NewMessageClient(cfg.MessageServiceAddr, cfg.GRPCTimeout)
	if err != nil {
		return err
	}
	defer func() {
		if err := messageClient.Close(); err != nil {
			slog.Warn("failed to close message client", "error", err)
		}
	}()

	jwtMiddleware := middleware.JWTAuth(cfg.JWTSecret)
	ipRateLimiter := middleware.IPRateLimiter(valkeyClient, cfg.RateLimitRequests, cfg.RateLimitWindow)
	userRateLimiter := middleware.UserRateLimiter(valkeyClient, cfg.RateLimitRequests, cfg.RateLimitWindow)

	healthHandler := healthcheck.New()

	httpHandler := handler.NewHTTPHandler(authClient)
	userHandler := handler.NewUserHandler(userClient)
	roomHandler := handler.NewRoomHandler(roomClient)
	messageHandler := handler.NewMessageHandler(messageClient)
	wsAuthHandler := handler.NewWSAuthHandler(valkeyClient, cfg.WSTicketTTL)

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
