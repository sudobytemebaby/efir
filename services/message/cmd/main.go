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
	"github.com/sudobytemebaby/efir/services/message/internal/client"
	"github.com/sudobytemebaby/efir/services/message/internal/config"
	"github.com/sudobytemebaby/efir/services/message/internal/handler"
	"github.com/sudobytemebaby/efir/services/message/internal/nats"
	"github.com/sudobytemebaby/efir/services/message/internal/repository"
	"github.com/sudobytemebaby/efir/services/message/internal/service"
	messagev1 "github.com/sudobytemebaby/efir/services/shared/gen/message"
	sharedcfg "github.com/sudobytemebaby/efir/services/shared/pkg/config"
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/middleware"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
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
		return fmt.Errorf("create jetstream: %w", err)
	}

	if err := sharednats.ProvisionStreams(ctx, js, nats.Streams()); err != nil {
		return fmt.Errorf("provision nats streams: %w", err)
	}

	roomClient, err := client.NewRoomClient(cfg.Room.Addr, cfg.Room.CallTimeout, cfg.Room.RetryDelays)
	if err != nil {
		return fmt.Errorf("create room client: %w", err)
	}
	defer func() {
		if err := roomClient.Close(); err != nil {
			slog.Warn("failed to close room client", "error", err)
		}
	}()

	msgRepo := repository.NewMessageRepository(pgPool)
	publisher := nats.NewPublisher(js)
	msgSvc := service.NewMessageService(msgRepo, roomClient, publisher)
	msgHandler, err := handler.NewMessageHandler(msgSvc)
	if err != nil {
		return fmt.Errorf("create message handler: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RequestIDInterceptor(),
			middleware.RecoveryInterceptor(l),
			middleware.LoggingInterceptor(l),
		),
	)
	messagev1.RegisterMessageServiceServer(grpcServer, msgHandler)

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

	errCh := make(chan error, 2)

	grpcLis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		return fmt.Errorf("create grpc listener: %w", err)
	}
	healthLis, err := net.Listen("tcp", cfg.Server.HealthListenAddr)
	if err != nil {
		return fmt.Errorf("create health listener: %w", err)
	}

	go func() {
		slog.Info("starting gRPC server", "addr", grpcLis.Addr())
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		slog.Info("starting health server", "addr", healthServer.Addr)
		if err := healthServer.Serve(healthLis); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health server error: %w", err)
		}
	}()

	healthHandler.SetReady(true)

	select {
	case err := <-errCh:
		grpcServer.GracefulStop()
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
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

	slog.Info("server stopped gracefully")
	return nil
}
