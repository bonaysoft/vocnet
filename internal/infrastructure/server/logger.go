package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/sirupsen/logrus"
)

// InterceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func InterceptorLogger() logging.Logger {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		logger.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

func Logger() connect.UnaryInterceptorFunc {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)

			duration := time.Since(start)
			code := connect.CodeOf(err)
			level := slog.LevelInfo
			if err != nil {
				switch code {
				case connect.CodeInvalidArgument, connect.CodeFailedPrecondition, connect.CodeNotFound, connect.CodeAlreadyExists, connect.CodePermissionDenied, connect.CodeUnauthenticated:
					level = slog.LevelWarn
				default:
					level = slog.LevelError
				}
			}

			attrs := []slog.Attr{
				slog.String("procedure", req.Spec().Procedure),
				slog.String("status", code.String()),
				slog.Duration("duration", duration),
			}

			if httpMethod := req.HTTPMethod(); httpMethod != "" {
				attrs = append(attrs, slog.String("http_method", httpMethod))
			}

			if stream := req.Spec().StreamType.String(); stream != "" {
				attrs = append(attrs, slog.String("stream", stream))
			}

			attrs = append(attrs, slog.String("idempotency", req.Spec().IdempotencyLevel.String()))

			if peer := req.Peer(); peer.Addr != "" {
				attrs = append(attrs, slog.String("peer_addr", peer.Addr))
			}

			if protocol := req.Peer().Protocol; protocol != "" {
				attrs = append(attrs, slog.String("protocol", protocol))
			}

			if q := req.Peer().Query.Encode(); q != "" {
				attrs = append(attrs, slog.String("query", q))
			}

			if ua := req.Header().Get("User-Agent"); ua != "" {
				attrs = append(attrs, slog.String("user_agent", ua))
			}

			if requestID := req.Header().Get("X-Request-Id"); requestID != "" {
				attrs = append(attrs, slog.String("request_id", requestID))
			}

			if forwardedFor := firstForwardedFor(req.Header()); forwardedFor != "" {
				attrs = append(attrs, slog.String("client_ip", forwardedFor))
			}

			if contentType := req.Header().Get("Content-Type"); contentType != "" {
				attrs = append(attrs, slog.String("content_type", contentType))
			}

			if accept := req.Header().Get("Accept"); accept != "" {
				attrs = append(attrs, slog.String("accept", accept))
			}

			if encoding := req.Header().Get("Content-Encoding"); encoding != "" {
				attrs = append(attrs, slog.String("content_encoding", encoding))
			}

			if grpcEncoding := req.Header().Get("Grpc-Encoding"); grpcEncoding != "" {
				attrs = append(attrs, slog.String("grpc_encoding", grpcEncoding))
			}

			attrs = append(attrs, slog.Int("request_header_count", headerCount(req.Header())))

			if cl := contentLength(req.Header()); cl >= 0 {
				attrs = append(attrs, slog.Int("request_bytes", cl))
			}

			if resp != nil {
				if cl := contentLength(resp.Header()); cl >= 0 {
					attrs = append(attrs, slog.Int("response_bytes", cl))
				}
				if len(resp.Header()) > 0 {
					attrs = append(attrs, slog.Int("response_header_count", headerCount(resp.Header())))
				}
				if len(resp.Trailer()) > 0 {
					attrs = append(attrs, slog.Int("response_trailer_count", headerCount(resp.Trailer())))
				}
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}

			logger.LogAttrs(ctx, level, "request completed", attrs...)

			return resp, err
		}
	}
}

func firstForwardedFor(header http.Header) string {
	forwarded := header.Get("X-Forwarded-For")
	if forwarded == "" {
		return ""
	}
	for _, part := range strings.Split(forwarded, ",") {
		if candidate := strings.TrimSpace(part); candidate != "" {
			return candidate
		}
	}
	return ""
}

func headerCount(header http.Header) int {
	count := 0
	for key := range header {
		count += len(header[key])
	}
	return count
}

func contentLength(header http.Header) int {
	if header == nil {
		return -1
	}
	if cl := header.Get("Content-Length"); cl != "" {
		if parsed, err := strconv.Atoi(cl); err == nil {
			return parsed
		}
	}
	return -1
}

// NewLogger builds a configured logrus logger from application config.
func NewLogger(cfg *config.Config) (*logrus.Logger, error) {
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	logger.SetLevel(level)
	if cfg.Log.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
	return logger, nil
}
