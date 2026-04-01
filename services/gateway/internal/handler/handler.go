package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/sudobytemebaby/efir/services/shared/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var maxBodySize int64 = 1 << 20

func SetMaxBodySize(n int64) {
	if n > 0 {
		maxBodySize = n
	}
}

var marshaler = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

var unmarshaler = protojson.UnmarshalOptions{
	DiscardUnknown: true,
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
	errors.CodeRateLimited:      "rate limit exceeded",
}

func WriteCode(w http.ResponseWriter, code errors.Code) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code.ToHTTPCode())
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: codeToMessage[code],
		Code:  string(code),
	})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, r *http.Request, err error, msg string) {
	slog.ErrorContext(r.Context(), msg, "error", err)

	code, ok := errors.FromError(err)
	if !ok {
		code = errors.CodeInternal
	}

	WriteCode(w, code)
}

func WriteProto(w http.ResponseWriter, status int, msg proto.Message) {
	b, err := marshaler.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal proto response", "error", err)
		WriteCode(w, errors.CodeInternal)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func ReadProto(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > maxBodySize {
		return errors.CodeInvalidArgument.Error("request body too large")
	}
	if err := unmarshaler.Unmarshal(body, msg); err != nil {
		return errors.CodeInvalidArgument.Wrap(err)
	}
	return nil
}
