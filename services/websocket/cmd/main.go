package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/websocket/internal/config"
	"github.com/sudobytemebaby/efir/services/websocket/internal/handler"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	"github.com/sudobytemebaby/efir/services/websocket/internal/subscriber"
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

	nc, err := nats.Connect(cfg.NATSURL, cfg.NATSUser, cfg.NATSPass)
	if err != nil {
		slog.Error("failed to connect to nats", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nats.New(nc)
	if err != nil {
		slog.Error("failed to create jetstream", "error", err)
		os.Exit(1)
	}

	wsHub := hub.NewHub()
	go wsHub.Run(ctx)

	sub := subscriber.NewSubscriber(wsHub, js)
	if err := sub.Start(ctx); err != nil {
		slog.Error("failed to start subscriber", "error", err)
		os.Exit(1)
	}

	wsHandler := handler.NewWebSocketHandler(wsHub, cfg.GatewayURL, valkeyClient, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", okHandler)
	mux.HandleFunc("/ready", okHandler)
	mux.HandleFunc("/ws", wsHandler.HandleWS)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("websocket service started", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		slog.Error("websocket service stopped unexpectedly", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down websocket service", "error", err)
		os.Exit(1)
	}

	slog.Info("websocket service stopped gracefully")
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
