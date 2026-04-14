package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/user/client/userservice"
	userpb "zfeed/app/rpc/user/user"
)

func TestQueryUserProfileLoadsCountsFromCountRPC(t *testing.T) {
	logic := NewQueryUserProfileLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{
			profile: &userservice.GetUserProfileRes{
				UserProfile: &userservice.UserProfile{
					UserId:   2001,
					Nickname: "alice",
					Avatar:   "https://example.com/avatar.png",
					Bio:      "hello",
					Gender:   userpb.Gender_GENDER_FEMALE,
				},
			},
		},
		CountRpc: &stubCounterService{
			counts: &counterservice.GetUserProfileCountsRes{
				FollowingCount: 11,
				FollowedCount:  22,
				LikeCount:      33,
				FavoriteCount:  44,
				ContentCount:   55,
			},
		},
	})

	resp, err := logic.QueryUserProfile(&types.QueryUserProfileReq{UserId: 2001})
	if err != nil {
		t.Fatalf("query user profile: %v", err)
	}

	if resp.UserProfileInfo.UserId != 2001 || resp.UserProfileInfo.Nickname != "alice" {
		t.Fatalf("unexpected user profile info: %+v", resp.UserProfileInfo)
	}
	if resp.UserProfileCounts.FolloweeCount != 11 ||
		resp.UserProfileCounts.FollowerCount != 22 ||
		resp.UserProfileCounts.LikeReceivedCount != 33 ||
		resp.UserProfileCounts.FavoriteReceivedCount != 44 ||
		resp.UserProfileCounts.ContentCount != 55 {
		t.Fatalf("unexpected user profile counts: %+v", resp.UserProfileCounts)
	}
}

func TestQueryUserProfileDegradesWhenCountRPCFails(t *testing.T) {
	logic := NewQueryUserProfileLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{
			profile: &userservice.GetUserProfileRes{
				UserProfile: &userservice.UserProfile{
					UserId:   2002,
					Nickname: "bob",
				},
			},
		},
		CountRpc: &stubCounterService{
			err: errors.New("count rpc unavailable"),
		},
	})

	resp, err := logic.QueryUserProfile(&types.QueryUserProfileReq{UserId: 2002})
	if err != nil {
		t.Fatalf("query user profile with count degrade: %v", err)
	}
	if resp.UserProfileInfo.UserId != 2002 || resp.UserProfileInfo.Nickname != "bob" {
		t.Fatalf("unexpected user profile info: %+v", resp.UserProfileInfo)
	}
	if resp.UserProfileCounts != (types.UserProfileCounts{}) {
		t.Fatalf("user profile counts should degrade to zero values, got %+v", resp.UserProfileCounts)
	}
}

func TestQueryUserProfileDegradesWhenCountRPCTimeouts(t *testing.T) {
	oldTimeout := defaultCountRPCTimeout
	defaultCountRPCTimeout = 10 * time.Millisecond
	defer func() {
		defaultCountRPCTimeout = oldTimeout
	}()

	logic := NewQueryUserProfileLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{
			profile: &userservice.GetUserProfileRes{
				UserProfile: &userservice.UserProfile{
					UserId:   2003,
					Nickname: "carol",
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

	resp, err := logic.QueryUserProfile(&types.QueryUserProfileReq{UserId: 2003})
	if err != nil {
		t.Fatalf("query user profile with count timeout degrade: %v", err)
	}
	if resp.UserProfileInfo.UserId != 2003 || resp.UserProfileCounts != (types.UserProfileCounts{}) {
		t.Fatalf("unexpected degraded response: %+v", resp)
	}
}

func TestQueryUserProfileFailsWhenUserRPCFails(t *testing.T) {
	logic := NewQueryUserProfileLogic(context.Background(), &svc.ServiceContext{
		UserRpc: &stubUserService{
			err: errors.New("user rpc failed"),
		},
		CountRpc: &stubCounterService{
			counts: &counterservice.GetUserProfileCountsRes{
				FollowingCount: 1,
			},
		},
	})

	if _, err := logic.QueryUserProfile(&types.QueryUserProfileReq{UserId: 2004}); err == nil {
		t.Fatal("expected user rpc failure")
	}
}

type stubUserService struct {
	profile *userservice.GetUserProfileRes
	me      *userservice.GetMeRes
	err     error
}

func (s *stubUserService) Register(ctx context.Context, in *userservice.RegisterReq, opts ...grpc.CallOption) (*userservice.RegisterRes, error) {
	return &userservice.RegisterRes{}, nil
}

func (s *stubUserService) Login(ctx context.Context, in *userservice.LoginReq, opts ...grpc.CallOption) (*userservice.LoginRes, error) {
	return &userservice.LoginRes{}, nil
}

func (s *stubUserService) Logout(ctx context.Context, in *userservice.LogoutReq, opts ...grpc.CallOption) (*userservice.LogoutRes, error) {
	return &userservice.LogoutRes{}, nil
}

func (s *stubUserService) GetMe(ctx context.Context, in *userservice.GetMeReq, opts ...grpc.CallOption) (*userservice.GetMeRes, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.me, nil
}

func (s *stubUserService) GetUser(ctx context.Context, in *userservice.GetUserReq, opts ...grpc.CallOption) (*userservice.GetUserRes, error) {
	return &userservice.GetUserRes{}, nil
}

func (s *stubUserService) GetUserProfile(ctx context.Context, in *userservice.GetUserProfileReq, opts ...grpc.CallOption) (*userservice.GetUserProfileRes, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.profile, nil
}

func (s *stubUserService) BatchGetUser(ctx context.Context, in *userservice.BatchGetUserReq, opts ...grpc.CallOption) (*userservice.BatchGetUserRes, error) {
	return &userservice.BatchGetUserRes{}, nil
}

type stubCounterService struct {
	counts *counterservice.GetUserProfileCountsRes
	err    error
	delay  time.Duration
}

func (s *stubCounterService) GetCount(ctx context.Context, in *counterservice.GetCountReq, opts ...grpc.CallOption) (*counterservice.GetCountRes, error) {
	return &counterservice.GetCountRes{}, nil
}

func (s *stubCounterService) BatchGetCount(ctx context.Context, in *counterservice.BatchGetCountReq, opts ...grpc.CallOption) (*counterservice.BatchGetCountRes, error) {
	return &counterservice.BatchGetCountRes{}, nil
}

func (s *stubCounterService) Inc(ctx context.Context, in *counterservice.IncReq, opts ...grpc.CallOption) (*counterservice.IncRes, error) {
	return &counterservice.IncRes{}, nil
}

func (s *stubCounterService) Dec(ctx context.Context, in *counterservice.DecReq, opts ...grpc.CallOption) (*counterservice.DecRes, error) {
	return &counterservice.DecRes{}, nil
}

func (s *stubCounterService) GetUserProfileCounts(ctx context.Context, in *counterservice.GetUserProfileCountsReq, opts ...grpc.CallOption) (*counterservice.GetUserProfileCountsRes, error) {
	if s.delay > 0 {
		timer := time.NewTimer(s.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.counts, nil
}
