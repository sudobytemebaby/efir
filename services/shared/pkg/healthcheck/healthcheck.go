// Package healthcheck provides HTTP health and readiness endpoints.
package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Handler struct {
	ready atomic.Bool
}

type Response struct {
	Status string `json:"status"`
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) SetReady(ready bool) {
	h.ready.Store(ready)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response{Status: "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if !h.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(Response{Status: "not ready"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response{Status: "ready"})
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/ready", h.Ready)
}

func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			h.Health(w, r)
			return
		case "/ready":
			h.Ready(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) AwaitReady(ctx context.Context, checkFn func() bool, intervalMs int) {
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if checkFn() {
				h.SetReady(true)
				return
			}
		}
	}
}
