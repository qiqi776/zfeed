package logic

import (
	"context"
	"testing"

	"zfeed/app/rpc/user/internal/model"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
)

func TestBatchGetUserFiltersNonActive(t *testing.T) {
	db := newUserLogicTestDB(t)
	users := []model.ZfeedUser{
		{
			ID:        3201,
			Username:  "active-one",
			Mobile:    "13800003201",
			Nickname:  "active one",
			Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
			IsDeleted: 0,
		},
		{
			ID:        3202,
			Username:  "disabled-user",
			Mobile:    "13800003202",
			Nickname:  "disabled user",
			Status:    int32(user.UserStatus_USER_STATUS_DISABLED),
			IsDeleted: 0,
		},
		{
			ID:        3203,
			Username:  "unknown-user",
			Mobile:    "13800003203",
			Nickname:  "unknown user",
			Status:    int32(user.UserStatus_USER_STATUS_UNKNOWN),
			IsDeleted: 0,
		},
		{
			ID:        3204,
			Username:  "active-two",
			Mobile:    "13800003204",
			Nickname:  "active two",
			Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
			IsDeleted: 0,
		},
		{
			ID:        3205,
			Username:  "unexpected-user",
			Mobile:    "13800003205",
			Nickname:  "unexpected user",
			Status:    99,
			IsDeleted: 0,
		},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	logic := NewBatchGetUserLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	resp, err := logic.BatchGetUser(&user.BatchGetUserReq{
		UserIds: []int64{3204, 3202, 3201, 3203, 3205, 3204, -1, 0},
	})
	if err != nil {
		t.Fatalf("BatchGetUser returned error: %v", err)
	}

	got := resp.GetUsers()
	if len(got) != 2 {
		t.Fatalf("users len = %d, want 2: %+v", len(got), got)
	}
	if got[0].GetUserId() != 3204 || got[0].GetStatus() != user.UserStatus_USER_STATUS_ACTIVE {
		t.Fatalf("first user = %+v, want active user 3204", got[0])
	}
	if got[1].GetUserId() != 3201 || got[1].GetStatus() != user.UserStatus_USER_STATUS_ACTIVE {
		t.Fatalf("second user = %+v, want active user 3201", got[1])
	}
}

func TestBatchGetUserHidesMobile(t *testing.T) {
	db := newUserLogicTestDB(t)
	if err := db.Create(&model.ZfeedUser{
		ID:        3801,
		Username:  "batch-public",
		Mobile:    "13800003801",
		Nickname:  "batch public",
		Avatar:    "https://example.com/batch.png",
		Bio:       "batch bio",
		Gender:    int32(user.Gender_GENDER_MALE),
		Status:    int32(user.UserStatus_USER_STATUS_ACTIVE),
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	logic := NewBatchGetUserLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
	})
	resp, err := logic.BatchGetUser(&user.BatchGetUserReq{UserIds: []int64{3801}})
	if err != nil {
		t.Fatalf("BatchGetUser: %v", err)
	}
	got := resp.GetUsers()
	if len(got) != 1 {
		t.Fatalf("users len = %d, want 1", len(got))
	}
	if got[0].GetUserId() != 3801 || got[0].GetNickname() != "batch public" {
		t.Fatalf("user info = %+v, want batch public fields", got[0])
	}
	if got[0].GetMobile() != "" {
		t.Fatalf("mobile = %q, want empty", got[0].GetMobile())
	}
}
