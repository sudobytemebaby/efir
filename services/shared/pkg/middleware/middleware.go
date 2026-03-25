// Package middleware provides gRPC interceptors for logging, recovery, and user ID extraction.
package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	MetadataKeyUserID    = "x-user-id"
	MetadataKeyRequestID = "x-request-id"
)

func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		requestIDs := md.Get(MetadataKeyRequestID)
		var requestID string
		if len(requestIDs) > 0 && requestIDs[0] != "" {
			requestID = requestIDs[0]
		} else {
			requestID = uuid.New().String()
		}

		ctx = context.WithValue(ctx, contextKeyRequestID{}, requestID)
		return handler(ctx, req)
	}
}

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		requestID, _ := GetRequestID(ctx)

		logger.Info("gRPC request started",
			"method", info.FullMethod,
			"request_id", requestID,
		)

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
		}

		logger.Log(ctx, level, "gRPC request completed",
			"method", info.FullMethod,
			"request_id", requestID,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)

		return resp, err
	}
}

func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			if r := recover(); r != nil {
				requestID, _ := GetRequestID(ctx)
				logger.Error("panic recovered",
					"method", info.FullMethod,
					"request_id", requestID,
					"panic", r,
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func UserIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		userIDs := md.Get(MetadataKeyUserID)
		if len(userIDs) > 0 {
			ctx = context.WithValue(ctx, contextKeyUserID{}, userIDs[0])
		}

		return handler(ctx, req)
	}
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKeyUserID{}).(string)
	return userID, ok
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(contextKeyRequestID{}).(string)
	return requestID, ok
}

type contextKeyUserID struct{}
type contextKeyRequestID struct{}
