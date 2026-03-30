package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sudobytemebaby/efir/services/gateway/internal/handler"
	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSetMaxBodySize(t *testing.T) {
	t.Run("positive value takes effect", func(t *testing.T) {
		handler.SetMaxBodySize(10)
		defer handler.SetMaxBodySize(1 << 20) // restore default

		body := strings.Repeat("x", 20)
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var msg wrapperspb.StringValue
		err := handler.ReadProto(r, &msg)
		assert.Error(t, err)
	})

	t.Run("zero is ignored", func(t *testing.T) {
		handler.SetMaxBodySize(0)
		// should not panic or change behavior
	})

	t.Run("negative is ignored", func(t *testing.T) {
		handler.SetMaxBodySize(-1)
	})
}

func TestWriteCode(t *testing.T) {
	tests := []struct {
		code       errors.Code
		wantStatus int
		wantCode   string
	}{
		{errors.CodeNotFound, 404, "NOT_FOUND"},
		{errors.CodeAlreadyExists, 409, "ALREADY_EXISTS"},
		{errors.CodePermissionDenied, 403, "PERMISSION_DENIED"},
		{errors.CodeUnauthenticated, 401, "UNAUTHENTICATED"},
		{errors.CodeInvalidArgument, 400, "INVALID_ARGUMENT"},
		{errors.CodeUnavailable, 503, "UNAVAILABLE"},
		{errors.CodeInternal, 500, "INTERNAL"},
		{errors.CodeRateLimited, 429, "RATE_LIMITED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.WriteCode(w, tt.code)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var body struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tt.wantCode, body.Code)
			assert.NotEmpty(t, body.Error)
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	handler.WriteJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "value", body["key"])
}

func TestWriteError(t *testing.T) {
	t.Run("gRPC status error maps correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		err := status.Error(codes.NotFound, "not found")
		handler.WriteError(w, r, err, "test")

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("plain error maps to internal", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.WriteError(w, r, fmt.Errorf("something broke"), "test")

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestWriteProto(t *testing.T) {
	t.Run("valid message", func(t *testing.T) {
		w := httptest.NewRecorder()
		msg := wrapperspb.String("hello")
		handler.WriteProto(w, http.StatusOK, msg)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Contains(t, w.Body.String(), "hello")
	})

	t.Run("nil message", func(t *testing.T) {
		w := httptest.NewRecorder()
		handler.WriteProto(w, http.StatusOK, nil)

		// nil proto message should marshal to "{}"
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestReadProto(t *testing.T) {
	t.Run("valid body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`"test"`))
		var msg wrapperspb.StringValue
		err := handler.ReadProto(r, &msg)
		assert.NoError(t, err)
		assert.Equal(t, "test", msg.GetValue())
	})

	t.Run("invalid JSON", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{invalid`))
		var msg wrapperspb.StringValue
		err := handler.ReadProto(r, &msg)
		assert.Error(t, err)
	})

	t.Run("body too large", func(t *testing.T) {
		handler.SetMaxBodySize(5)
		defer handler.SetMaxBodySize(1 << 20)

		body := strings.Repeat("x", 100)
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var msg wrapperspb.StringValue
		err := handler.ReadProto(r, &msg)
		assert.Error(t, err)
	})
}
