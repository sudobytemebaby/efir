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
	userv1 "github.com/sudobytemebaby/efir/services/shared/gen/user"
	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/middleware"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/user/internal/config"
	"github.com/sudobytemebaby/efir/services/user/internal/handler"
	"github.com/sudobytemebaby/efir/services/user/internal/nats"
	"github.com/sudobytemebaby/efir/services/user/internal/repository"
	"github.com/sudobytemebaby/efir/services/user/internal/service"
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

	pgPool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	nc, err := sharednats.Connect(cfg.NATS.URL, cfg.NATS.User, cfg.NATS.Pass, sharednats.ConnectOptions{
		ReconnectWait: cfg.NATS.ReconnectWait,
		MaxReconnects: cfg.NATS.MaxReconnects,
	})
	if err != nil {
		return fmt.Errorf("connect to nats: %w", err)
	}
	defer nc.Close()

	js, err := sharednats.New(nc)
	if err != nil {
		return fmt.Errorf("create jetstream context: %w", err)
	}

	userRepo := repository.NewUserRepository(pgPool)
	userSvc := service.NewUserService(userRepo)

	userHandler, err := handler.NewUserHandler(userSvc)
	if err != nil {
		return fmt.Errorf("create user handler: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDInterceptor(),
			middleware.RecoveryInterceptor(l),
			middleware.LoggingInterceptor(l),
		),
	)
	userv1.RegisterUserServiceServer(grpcServer, userHandler)
	if cfg.Env == sharedcfg.EnvDevelopment {
		reflection.Register(grpcServer)
	}

	healthHandler := healthcheck.New()
	healthMux := http.NewServeMux()
	healthHandler.Register(healthMux)
	healthServer := &http.Server{
		Addr:              ":8080",
		Handler:           healthMux,
		ReadHeaderTimeout: cfg.Timeouts.ReadHeader,
	}

	subscriber := nats.NewSubscriber(js, userSvc, nats.SubscriberConfig{
		MaxDeliver: cfg.NATS.MaxDeliver,
		AckWait:    cfg.NATS.AckWait,
		RetryWait:  cfg.NATS.ConsumerRetryWait,
	})
	if err := subscriber.Start(ctx); err != nil {
		return fmt.Errorf("start NATS subscriber: %w", err)
	}

	errCh := make(chan error, 2)

	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}
	healthLis, err := net.Listen("tcp", cfg.Server.HealthListenAddr)
	if err != nil {
		return fmt.Errorf("health listen: %w", err)
	}

	go func() {
		slog.Info("grpc server started", "port", cfg.Server.GRPCPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- fmt.Errorf("grpc serve: %w", err)
		}
	}()

	go func() {
		slog.Info("health server started", "addr", healthServer.Addr)
		if err := healthServer.Serve(healthLis); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health serve: %w", err)
		}
	}()

	healthHandler.SetReady(true)

	select {
	case err := <-errCh:
		grpcServer.GracefulStop()
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutting down servers")
	}

	grpcDone := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(grpcDone) }()
	select {
	case <-grpcDone:
		slog.Info("grpc server stopped gracefully")
	case <-time.After(cfg.Timeouts.GRPCGraceful):
		grpcServer.Stop()
		slog.Warn("grpc server force stopped after timeout")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.Shutdown)
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
