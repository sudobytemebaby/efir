package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	h := New()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	h.Health(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}
}

func TestReadyNotReady(t *testing.T) {
	h := New()
	h.ready.Store(false)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ready", nil)

	h.Ready(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestReadyReady(t *testing.T) {
	h := New()
	h.ready.Store(true)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ready", nil)

	h.Ready(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSetReady(t *testing.T) {
	h := New()

	if h.ready.Load() {
		t.Error("expected initial ready to be false")
	}

	h.SetReady(true)

	if !h.ready.Load() {
		t.Error("expected ready to be true after SetReady(true)")
	}
}

func TestMiddlewarePassthrough(t *testing.T) {
	h := New()
	h.ready.Store(true)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := h.Middleware(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/other", nil)
	middleware.ServeHTTP(w, r)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMiddlewareHealth(t *testing.T) {
	h := New()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	middleware := h.Middleware(next)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	middleware.ServeHTTP(w, r)

	if nextCalled {
		t.Error("expected next handler NOT to be called for /health")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
