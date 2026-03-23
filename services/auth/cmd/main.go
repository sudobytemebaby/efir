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
		return fmt.Errorf("load config: %w", err)
	}

	logLevel, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		slog.Warn("invalid log level in config, falling back to info", "value", cfg.LogLevel)
		logLevel = logger.LevelInfo
	}

	l := logger.New(logger.Options{Level: logLevel})
	slog.SetDefault(l)

	pgPool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	valkeyClient, err := vk.NewClient(vk.ClientOption{
		InitAddress: []string{cfg.ValkeyAddr},
		Password:    cfg.ValkeyPass,
	})
	if err != nil {
		return fmt.Errorf("connect to valkey: %w", err)
	}
	defer valkeyClient.Close()

	nc, err := sharednats.Connect(cfg.NATSURL, cfg.NATSUser, cfg.NATSPass)
	if err != nil {
		return fmt.Errorf("connect to nats: %w", err)
	}
	defer nc.Close()

	js, err := sharednats.New(nc)
	if err != nil {
		return fmt.Errorf("create jetstream context: %w", err)
	}

	if err := sharednats.ProvisionStreams(ctx, js, nats.Streams()); err != nil {
		return fmt.Errorf("provision nats streams: %w", err)
	}

	limiter := ratelimit.NewValkeyLimiter(valkeyClient, cfg.RateLimitRequests, cfg.RateLimitWindow)

	accountRepo := repository.NewAccountRepository(pgPool)
	tokenRepo := repository.NewTokenRepository(valkeyClient)
	publisher := nats.NewPublisher(js)

	authSvc := service.NewAuthService(
		accountRepo,
		tokenRepo,
		publisher,
		limiter,
		cfg.JWTSecret,
		cfg.AccessTTL,
		cfg.RefreshTTL,
	)

	authHandler, err := handler.NewAuthHandler(authSvc)
	if err != nil {
		return fmt.Errorf("create auth handler: %w", err)
	}

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

	healthHandler := healthcheck.New()
	healthHandler.SetReady(true)
	healthMux := http.NewServeMux()
	healthHandler.Register(healthMux)
	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           healthMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

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

	select {
	case err := <-errCh:
		grpcServer.GracefulStop()
		return fmt.Errorf("server error: %w", err)
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
	return nil
}
