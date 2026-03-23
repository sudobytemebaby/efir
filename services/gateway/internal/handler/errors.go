package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

var codeToMessage = map[errors.Code]string{
	errors.CodeNotFound:         "resource not found",
	errors.CodeAlreadyExists:    "resource already exists",
	errors.CodePermissionDenied: "permission denied",
	errors.CodeUnauthenticated:  "authentication required",
	errors.CodeInvalidArgument:  "invalid request",
	errors.CodeUnavailable:      "service temporarily unavailable",
	errors.CodeInternal:         "internal server error",
}

func WriteError(w http.ResponseWriter, r *http.Request, err error, msg string) {
	slog.ErrorContext(r.Context(), msg, "error", err)

	code, ok := errors.FromError(err)
	if !ok {
		code = errors.CodeInternal
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code.ToHTTPCode())
	if encodeErr := json.NewEncoder(w).Encode(errorResponse{
		Error: codeToMessage[code],
		Code:  string(code),
	}); encodeErr != nil {
		slog.ErrorContext(r.Context(), "failed to encode error response", "error", encodeErr)
	}
}
