package logic

import (
	"testing"
	"time"
)

func TestResolveRegisterNicknameFallsBackToMobile(t *testing.T) {
	got := resolveRegisterNickname("+8613800000000", " ")
	if got != "+8613800000000" {
		t.Fatalf("nickname = %q, want %q", got, "+8613800000000")
	}
}

func TestResolveRegisterEmailFallsBackToGeneratedLocalAddress(t *testing.T) {
	got := resolveRegisterEmail("+8613800000000", "")
	want := "register-8613800000000@zfeed.local"
	if got != want {
		t.Fatalf("email = %q, want %q", got, want)
	}
}

func TestResolveRegisterBirthdayAllowsMissingValue(t *testing.T) {
	if got := resolveRegisterBirthday(0); got != nil {
		t.Fatalf("birthday = %v, want nil", got)
	}

	unix := time.Date(2000, 5, 6, 12, 0, 0, 0, time.UTC).Unix()
	got := resolveRegisterBirthday(unix)
	if got == nil {
		t.Fatal("expected birthday to be normalized")
	}
	if got.Year() != 2000 || got.Month() != time.May || got.Day() != 6 {
		t.Fatalf("birthday = %v, want 2000-05-06", got)
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Fatalf("birthday time should be truncated, got %v", got)
	}
}
