package errors

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusError_Error(t *testing.T) {
	se := &StatusError{code: CodeNotFound, msg: "not found"}
	if se.Error() != "not found" {
		t.Errorf("expected 'not found', got %q", se.Error())
	}
}

func TestStatusError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	se := &StatusError{code: CodeInternal, msg: "internal error", err: inner}
	if se.Unwrap() != inner {
		t.Errorf("expected unwrap to return inner error")
	}
}

func TestStatusError_UnwrapNil(t *testing.T) {
	se := &StatusError{code: CodeInternal, msg: "internal error"}
	if se.Unwrap() != nil {
		t.Errorf("expected nil unwrap")
	}
}

func TestStatusError_GRPCStatus(t *testing.T) {
	se := &StatusError{code: CodeNotFound, msg: "not found"}
	gs := se.GRPCStatus()
	if gs.Code() != codes.NotFound {
		t.Errorf("expected codes.NotFound, got %v", gs.Code())
	}
	if gs.Message() != "not found" {
		t.Errorf("expected 'not found', got %q", gs.Message())
	}
}

func TestStatusError_ErrorsIs(t *testing.T) {
	inner := errors.New("inner error")
	se := &StatusError{code: CodeInternal, msg: "internal error", err: inner}
	if !errors.Is(se, inner) {
		t.Error("errors.Is should work through StatusError chain")
	}
}

func TestStatusError_ErrorsAs(t *testing.T) {
	inner := errors.New("inner error")
	se := &StatusError{code: CodeInternal, msg: "internal error", err: inner}
	var target *StatusError
	if !errors.As(se, &target) {
		t.Error("errors.As should find StatusError in chain")
	}
	if target.code != CodeInternal {
		t.Errorf("expected CodeInternal, got %v", target.code)
	}
}

func TestCode_Wrap(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := CodeInternal.Wrap(inner)

	var se *StatusError
	if !errors.As(wrapped, &se) {
		t.Fatal("expected StatusError")
	}
	if se.code != CodeInternal {
		t.Errorf("expected CodeInternal, got %v", se.code)
	}
	if se.msg != codeToDefaultMessage[CodeInternal] {
		t.Errorf("expected default message, got %q", se.msg)
	}
	if se.err != inner {
		t.Errorf("expected inner error to be preserved")
	}
}

func TestCode_WrapNil(t *testing.T) {
	if CodeInternal.Wrap(nil) != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestCode_Wrap_AlreadyCorrectCode(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := CodeInternal.Wrap(inner)

	wrapped2 := CodeInternal.Wrap(wrapped)
	if wrapped != wrapped2 {
		t.Error("re-wrapping with same code should return original")
	}
}

func TestCode_Error(t *testing.T) {
	err := CodeNotFound.Error("custom not found message")
	var se *StatusError
	if !errors.As(err, &se) {
		t.Fatal("expected StatusError")
	}
	if se.code != CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", se.code)
	}
	if se.msg != "custom not found message" {
		t.Errorf("expected 'custom not found message', got %q", se.msg)
	}
}

func TestFromError_StatusError(t *testing.T) {
	se := &StatusError{code: CodeNotFound, msg: "not found"}
	code, ok := FromError(se)
	if !ok {
		t.Fatal("expected ok")
	}
	if code != CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", code)
	}
}

func TestFromError_GRPCStatus(t *testing.T) {
	gs := status.New(codes.NotFound, "not found")
	err := gs.Err()

	code, ok := FromError(err)
	if !ok {
		t.Fatal("expected ok")
	}
	if code != CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", code)
	}
}

func TestFromError_Nil(t *testing.T) {
	_, ok := FromError(nil)
	if ok {
		t.Error("expected false for nil error")
	}
}

func TestToGRPCCode(t *testing.T) {
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
		{Code("UNKNOWN"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.ToGRPCCode(); got != tt.expected {
				t.Errorf("ToGRPCCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestToHTTPCode(t *testing.T) {
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
		{Code("UNKNOWN"), 500},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.ToHTTPCode(); got != tt.expected {
				t.Errorf("ToHTTPCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCodeToDefaultMessage(t *testing.T) {
	codes := []Code{CodeNotFound, CodeAlreadyExists, CodePermissionDenied, CodeUnauthenticated, CodeInvalidArgument, CodeUnavailable, CodeInternal}
	for _, c := range codes {
		if msg, ok := codeToDefaultMessage[c]; !ok {
			t.Errorf("missing default message for %v", c)
		} else if msg == "" {
			t.Errorf("empty default message for %v", c)
		}
	}
}
