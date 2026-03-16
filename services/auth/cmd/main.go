package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sudobytemebaby/efir/services/auth/internal/config"
	"github.com/sudobytemebaby/efir/services/auth/internal/handler"
	"github.com/sudobytemebaby/efir/services/auth/internal/nats"
	"github.com/sudobytemebaby/efir/services/auth/internal/ratelimit"
	"github.com/sudobytemebaby/efir/services/auth/internal/repository"
	"github.com/sudobytemebaby/efir/services/auth/internal/service"
	authv1 "github.com/sudobytemebaby/efir/services/shared/gen/auth"
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/middleware"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	vk "github.com/valkey-io/valkey-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

	// 1. Database
	pgPool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		slog.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}

	// 2. Valkey
	valkeyClient, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{cfg.ValkeyAddr},
		Password:    cfg.ValkeyPass,
	})
	if err != nil {
		slog.Error("failed to connect to valkey", "error", err)
		os.Exit(1)
	}
	defer valkeyClient.Close()

	// 3. NATS
	nc, err := sharednats.Connect(cfg.NATSURL, cfg.NATSUser, cfg.NATSPass)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := sharednats.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream context", "error", err)
		os.Exit(1)
	}

	if err := sharednats.ProvisionStreams(ctx, js, nats.Streams()); err != nil {
		slog.Error("failed to provision nats streams", "error", err)
		os.Exit(1)
	}

	// 4. Rate limiter
	rateLimitWindow, err := cfg.ParseRateLimitWindow()
	if err != nil {
		slog.Error("invalid rate limit window", "error", err)
		os.Exit(1)
	}
	limiter := ratelimit.NewValkeyLimiter(valkeyClient, cfg.RateLimitRequests, rateLimitWindow)

	// 5. Initialize layers
	accountRepo := repository.NewAccountRepository(pgPool)
	tokenRepo := repository.NewTokenRepository(valkeyClient)
	publisher := nats.NewPublisher(js)

	accessTTL, err := cfg.ParseAccessTTL()
	if err != nil {
		slog.Error("invalid access ttl", "error", err)
		os.Exit(1)
	}
	refreshTTL, err := cfg.ParseRefreshTTL()
	if err != nil {
		slog.Error("invalid refresh ttl", "error", err)
		os.Exit(1)
	}

	authSvc := service.NewAuthService(
		accountRepo,
		tokenRepo,
		publisher,
		limiter,
		cfg.JWTSecret,
		accessTTL,
		refreshTTL,
	)

	// 6. Handler
	authHandler, err := handler.NewAuthHandler(authSvc)
	if err != nil {
		slog.Error("failed to create auth handler", "error", err)
		os.Exit(1)
	}

	// 7. gRPC Server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor(l),
			middleware.LoggingInterceptor(l),
		),
	)
	authv1.RegisterAuthServiceServer(grpcServer, authHandler)
	if cfg.Env == config.EnvDevelopment {
		reflection.Register(grpcServer)
	}

	// 8. Healthcheck Server
	healthHandler := healthcheck.New()
	healthHandler.SetReady(true)
	healthMux := http.NewServeMux()
	healthHandler.Register(healthMux)
	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           healthMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// 9. Start servers
	errCh := make(chan error, 2)

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
		if err != nil {
			errCh <- fmt.Errorf("grpc listen: %w", err)
			return
		}
		slog.Info("grpc server started", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	go func() {
		slog.Info("health server started", "addr", healthServer.Addr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health serve: %w", err)
		}
	}()

	// 10. Wait for shutdown
	select {
	case err := <-errCh:
		slog.Error("server error", "error", err)
	case <-ctx.Done():
		slog.Info("shutting down servers")
	}

	grpcServer.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down health server", "error", err)
	}

	select {
	case err := <-errCh:
		slog.Error("secondary server error", "error", err)
	default:
	}

	slog.Info("service stopped")
}
