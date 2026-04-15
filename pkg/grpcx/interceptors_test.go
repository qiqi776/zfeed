package grpcx

import (
	"context"
	"fmt"
	"net"
	"testing"

	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestRequestIDServerInterceptorGeneratesRequestID(t *testing.T) {
	resp, err := RequestIDServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			t.Fatal("expected metadata in context")
		}
		requestID := firstMetadataValue(md, requestIDMetadataKey)
		if requestID == "" {
			t.Fatal("expected generated request id")
		}
		return requestID, nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp == "" {
		t.Fatal("expected response to contain request id")
	}
}

func TestRequestIDServerInterceptorAddsStructuredFieldsToContext(t *testing.T) {
	writer, restore := useCaptureWriter()
	defer restore()

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5001},
	})
	_, err := RequestIDServerInterceptor(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: "/zfeed.search.SearchService/SearchUsers",
	}, func(ctx context.Context, req any) (any, error) {
		logx.WithContext(ctx).Infow("handler log")
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	entry := writer.lastEntry()
	if entry.level != "info" {
		t.Fatalf("log level = %q, want info", entry.level)
	}
	if entry.message != "handler log" {
		t.Fatalf("log message = %q, want handler log", entry.message)
	}
	assertLogFieldNotEmpty(t, entry, "request_id")
	assertLogFieldEquals(t, entry, "rpc_system", rpcSystemGRPC)
	assertLogFieldEquals(t, entry, "rpc_kind", rpcKindServer)
	assertLogFieldEquals(t, entry, "grpc_full_method", "/zfeed.search.SearchService/SearchUsers")
	assertLogFieldEquals(t, entry, "grpc_service", "zfeed.search.SearchService")
	assertLogFieldEquals(t, entry, "grpc_method", "SearchUsers")
	assertLogFieldEquals(t, entry, "peer_addr", "127.0.0.1:5001")
}

func TestStructuredLoggingServerInterceptorLogsDurationAndCode(t *testing.T) {
	writer, restore := useCaptureWriter()
	defer restore()

	info := &grpc.UnaryServerInfo{FullMethod: "/zfeed.user.UserService/GetMe"}
	_, err := RequestIDServerInterceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return StructuredLoggingServerInterceptor(ctx, req, info, func(context.Context, any) (any, error) {
			return "ok", nil
		})
	})
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	entry := writer.lastEntry()
	if entry.level != "info" {
		t.Fatalf("log level = %q, want info", entry.level)
	}
	if entry.message != "grpc server request handled" {
		t.Fatalf("log message = %q, want grpc server request handled", entry.message)
	}
	assertLogFieldEquals(t, entry, "grpc_code", codes.OK.String())
	assertLogFieldNotEmpty(t, entry, "duration")
	assertLogFieldEquals(t, entry, "grpc_service", "zfeed.user.UserService")
	assertLogFieldEquals(t, entry, "grpc_method", "GetMe")
}

func TestChainUnaryClientInterceptorsAddsRequestIDAndLogs(t *testing.T) {
	writer, restore := useCaptureWriter()
	defer restore()

	interceptor := ChainUnaryClientInterceptors(RequestIDClientInterceptor, StructuredLoggingClientInterceptor)
	err := interceptor(context.Background(), "/zfeed.search.SearchService/SearchUsers", nil, nil, nil, func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata in context")
		}
		if requestID := firstMetadataValue(md, requestIDMetadataKey); requestID == "" {
			t.Fatal("expected generated request id")
		}
		return status.Error(codes.NotFound, "missing")
	})
	if err == nil {
		t.Fatal("expected grpc error")
	}

	entry := writer.lastEntry()
	if entry.level != "error" {
		t.Fatalf("log level = %q, want error", entry.level)
	}
	if entry.message != "grpc client call failed" {
		t.Fatalf("log message = %q, want grpc client call failed", entry.message)
	}
	assertLogFieldNotEmpty(t, entry, "request_id")
	assertLogFieldEquals(t, entry, "rpc_kind", rpcKindClient)
	assertLogFieldEquals(t, entry, "grpc_service", "zfeed.search.SearchService")
	assertLogFieldEquals(t, entry, "grpc_method", "SearchUsers")
	assertLogFieldEquals(t, entry, "grpc_code", codes.NotFound.String())
	assertLogFieldEquals(t, entry, "grpc_message", "missing")
	assertLogFieldNotEmpty(t, entry, "duration")
}

func TestErrorUnaryServerInterceptorMapsBizError(t *testing.T) {
	_, err := ErrorUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		return nil, errorx.NewNotFound("not found")
	})
	if err == nil {
		t.Fatal("expected grpc error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status, got %T", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("grpc code = %s, want %s", st.Code(), codes.NotFound)
	}
	if st.Message() != "not found" {
		t.Fatalf("grpc message = %q, want %q", st.Message(), "not found")
	}
}

func TestRecoveryUnaryServerInterceptorMapsPanic(t *testing.T) {
	_, err := RecoveryUnaryServerInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"}, func(context.Context, any) (any, error) {
		panic("boom")
	})
	if err == nil {
		t.Fatal("expected grpc error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status, got %T", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("grpc code = %s, want %s", st.Code(), codes.Internal)
	}
	if st.Message() != errorx.DefaultErrorMessage {
		t.Fatalf("grpc message = %q, want %q", st.Message(), errorx.DefaultErrorMessage)
	}
}

type captureWriter struct {
	entries []capturedLogEntry
}

type capturedLogEntry struct {
	level   string
	message string
	fields  map[string]any
}

func (w *captureWriter) Alert(any)                  {}
func (w *captureWriter) Close() error               { return nil }
func (w *captureWriter) Severe(any)                 {}
func (w *captureWriter) Stack(any)                  {}
func (w *captureWriter) Stat(any, ...logx.LogField) {}

func (w *captureWriter) Debug(v any, fields ...logx.LogField) {
	w.entries = append(w.entries, newCapturedLogEntry("debug", v, fields))
}

func (w *captureWriter) Error(v any, fields ...logx.LogField) {
	w.entries = append(w.entries, newCapturedLogEntry("error", v, fields))
}

func (w *captureWriter) Info(v any, fields ...logx.LogField) {
	w.entries = append(w.entries, newCapturedLogEntry("info", v, fields))
}

func (w *captureWriter) Slow(v any, fields ...logx.LogField) {
	w.entries = append(w.entries, newCapturedLogEntry("slow", v, fields))
}

func (w *captureWriter) lastEntry() capturedLogEntry {
	if len(w.entries) == 0 {
		return capturedLogEntry{}
	}
	return w.entries[len(w.entries)-1]
}

func newCapturedLogEntry(level string, v any, fields []logx.LogField) capturedLogEntry {
	entry := capturedLogEntry{
		level:   level,
		message: fmt.Sprint(v),
		fields:  make(map[string]any, len(fields)),
	}
	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		entry.fields[field.Key] = field.Value
	}
	return entry
}

func useCaptureWriter() (*captureWriter, func()) {
	prev := logx.Reset()
	writer := &captureWriter{}
	logx.SetWriter(writer)
	return writer, func() {
		logx.SetWriter(prev)
	}
}

func assertLogFieldEquals(t *testing.T, entry capturedLogEntry, key, want string) {
	t.Helper()
	got, ok := entry.fields[key]
	if !ok {
		t.Fatalf("expected log field %q to exist", key)
	}
	if got != want {
		t.Fatalf("log field %q = %v, want %q", key, got, want)
	}
}

func assertLogFieldNotEmpty(t *testing.T, entry capturedLogEntry, key string) {
	t.Helper()
	got, ok := entry.fields[key]
	if !ok {
		t.Fatalf("expected log field %q to exist", key)
	}
	if got == "" {
		t.Fatalf("log field %q should not be empty", key)
	}
}
