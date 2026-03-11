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
	return codeToGRPCCode[c]
}

func (c Code) ToHTTPCode() int {
	return codeToHTTPCode[c]
}

func (c Code) Error(msg string) error {
	return status.Error(c.ToGRPCCode(), msg)
}

func (c Code) Wrap(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(c.ToGRPCCode(), err.Error())
}

func FromError(err error) Code {
	if err == nil {
		return ""
	}
	s, ok := status.FromError(err)
	if !ok {
		return CodeInternal
	}

	switch s.Code() {
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists:
		return CodeAlreadyExists
	case codes.PermissionDenied:
		return CodePermissionDenied
	case codes.Unauthenticated:
		return CodeUnauthenticated
	case codes.InvalidArgument:
		return CodeInvalidArgument
	case codes.Unavailable:
		return CodeUnavailable
	default:
		return CodeInternal
	}
}
