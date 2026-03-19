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
	"github.com/sudobytemebaby/efir/services/shared/pkg/healthcheck"
	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/middleware"
	sharednats "github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logLevel, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level: %v\n", err)
		os.Exit(1)
	}
	l := logger.New(logger.Options{Level: logLevel})
	slog.SetDefault(l)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	nc, err := sharednats.Connect(cfg.NATSURL, cfg.NATSUser, cfg.NATSPass)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := sharednats.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		os.Exit(1)
	}

	if err := sharednats.ProvisionStreams(ctx, js, nats.Streams()); err != nil {
		slog.Error("failed to provision nats streams", "error", err)
		os.Exit(1)
	}

	roomCallTimeout, err := cfg.ParseRoomCallTimeout()
	if err != nil {
		slog.Error("failed to parse room call timeout", "error", err)
		os.Exit(1)
	}

	roomClient, err := client.NewRoomClient(cfg.RoomServiceAddr, roomCallTimeout)
	if err != nil {
		slog.Error("failed to create room client", "error", err)
		os.Exit(1)
	}

	msgRepo := repository.NewMessageRepository(pgPool)
	publisher := nats.NewPublisher(js)
	msgSvc := service.NewMessageService(msgRepo, roomClient, publisher)
	msgHandler, err := handler.NewMessageHandler(msgSvc)
	if err != nil {
		slog.Error("failed to create message handler", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.RecoveryInterceptor(l),
			middleware.LoggingInterceptor(l),
		),
	)
	messagev1.RegisterMessageServiceServer(grpcServer, msgHandler)

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
		addr := ":" + cfg.GRPCPort
		slog.Info("starting gRPC server", "addr", addr)
		if err := grpcServer.Serve(listener(addr)); err != nil {
			errCh <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	go func() {
		slog.Info("starting health server", "addr", healthServer.Addr)
		if err := healthServer.Serve(listener(healthServer.Addr)); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		slog.Error("server stopped unexpectedly", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	grpcServer.GracefulStop()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down health server", "error", err)
	}

	slog.Info("server stopped gracefully")
}

func listener(addr string) net.Listener {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("failed to create listener", "addr", addr, "error", err)
		os.Exit(1)
	}
	return l
}
