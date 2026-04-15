package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/user/client/userservice"
	userpb "zfeed/app/rpc/user/user"
)

func TestGetMeLoadsCountsFromCountRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(3001))
	logic := NewGetMeLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			me: &userservice.GetMeRes{
				UserInfo: &userservice.UserInfo{
					UserId:   3001,
					Mobile:   "13800000000",
					Nickname: "me",
					Avatar:   "https://example.com/avatar.png",
					Bio:      "bio",
					Gender:   userpb.Gender_GENDER_MALE,
					Status:   userpb.UserStatus_USER_STATUS_ACTIVE,
				},
			},
		},
		CountRpc: &stubCounterService{
			counts: &counterservice.GetUserProfileCountsRes{
				FollowingCount: 5,
				FollowedCount:  6,
				LikeCount:      7,
				FavoriteCount:  8,
				ContentCount:   9,
			},
		},
	})

	resp, err := logic.GetMe()
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if resp.UserInfo.UserId != 3001 || resp.UserInfo.Nickname != "me" {
		t.Fatalf("unexpected user info: %+v", resp.UserInfo)
	}
	if resp.FolloweeCount != 5 || resp.FollowerCount != 6 || resp.LikeReceivedCount != 7 || resp.FavoriteReceivedCount != 8 || resp.ContentCount != 9 {
		t.Fatalf("unexpected count fields: %+v", resp)
	}
}

func TestGetMeLoadsProfileExtraFields(t *testing.T) {
	birthday := time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)

	ctx := context.WithValue(context.Background(), "user_id", int64(3010))
	logic := NewGetMeLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			me: &userservice.GetMeRes{
				UserInfo: &userservice.UserInfo{
					UserId:   3010,
					Mobile:   "13800000000",
					Nickname: "me",
					Avatar:   "https://example.com/avatar.png",
					Email:    "me@example.com",
					Birthday: birthday.Unix(),
				},
			},
		},
		CountRpc: &stubCounterService{
			counts: &counterservice.GetUserProfileCountsRes{},
		},
	})

	resp, err := logic.GetMe()
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if resp.UserInfo.Email != "me@example.com" {
		t.Fatalf("email = %q, want %q", resp.UserInfo.Email, "me@example.com")
	}
	if resp.UserInfo.Birthday != birthday.Unix() {
		t.Fatalf("birthday = %d, want %d", resp.UserInfo.Birthday, birthday.Unix())
	}
}

func TestGetMeDegradesWhenCountRPCFails(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(3002))
	logic := NewGetMeLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			me: &userservice.GetMeRes{
				UserInfo: &userservice.UserInfo{
					UserId:   3002,
					Nickname: "degrade",
				},
			},
		},
		CountRpc: &stubCounterService{
			err: errors.New("count rpc failed"),
		},
	})

	resp, err := logic.GetMe()
	if err != nil {
		t.Fatalf("get me with count degrade: %v", err)
	}
	if resp.UserInfo.UserId != 3002 || resp.UserInfo.Nickname != "degrade" {
		t.Fatalf("unexpected user info: %+v", resp.UserInfo)
	}
	if resp.FolloweeCount != 0 || resp.FollowerCount != 0 || resp.LikeReceivedCount != 0 || resp.FavoriteReceivedCount != 0 {
		t.Fatalf("count fields should degrade to zero: %+v", resp)
	}
}

func TestGetMeDegradesWhenCountRPCTimeouts(t *testing.T) {
	oldTimeout := defaultCountRPCTimeout
	defaultCountRPCTimeout = 10 * time.Millisecond
	defer func() {
		defaultCountRPCTimeout = oldTimeout
	}()

	ctx := context.WithValue(context.Background(), "user_id", int64(3003))
	logic := NewGetMeLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			me: &userservice.GetMeRes{
				UserInfo: &userservice.UserInfo{
					UserId:   3003,
					Nickname: "timeout",
				},
			},
		},
		CountRpc: &stubCounterService{
			delay: 50 * time.Millisecond,
			counts: &counterservice.GetUserProfileCountsRes{
				FollowingCount: 99,
			},
		},
	})

	resp, err := logic.GetMe()
	if err != nil {
		t.Fatalf("get me with count timeout degrade: %v", err)
	}
	if resp.UserInfo.UserId != 3003 {
		t.Fatalf("unexpected user info: %+v", resp.UserInfo)
	}
	if resp.FolloweeCount != 0 || resp.FollowerCount != 0 || resp.LikeReceivedCount != 0 || resp.FavoriteReceivedCount != 0 {
		t.Fatalf("count fields should degrade to zero after timeout: %+v", resp)
	}
}

func TestGetMeFailsWhenUserRPCFails(t *testing.T) {
	ctx := context.WithValue(context.Background(), "user_id", int64(3004))
	logic := NewGetMeLogic(ctx, &svc.ServiceContext{
		UserRpc: &stubUserService{
			err: errors.New("user rpc failed"),
		},
		CountRpc: &stubCounterService{
			counts: &counterservice.GetUserProfileCountsRes{
				FollowingCount: 1,
			},
		},
	})

	if _, err := logic.GetMe(); err == nil {
		t.Fatal("expected user rpc failure")
	}
}

func TestGetMeFailsWithoutUserIDInContext(t *testing.T) {
	logic := NewGetMeLogic(context.Background(), &svc.ServiceContext{})
	if _, err := logic.GetMe(); err == nil {
		t.Fatal("expected missing user id error")
	}
}
