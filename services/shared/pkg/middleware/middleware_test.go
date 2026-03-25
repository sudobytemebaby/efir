package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGetUserID(t *testing.T) {
	ctx := context.Background()

	_, ok := GetUserID(ctx)
	if ok {
		t.Error("expected no userID in empty context")
	}

	ctx = context.WithValue(ctx, contextKeyUserID{}, "test-user-id")

	userID, ok := GetUserID(ctx)
	if !ok {
		t.Error("expected userID to be found")
	}
	if userID != "test-user-id" {
		t.Errorf("expected test-user-id, got %s", userID)
	}
}

func TestUserIDInterceptor(t *testing.T) {
	interceptor := UserIDInterceptor()

	md := metadata.Pairs(MetadataKeyUserID, "user-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		userID, ok := GetUserID(ctx)
		if !ok {
			t.Error("expected userID in context from interceptor")
		}
		if userID != "user-123" {
			t.Errorf("expected user-123, got %s", userID)
		}
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	if resp != "response" {
		t.Errorf("unexpected response: %v", resp)
	}
}

func TestUserIDInterceptorNoMetadata(t *testing.T) {
	interceptor := UserIDInterceptor()

	ctx := context.Background()

	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		_, ok := GetUserID(ctx)
		if ok {
			t.Error("did not expect userID in context")
		}
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	_ = resp
}

func TestRecoveryInterceptor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		panic("something went wrong")
	}

	resp, err := interceptor(ctx, req, info, handler)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
	if err == nil {
		t.Error("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expected status error, got %T", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected Internal code, got %v", st.Code())
	}
}

func TestRecoveryInterceptorNoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestLoggingInterceptorSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	interceptor := LoggingInterceptor(logger)

	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(ctx, req, info, handler)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if resp != "response" {
		t.Errorf("expected response, got %v", resp)
	}
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("expected log output")
	}
}

func TestLoggingInterceptorError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	interceptor := LoggingInterceptor(logger)

	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handlerCalled := false
	expectedErr := status.Error(codes.NotFound, "not found")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, expectedErr
	}

	resp, err := interceptor(ctx, req, info, handler)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
	if err != expectedErr {
		t.Errorf("expected expectedErr, got %v", err)
	}

	logOutput := buf.String()
	if logOutput == "" {
		t.Error("expected log output")
	}
}

func TestLoggingInterceptorDurationRecorded(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	interceptor := LoggingInterceptor(logger)

	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	_, _ = interceptor(ctx, req, info, handler)

	logOutput := buf.String()
	if logOutput == "" {
		t.Fatal("expected log output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("duration_ms")) {
		t.Error("expected duration_ms in log output")
	}
}
