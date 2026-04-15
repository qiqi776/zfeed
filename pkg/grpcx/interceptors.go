package grpcx

import (
	"context"
	"errors"
	"runtime/debug"

	"zfeed/pkg/errorx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDMetadataKey = "x-request-id"

type unaryInterceptorAdder interface {
	AddUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor)
}

func ClientInterceptorOption() zrpc.ClientOption {
	return zrpc.WithUnaryClientInterceptor(RequestIDClientInterceptor)
}

func InstallServerInterceptors(server unaryInterceptorAdder) {
	server.AddUnaryInterceptors(RequestIDServerInterceptor, RecoveryUnaryServerInterceptor, ErrorUnaryServerInterceptor)
}

func RequestIDClientInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}
	if len(md.Get(requestIDMetadataKey)) == 0 {
		md.Set(requestIDMetadataKey, uuid.NewString())
	}

	return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
}

func RequestIDServerInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	} else {
		md = md.Copy()
	}

	requestID := firstMetadataValue(md, requestIDMetadataKey)
	if requestID == "" {
		requestID = uuid.NewString()
		md.Set(requestIDMetadataKey, requestID)
	}
	ctx = metadata.NewIncomingContext(ctx, md)
	_ = grpc.SetHeader(ctx, metadata.Pairs(requestIDMetadataKey, requestID))

	return handler(ctx, req)
}

func ErrorUnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	if err == nil {
		return resp, nil
	}

	var bizErr *errorx.BizError
	if errors.As(err, &bizErr) {
		return resp, bizErr.GRPCStatus().Err()
	}
	if _, ok := status.FromError(err); ok {
		return resp, err
	}

	return resp, errorx.NewInternal(errorx.DefaultErrorMessage).GRPCStatus().Err()
}

func RecoveryUnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		method := ""
		if info != nil {
			method = info.FullMethod
		}
		logx.WithContext(ctx).Errorf("grpc panic recovered, method=%s, panic=%v, stack=%s", method, recovered, debug.Stack())
		err = errorx.NewInternal(errorx.DefaultErrorMessage).GRPCStatus().Err()
		resp = nil
	}()

	return handler(ctx, req)
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
