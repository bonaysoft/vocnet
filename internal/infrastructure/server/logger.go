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
			level := determineLogLevel(code, err)
			attrs := buildLogAttributes(req, resp, code, duration, err)

			logger.LogAttrs(ctx, level, "request completed", attrs...)

			return resp, err
		}
	}
}

func determineLogLevel(code connect.Code, err error) slog.Level {
	if err == nil {
		return slog.LevelInfo
	}
	switch code {
	case connect.CodeInvalidArgument, connect.CodeFailedPrecondition, connect.CodeNotFound,
		connect.CodeAlreadyExists, connect.CodePermissionDenied, connect.CodeUnauthenticated:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func buildLogAttributes(req connect.AnyRequest, resp connect.AnyResponse, code connect.Code, duration time.Duration, err error) []slog.Attr {
	attrs := requestAttributes(req, code, duration)
	attrs = append(attrs, responseAttributes(resp)...)
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	return attrs
}

func requestAttributes(req connect.AnyRequest, code connect.Code, duration time.Duration) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("procedure", req.Spec().Procedure),
		slog.String("status", code.String()),
		slog.Duration("duration", duration),
	}

	appendStringAttr(&attrs, "http_method", req.HTTPMethod())
	appendStringAttr(&attrs, "stream", req.Spec().StreamType.String())
	appendStringAttr(&attrs, "idempotency", req.Spec().IdempotencyLevel.String())

	peer := req.Peer()
	appendStringAttr(&attrs, "peer_addr", peer.Addr)
	appendStringAttr(&attrs, "protocol", peer.Protocol)
	appendStringAttr(&attrs, "query", peer.Query.Encode())

	header := req.Header()
	appendStringAttr(&attrs, "user_agent", header.Get("User-Agent"))
	appendStringAttr(&attrs, "request_id", header.Get("X-Request-Id"))
	appendStringAttr(&attrs, "client_ip", firstForwardedFor(header))
	appendStringAttr(&attrs, "content_type", header.Get("Content-Type"))
	appendStringAttr(&attrs, "accept", header.Get("Accept"))
	appendStringAttr(&attrs, "content_encoding", header.Get("Content-Encoding"))
	appendStringAttr(&attrs, "grpc_encoding", header.Get("Grpc-Encoding"))

	attrs = append(attrs, slog.Int("request_header_count", headerCount(header)))
	if cl := contentLength(header); cl >= 0 {
		attrs = append(attrs, slog.Int("request_bytes", cl))
	}

	return attrs
}

func responseAttributes(resp connect.AnyResponse) []slog.Attr {
	if resp == nil {
		return nil
	}

	attrs := make([]slog.Attr, 0, 3)
	if cl := contentLength(resp.Header()); cl >= 0 {
		attrs = append(attrs, slog.Int("response_bytes", cl))
	}
	if len(resp.Header()) > 0 {
		attrs = append(attrs, slog.Int("response_header_count", headerCount(resp.Header())))
	}
	if len(resp.Trailer()) > 0 {
		attrs = append(attrs, slog.Int("response_trailer_count", headerCount(resp.Trailer())))
	}
	return attrs
}

func appendStringAttr(attrs *[]slog.Attr, key, value string) {
	if value == "" {
		return
	}
	*attrs = append(*attrs, slog.String(key, value))
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
