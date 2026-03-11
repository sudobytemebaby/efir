package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
