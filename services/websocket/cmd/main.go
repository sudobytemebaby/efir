package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sudobytemebaby/efir/services/shared/pkg/logger"
	"github.com/sudobytemebaby/efir/services/shared/pkg/nats"
	"github.com/sudobytemebaby/efir/services/websocket/internal/config"
	"github.com/sudobytemebaby/efir/services/websocket/internal/handler"
	"github.com/sudobytemebaby/efir/services/websocket/internal/hub"
	"github.com/sudobytemebaby/efir/services/websocket/internal/subscriber"
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

	nc, err := nats.Connect(cfg.NATS.URL, cfg.NATS.User, cfg.NATS.Pass, nats.ConnectOptions{
		ReconnectWait: cfg.NATS.ReconnectWait,
		MaxReconnects: cfg.NATS.MaxReconnects,
	})
	if err != nil {
		return err
	}
	defer nc.Close()

	js, err := nats.New(nc)
	if err != nil {
		return err
	}

	wsHub := hub.NewHub(cfg.Server.HubBufferSize)
	go wsHub.Run(ctx)

	sub := subscriber.NewSubscriber(wsHub, js, subscriber.SubscriberConfig{
		MaxDeliver: cfg.NATS.MaxDeliver,
		AckWait:    cfg.NATS.AckWait,
		RetryWait:  cfg.NATS.ConsumerRetryWait,
	})
	if err := sub.Start(ctx); err != nil {
		return err
	}

	wsHandler := handler.NewWebSocketHandler(wsHub, cfg.Services.GatewayURL, valkeyClient, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", okHandler)
	mux.HandleFunc("/ready", okHandler)
	mux.HandleFunc("/ws", wsHandler.HandleWS)

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           mux,
		ReadHeaderTimeout: cfg.Timeouts.ReadHeader,
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
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Timeouts.Shutdown)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down websocket service", "error", err)
	}

	slog.Info("websocket service stopped gracefully")
	return nil
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
