// Package errors provides error code mapping between gRPC, HTTP, and internal codes.
package errors

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Code string

const (
	CodeNotFound         Code = "NOT_FOUND"
	CodeAlreadyExists    Code = "ALREADY_EXISTS"
	CodePermissionDenied Code = "PERMISSION_DENIED"
	CodeUnauthenticated  Code = "UNAUTHENTICATED"
	CodeInvalidArgument  Code = "INVALID_ARGUMENT"
	CodeUnavailable      Code = "UNAVAILABLE"
	CodeInternal         Code = "INTERNAL"
)

var codeToGRPCCode = map[Code]codes.Code{
	CodeNotFound:         codes.NotFound,
	CodeAlreadyExists:    codes.AlreadyExists,
	CodePermissionDenied: codes.PermissionDenied,
	CodeUnauthenticated:  codes.Unauthenticated,
	CodeInvalidArgument:  codes.InvalidArgument,
	CodeUnavailable:      codes.Unavailable,
	CodeInternal:         codes.Internal,
}

var codeToHTTPCode = map[Code]int{
	CodeNotFound:         404,
	CodeAlreadyExists:    409,
	CodePermissionDenied: 403,
	CodeUnauthenticated:  401,
	CodeInvalidArgument:  400,
	CodeUnavailable:      503,
	CodeInternal:         500,
}

func (c Code) ToGRPCCode() codes.Code {
	if code, ok := codeToGRPCCode[c]; ok {
		return code
	}
	return codes.Internal
}

func (c Code) ToHTTPCode() int {
	if code, ok := codeToHTTPCode[c]; ok {
		return code
	}
	return 500
}

func (c Code) Error(msg string) error {
	return status.Error(c.ToGRPCCode(), msg)
}

func (c Code) Wrap(err error) error {
	if err == nil {
		return nil
	}
	if s, ok := status.FromError(err); ok && s.Code() == c.ToGRPCCode() {
		return err
	}
	return status.Error(c.ToGRPCCode(), err.Error())
}

func FromError(err error) (Code, bool) {
	if err == nil {
		return "", false
	}
	s, ok := status.FromError(err)
	if !ok {
		return CodeInternal, true
	}

	switch s.Code() {
	case codes.NotFound:
		return CodeNotFound, true
	case codes.AlreadyExists:
		return CodeAlreadyExists, true
	case codes.PermissionDenied:
		return CodePermissionDenied, true
	case codes.Unauthenticated:
		return CodeUnauthenticated, true
	case codes.InvalidArgument:
		return CodeInvalidArgument, true
	case codes.Unavailable:
		return CodeUnavailable, true
	default:
		return CodeInternal, true
	}
}
