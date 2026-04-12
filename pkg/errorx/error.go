package errorx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DefaultErrorCode    uint32 = 500
	DefaultErrorMessage string = "服务内部错误"
)

type BizError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

func New(message string, code uint32) *BizError {
	statusCode := normalizeHTTPStatus(code)
	return &BizError{
		Code:    uint32(statusCode),
		Message: message,
	}
}

func NewMsg(message string) *BizError {
	return New(message, DefaultErrorCode)
}

func NewBadRequest(message string) *BizError {
	return New(message, http.StatusBadRequest)
}

func NewUnauthorized(message string) *BizError {
	return New(message, http.StatusUnauthorized)
}

func NewForbidden(message string) *BizError {
	return New(message, http.StatusForbidden)
}

func NewNotFound(message string) *BizError {
	return New(message, http.StatusNotFound)
}

func NewConflict(message string) *BizError {
	return New(message, http.StatusConflict)
}

func NewInternal(message string) *BizError {
	return New(message, DefaultErrorCode)
}

func (e *BizError) Error() string {
	return e.Message
}

func (e *BizError) HTTPStatus() int {
	if e == nil {
		return http.StatusInternalServerError
	}

	return normalizeHTTPStatus(e.Code)
}

func (e *BizError) GRPCStatus() *status.Status {
	if e == nil {
		return status.New(codes.Internal, DefaultErrorMessage)
	}

	return status.New(grpcCodeFromHTTPStatus(e.HTTPStatus()), e.Message)
}

func Wrap(ctx context.Context, sysErr error, bizErr *BizError) *BizError {
	if sysErr == nil {
		return bizErr
	}

	stack := captureStack(2)
	logx.WithContext(ctx).WithCallerSkip(1).Errorf(
		"%s: %v\nStack:\n%s",
		bizErr.Message,
		sysErr,
		formatStack(stack),
	)

	return bizErr
}

func captureStack(skip int) []uintptr {
	const depth = 32
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip, pcs)
	return pcs[:n]
}

func formatStack(pcs []uintptr) string {
	var buf bytes.Buffer
	frames := runtime.CallersFrames(pcs)
	for {
		f, more := frames.Next()
		buf.WriteString(fmt.Sprintf("%s:%d %s\n", f.File, f.Line, f.Function))
		if !more {
			break
		}
	}
	return buf.String()
}

func ResponseFromError(err error) (int, any) {
	if err == nil {
		return http.StatusOK, nil
	}

	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr.HTTPStatus(), Response{
			Code:    bizErr.Code,
			Message: bizErr.Message,
		}
	}

	if st, ok := status.FromError(err); ok {
		statusCode := httpStatusFromGRPCCode(st.Code())
		return statusCode, Response{
			Code:    uint32(statusCode),
			Message: st.Message(),
		}
	}

	return http.StatusBadRequest, Response{
		Code:    http.StatusBadRequest,
		Message: err.Error(),
	}
}

func normalizeHTTPStatus(code uint32) int {
	statusCode := int(code)
	if statusCode < http.StatusContinue || statusCode > 599 {
		return http.StatusInternalServerError
	}

	return statusCode
}

func grpcCodeFromHTTPStatus(statusCode int) codes.Code {
	switch statusCode {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

func httpStatusFromGRPCCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
