package grpcx

import (
	"context"
	"errors"
	"runtime/debug"
	"strings"
	"time"

	"zfeed/pkg/errorx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const requestIDMetadataKey = "x-request-id"

const (
	rpcKindClient = "client"
	rpcKindServer = "server"
	rpcSystemGRPC = "grpc"
)

type rpcMethodInfo struct {
	FullMethod string
	Service    string
	Method     string
}

type unaryInterceptorAdder interface {
	AddUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor)
}

func ClientInterceptorOption() zrpc.ClientOption {
	return zrpc.WithUnaryClientInterceptor(ChainUnaryClientInterceptors(
		RequestIDClientInterceptor,
		StructuredLoggingClientInterceptor,
	))
}

func InstallServerInterceptors(server unaryInterceptorAdder) {
	server.AddUnaryInterceptors(
		RequestIDServerInterceptor,
		StructuredLoggingServerInterceptor,
		RecoveryUnaryServerInterceptor,
		ErrorUnaryServerInterceptor,
	)
}

func ChainUnaryClientInterceptors(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		chainedInvoker := invoker
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chainedInvoker
			chainedInvoker = func(current grpc.UnaryClientInterceptor, currentNext grpc.UnaryInvoker) grpc.UnaryInvoker {
				return func(currentCtx context.Context, currentMethod string, currentReq, currentReply any, currentCC *grpc.ClientConn, currentOpts ...grpc.CallOption) error {
					return current(currentCtx, currentMethod, currentReq, currentReply, currentCC, currentNext, currentOpts...)
				}
			}(interceptor, next)
		}

		return chainedInvoker(ctx, method, req, reply, cc, opts...)
	}
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

func StructuredLoggingClientInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	startedAt := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)

	methodInfo := parseRPCMethod(method)
	requestID := requestIDFromOutgoingContext(ctx)
	logCtx := withRPCLoggingContext(ctx, rpcKindClient, requestID, methodInfo, grpcTargetField(cc))
	fields := []logx.LogField{
		logx.Field("grpc_code", status.Code(err).String()),
	}
	logger := logx.WithContext(logCtx).WithDuration(time.Since(startedAt))
	if err != nil {
		st := status.Convert(err)
		fields = append(fields, logx.Field("grpc_message", st.Message()))
		logger.Errorw("grpc client call failed", fields...)
		return err
	}

	logger.Infow("grpc client call handled", fields...)
	return nil
}

func RequestIDServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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
	ctx = withRPCLoggingContext(ctx, rpcKindServer, requestID, parseUnaryServerInfo(info), grpcPeerField(ctx))

	return handler(ctx, req)
}

func StructuredLoggingServerInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	startedAt := time.Now()
	resp, err := handler(ctx, req)

	fields := []logx.LogField{
		logx.Field("grpc_code", status.Code(err).String()),
	}
	logger := logx.WithContext(ctx).WithDuration(time.Since(startedAt))
	if err != nil {
		st := status.Convert(err)
		fields = append(fields, logx.Field("grpc_message", st.Message()))
		logger.Errorw("grpc server request failed", fields...)
		return resp, err
	}

	logger.Infow("grpc server request handled", fields...)
	return resp, nil
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

func requestIDFromOutgoingContext(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}
	return firstMetadataValue(md, requestIDMetadataKey)
}

func parseUnaryServerInfo(info *grpc.UnaryServerInfo) rpcMethodInfo {
	if info == nil {
		return rpcMethodInfo{}
	}
	return parseRPCMethod(info.FullMethod)
}

func parseRPCMethod(fullMethod string) rpcMethodInfo {
	info := rpcMethodInfo{FullMethod: fullMethod}
	trimmed := strings.TrimPrefix(fullMethod, "/")
	if trimmed == "" {
		return info
	}

	parts := strings.SplitN(trimmed, "/", 2)
	info.Service = parts[0]
	if len(parts) == 2 {
		info.Method = parts[1]
	}

	return info
}

func withRPCLoggingContext(ctx context.Context, rpcKind, requestID string, methodInfo rpcMethodInfo, extraFields ...logx.LogField) context.Context {
	fields := make([]logx.LogField, 0, 6+len(extraFields))
	if requestID != "" {
		fields = append(fields, logx.Field("request_id", requestID))
	}
	fields = append(fields,
		logx.Field("rpc_system", rpcSystemGRPC),
		logx.Field("rpc_kind", rpcKind),
	)
	if methodInfo.FullMethod != "" {
		fields = append(fields, logx.Field("grpc_full_method", methodInfo.FullMethod))
	}
	if methodInfo.Service != "" {
		fields = append(fields, logx.Field("grpc_service", methodInfo.Service))
	}
	if methodInfo.Method != "" {
		fields = append(fields, logx.Field("grpc_method", methodInfo.Method))
	}
	for _, field := range extraFields {
		if field.Key == "" {
			continue
		}
		fields = append(fields, field)
	}
	return logx.ContextWithFields(ctx, fields...)
}

func grpcPeerField(ctx context.Context) logx.LogField {
	pr, ok := peer.FromContext(ctx)
	if !ok || pr == nil || pr.Addr == nil {
		return logx.LogField{}
	}
	return logx.Field("peer_addr", pr.Addr.String())
}

func grpcTargetField(cc *grpc.ClientConn) logx.LogField {
	if cc == nil || cc.Target() == "" {
		return logx.LogField{}
	}
	return logx.Field("grpc_target", cc.Target())
}
