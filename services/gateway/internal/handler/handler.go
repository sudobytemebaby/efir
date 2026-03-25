package handler

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxBodySize = 1 << 20 // 1 MB

var marshaler = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

var unmarshaler = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

func WriteProto(w http.ResponseWriter, status int, msg proto.Message) {
	b, err := marshaler.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal proto response", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func ReadProto(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		if stderrors.Is(err, io.EOF) && len(body) >= maxBodySize {
			return errors.CodeInvalidArgument.Error("request body too large")
		}
		return err
	}
	return unmarshaler.Unmarshal(body, msg)
}

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
	if encErr := json.NewEncoder(w).Encode(errorResponse{
		Error: codeToMessage[code],
		Code:  string(code),
	}); encErr != nil {
		slog.ErrorContext(r.Context(), "failed to encode error response", "error", encErr)
	}
}
