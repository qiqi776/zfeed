package errorx

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestBizErrorGRPCStatus(t *testing.T) {
	err := NewConflict("手机号已注册")

	if got := err.HTTPStatus(); got != http.StatusConflict {
		t.Fatalf("expected http status %d, got %d", http.StatusConflict, got)
	}

	st := err.GRPCStatus()
	if st.Code() != codes.AlreadyExists {
		t.Fatalf("expected grpc code %s, got %s", codes.AlreadyExists, st.Code())
	}
	if st.Message() != "手机号已注册" {
		t.Fatalf("expected grpc message to be preserved, got %q", st.Message())
	}
}

func TestResponseFromError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "biz error",
			err:        NewUnauthorized("密码错误"),
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "密码错误",
		},
		{
			name:       "grpc error",
			err:        status.Error(codes.AlreadyExists, "手机号已注册"),
			wantStatus: http.StatusConflict,
			wantMsg:    "手机号已注册",
		},
		{
			name:       "plain error",
			err:        errors.New("bad request body"),
			wantStatus: http.StatusBadRequest,
			wantMsg:    "bad request body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, body := ResponseFromError(tc.err)
			if statusCode != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, statusCode)
			}

			resp, ok := body.(Response)
			if !ok {
				t.Fatalf("expected Response body, got %T", body)
			}
			if resp.Message != tc.wantMsg {
				t.Fatalf("expected message %q, got %q", tc.wantMsg, resp.Message)
			}
			if resp.Code != uint32(tc.wantStatus) {
				t.Fatalf("expected body code %d, got %d", tc.wantStatus, resp.Code)
			}
		})
	}
}
