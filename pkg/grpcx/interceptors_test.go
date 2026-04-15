package grpcx

import (
	"context"
	"testing"

	"zfeed/pkg/errorx"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
