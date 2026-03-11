package errors

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCodeToGRPCCode(t *testing.T) {
	tests := []struct {
		code     Code
		expected codes.Code
	}{
		{CodeNotFound, codes.NotFound},
		{CodeAlreadyExists, codes.AlreadyExists},
		{CodePermissionDenied, codes.PermissionDenied},
		{CodeUnauthenticated, codes.Unauthenticated},
		{CodeInvalidArgument, codes.InvalidArgument},
		{CodeUnavailable, codes.Unavailable},
		{CodeInternal, codes.Internal},
	}

	for _, tt := range tests {
		result := tt.code.ToGRPCCode()
		if result != tt.expected {
			t.Errorf("expected %v, got %v", tt.expected, result)
		}
	}
}

func TestCodeToHTTPCode(t *testing.T) {
	tests := []struct {
		code     Code
		expected int
	}{
		{CodeNotFound, 404},
		{CodeAlreadyExists, 409},
		{CodePermissionDenied, 403},
		{CodeUnauthenticated, 401},
		{CodeInvalidArgument, 400},
		{CodeUnavailable, 503},
		{CodeInternal, 500},
	}

	for _, tt := range tests {
		result := tt.code.ToHTTPCode()
		if result != tt.expected {
			t.Errorf("expected %v, got %v", tt.expected, result)
		}
	}
}

func TestError(t *testing.T) {
	err := CodeNotFound.Error("record not found")
	s, _ := status.FromError(err)

	if s.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", s.Code())
	}
	if s.Message() != "record not found" {
		t.Errorf("expected message, got %s", s.Message())
	}
}

func TestWrap(t *testing.T) {
	original := errors.New("original error")
	err := CodeNotFound.Wrap(original)
	s, _ := status.FromError(err)

	if s.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", s.Code())
	}
}

func TestWrapNil(t *testing.T) {
	err := CodeNotFound.Wrap(nil)
	if err != nil {
		t.Error("expected nil error")
	}
}

func TestFromError(t *testing.T) {
	tests := []struct {
		err      error
		expected Code
	}{
		{status.Error(codes.NotFound, "not found"), CodeNotFound},
		{status.Error(codes.AlreadyExists, "exists"), CodeAlreadyExists},
		{status.Error(codes.PermissionDenied, "forbidden"), CodePermissionDenied},
		{status.Error(codes.Unauthenticated, "unauthorized"), CodeUnauthenticated},
		{status.Error(codes.InvalidArgument, "invalid"), CodeInvalidArgument},
		{status.Error(codes.Unavailable, "unavailable"), CodeUnavailable},
		{status.Error(codes.Internal, "internal"), CodeInternal},
		{errors.New("random"), CodeInternal},
		{nil, ""},
	}

	for _, tt := range tests {
		result := FromError(tt.err)
		if result != tt.expected {
			t.Errorf("expected %v, got %v", tt.expected, result)
		}
	}
}
