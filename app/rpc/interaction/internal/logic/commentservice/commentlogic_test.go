package commentservicelogic

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
	"zfeed/app/rpc/user/client/userservice"
)

const (
	commentLogicTestMinID int64 = 911000
	commentLogicTestMaxID int64 = 911999
)

func TestCommentFlowCreateAndQuery(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 101
	contentUserID := commentLogicTestMinID + 201
	rootUserID := commentLogicTestMinID + 1
	replyUserID := commentLogicTestMinID + 2
	nestedUserID := commentLogicTestMinID + 3

	seedContentRow(t, db, contentID, contentUserID)

	svcCtx := &svc.ServiceContext{
		MysqlDb: db,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				rootUserID:   &userservice.UserInfo{UserId: rootUserID, Nickname: "root-user", Avatar: "root.png"},
				replyUserID:  &userservice.UserInfo{UserId: replyUserID, Nickname: "reply-user", Avatar: "reply.png"},
				nestedUserID: &userservice.UserInfo{UserId: nestedUserID, Nickname: "nested-user", Avatar: "nested.png"},
			},
		},
	}

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "reply comment",
		ParentId:      rootRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: rootUserID,
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	nestedRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        nestedUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "nested reply",
		ParentId:      replyRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: replyUserID,
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create nested reply: %v", err)
	}

	rootListRes, err := NewQueryCommentListLogic(context.Background(), svcCtx).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		Cursor:    0,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("query root comment list: %v", err)
	}
	if len(rootListRes.GetComments()) != 1 {
		t.Fatalf("root comment count = %d, want 1", len(rootListRes.GetComments()))
	}

	rootItem := rootListRes.GetComments()[0]
	if rootItem.GetCommentId() != rootRes.GetCommentId() {
		t.Fatalf("root comment id = %d, want %d", rootItem.GetCommentId(), rootRes.GetCommentId())
	}
	if rootItem.GetReplyCount() != 2 {
		t.Fatalf("root reply_count = %d, want 2", rootItem.GetReplyCount())
	}
	if rootItem.GetUserName() != "root-user" || rootItem.GetUserAvatar() != "root.png" {
		t.Fatalf("root user info = (%s, %s), want (root-user, root.png)", rootItem.GetUserName(), rootItem.GetUserAvatar())
	}

	replyListRes, err := NewQueryReplyListLogic(context.Background(), svcCtx).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootRes.GetCommentId(),
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("query reply list: %v", err)
	}
	if len(replyListRes.GetReplies()) != 2 {
		t.Fatalf("reply count = %d, want 2", len(replyListRes.GetReplies()))
	}

	nestedItem := replyListRes.GetReplies()[0]
	replyItem := replyListRes.GetReplies()[1]
	if nestedItem.GetCommentId() != nestedRes.GetCommentId() || replyItem.GetCommentId() != replyRes.GetCommentId() {
		t.Fatalf("reply order = [%d %d], want [%d %d]", nestedItem.GetCommentId(), replyItem.GetCommentId(), nestedRes.GetCommentId(), replyRes.GetCommentId())
	}
	if nestedItem.GetParentId() != replyRes.GetCommentId() || nestedItem.GetRootId() != rootRes.GetCommentId() {
		t.Fatalf("nested thread relation = parent:%d root:%d, want parent:%d root:%d", nestedItem.GetParentId(), nestedItem.GetRootId(), replyRes.GetCommentId(), rootRes.GetCommentId())
	}
	if nestedItem.GetReplyToUserId() != replyUserID {
		t.Fatalf("nested reply_to_user_id = %d, want %d", nestedItem.GetReplyToUserId(), replyUserID)
	}
	if nestedItem.GetUserName() != "nested-user" || replyItem.GetUserName() != "reply-user" {
		t.Fatalf("reply user names = [%s %s], want [nested-user reply-user]", nestedItem.GetUserName(), replyItem.GetUserName())
	}
}

func TestCommentLogicRejectsWrongReplyTarget(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 102
	contentUserID := commentLogicTestMinID + 202
	rootUserID := commentLogicTestMinID + 11
	replyUserID := commentLogicTestMinID + 12

	seedContentRow(t, db, contentID, contentUserID)

	svcCtx := &svc.ServiceContext{
		MysqlDb: db,
		UserRpc: &fakeCommentUserService{},
	}

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	_, err = NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "bad reply",
		ParentId:      rootRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: contentUserID,
		ContentUserId: contentUserID,
	})
	if err == nil {
		t.Fatal("expected reply target validation error")
	}
}

func TestDeleteCommentPhysicallyDeletesRootWithoutReplies(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 103
	contentUserID := commentLogicTestMinID + 203
	rootUserID := commentLogicTestMinID + 21

	seedContentRow(t, db, contentID, contentUserID)
	svcCtx := newCommentLogicTestSvcCtx(db, map[int64]*userservice.UserInfo{})

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	if _, err := NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    rootUserID,
		CommentId: rootRes.GetCommentId(),
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	}); err != nil {
		t.Fatalf("delete root comment: %v", err)
	}

	commentRepo := repositories.NewCommentRepository(context.Background(), db)
	commentDO, err := commentRepo.GetByIDIncludeDeleted(rootRes.GetCommentId())
	if err != nil {
		t.Fatalf("get deleted root comment: %v", err)
	}
	if commentDO != nil {
		t.Fatal("expected root comment to be physically deleted")
	}

	rootListRes, err := NewQueryCommentListLogic(context.Background(), svcCtx).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		Cursor:    0,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("query root comment list: %v", err)
	}
	if len(rootListRes.GetComments()) != 0 {
		t.Fatalf("root comment count = %d, want 0", len(rootListRes.GetComments()))
	}
}

func TestDeleteCommentTombstonesRootWithReplies(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 104
	contentUserID := commentLogicTestMinID + 204
	rootUserID := commentLogicTestMinID + 31
	replyUserID := commentLogicTestMinID + 32

	seedContentRow(t, db, contentID, contentUserID)
	svcCtx := newCommentLogicTestSvcCtx(db, map[int64]*userservice.UserInfo{
		replyUserID: &userservice.UserInfo{UserId: replyUserID, Nickname: "reply-user", Avatar: "reply.png"},
	})

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "reply comment",
		ParentId:      rootRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: rootUserID,
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	if _, err := NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    rootUserID,
		CommentId: rootRes.GetCommentId(),
		ContentId: contentID,
		RootId:    nil,
		ParentId:  nil,
		Scene:     interaction.Scene_ARTICLE,
	}); err != nil {
		t.Fatalf("delete root comment: %v", err)
	}

	rootListRes, err := NewQueryCommentListLogic(context.Background(), svcCtx).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		Cursor:    0,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("query root comment list: %v", err)
	}
	if len(rootListRes.GetComments()) != 1 {
		t.Fatalf("root comment count = %d, want 1", len(rootListRes.GetComments()))
	}
	rootItem := rootListRes.GetComments()[0]
	if rootItem.GetStatus() != repositories.CommentStatusDeleted {
		t.Fatalf("root status = %d, want %d", rootItem.GetStatus(), repositories.CommentStatusDeleted)
	}
	if rootItem.GetComment() != "该评论已删除" {
		t.Fatalf("root comment text = %q, want tombstone", rootItem.GetComment())
	}
	if rootItem.GetUserId() != 0 || rootItem.GetUserName() != "" || rootItem.GetUserAvatar() != "" {
		t.Fatalf("root tombstone user info = (%d,%q,%q), want zero values", rootItem.GetUserId(), rootItem.GetUserName(), rootItem.GetUserAvatar())
	}
	if rootItem.GetReplyCount() != 1 {
		t.Fatalf("root reply_count = %d, want 1", rootItem.GetReplyCount())
	}

	replyListRes, err := NewQueryReplyListLogic(context.Background(), svcCtx).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootRes.GetCommentId(),
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("query reply list: %v", err)
	}
	if len(replyListRes.GetReplies()) != 1 || replyListRes.GetReplies()[0].GetCommentId() != replyRes.GetCommentId() {
		t.Fatalf("reply list does not preserve child comment after tombstone delete")
	}
}

func TestDeleteCommentDeletesReplyAndDecrementsRootCount(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 105
	contentUserID := commentLogicTestMinID + 205
	rootUserID := commentLogicTestMinID + 41
	replyUserID := commentLogicTestMinID + 42

	seedContentRow(t, db, contentID, contentUserID)
	svcCtx := newCommentLogicTestSvcCtx(db, map[int64]*userservice.UserInfo{
		rootUserID: &userservice.UserInfo{UserId: rootUserID, Nickname: "root-user", Avatar: "root.png"},
	})

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "reply comment",
		ParentId:      rootRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: rootUserID,
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	rootCommentID := rootRes.GetCommentId()
	if _, err := NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    replyUserID,
		CommentId: replyRes.GetCommentId(),
		ContentId: contentID,
		RootId:    &rootCommentID,
		ParentId:  &rootCommentID,
		Scene:     interaction.Scene_ARTICLE,
	}); err != nil {
		t.Fatalf("delete reply comment: %v", err)
	}

	replyListRes, err := NewQueryReplyListLogic(context.Background(), svcCtx).QueryReplyList(&interaction.QueryReplyListReq{
		RootId:   rootRes.GetCommentId(),
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("query reply list: %v", err)
	}
	if len(replyListRes.GetReplies()) != 0 {
		t.Fatalf("reply count = %d, want 0", len(replyListRes.GetReplies()))
	}

	rootListRes, err := NewQueryCommentListLogic(context.Background(), svcCtx).QueryCommentList(&interaction.QueryCommentListReq{
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
		Cursor:    0,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("query root comment list: %v", err)
	}
	if len(rootListRes.GetComments()) != 1 {
		t.Fatalf("root comment count = %d, want 1", len(rootListRes.GetComments()))
	}
	if rootListRes.GetComments()[0].GetReplyCount() != 0 {
		t.Fatalf("root reply_count = %d, want 0", rootListRes.GetComments()[0].GetReplyCount())
	}
}

func TestDeleteCommentRejectsNonOwner(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 106
	contentUserID := commentLogicTestMinID + 206
	rootUserID := commentLogicTestMinID + 51
	otherUserID := commentLogicTestMinID + 52

	seedContentRow(t, db, contentID, contentUserID)
	svcCtx := newCommentLogicTestSvcCtx(db, map[int64]*userservice.UserInfo{})

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	_, err = NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    otherUserID,
		CommentId: rootRes.GetCommentId(),
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err == nil {
		t.Fatal("expected delete permission error")
	}
	if err.Error() != "无权限删除评论" {
		t.Fatalf("delete error = %q, want %q", err.Error(), "无权限删除评论")
	}
}

func TestDeleteCommentCleansTombstoneAncestorWhenLastChildRemoved(t *testing.T) {
	db, cleanup := openCommentLogicTestDB(t)
	defer cleanup()

	contentID := commentLogicTestMinID + 107
	contentUserID := commentLogicTestMinID + 207
	rootUserID := commentLogicTestMinID + 61
	replyUserID := commentLogicTestMinID + 62

	seedContentRow(t, db, contentID, contentUserID)
	svcCtx := newCommentLogicTestSvcCtx(db, map[int64]*userservice.UserInfo{})

	rootRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root comment",
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create root comment: %v", err)
	}

	replyRes, err := NewCommentLogic(context.Background(), svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "reply comment",
		ParentId:      rootRes.GetCommentId(),
		RootId:        rootRes.GetCommentId(),
		ReplyToUserId: rootUserID,
		ContentUserId: contentUserID,
	})
	if err != nil {
		t.Fatalf("create reply comment: %v", err)
	}

	rootCommentID := rootRes.GetCommentId()
	if _, err := NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    rootUserID,
		CommentId: rootRes.GetCommentId(),
		ContentId: contentID,
		Scene:     interaction.Scene_ARTICLE,
	}); err != nil {
		t.Fatalf("delete root comment: %v", err)
	}

	if _, err := NewDeleteCommentLogic(context.Background(), svcCtx).DeleteComment(&interaction.DeleteCommentReq{
		UserId:    replyUserID,
		CommentId: replyRes.GetCommentId(),
		ContentId: contentID,
		RootId:    &rootCommentID,
		ParentId:  &rootCommentID,
		Scene:     interaction.Scene_ARTICLE,
	}); err != nil {
		t.Fatalf("delete reply comment: %v", err)
	}

	commentRepo := repositories.NewCommentRepository(context.Background(), db)
	rootComment, err := commentRepo.GetByIDIncludeDeleted(rootRes.GetCommentId())
	if err != nil {
		t.Fatalf("get root comment: %v", err)
	}
	if rootComment != nil {
		t.Fatal("expected tombstone root to be physically cleaned after last child removal")
	}
}

func openCommentLogicTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	db, err := mysqltest.Open()
	if err != nil {
		t.Skipf("skip MySQL-backed comment logic test: %v", err)
	}

	if err := mysqltest.EnsureCommentTables(db); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("ensure comment tables: %v", err)
	}

	if err := mysqltest.CleanupCommentRowsByRange(db, commentLogicTestMinID, commentLogicTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup comment rows before run: %v", err)
	}
	if err := mysqltest.CleanupContentRowsByRange(db, commentLogicTestMinID, commentLogicTestMaxID); err != nil {
		_ = mysqltest.Close(db)
		t.Fatalf("cleanup content rows before run: %v", err)
	}

	return db, func() {
		if err := mysqltest.CleanupCommentRowsByRange(db, commentLogicTestMinID, commentLogicTestMaxID); err != nil {
			t.Fatalf("cleanup comment rows after run: %v", err)
		}
		if err := mysqltest.CleanupContentRowsByRange(db, commentLogicTestMinID, commentLogicTestMaxID); err != nil {
			t.Fatalf("cleanup content rows after run: %v", err)
		}
		if err := mysqltest.Close(db); err != nil {
			t.Fatalf("close db: %v", err)
		}
	}
}

func seedContentRow(t *testing.T, db *gorm.DB, contentID, userID int64) {
	t.Helper()

	if err := db.Exec(
		`INSERT INTO zfeed_content (
			id, user_id, content_type, status, visibility, like_count, favorite_count, comment_count,
			published_at, is_deleted, created_by, updated_by
		) VALUES (?, ?, 10, 10, 10, 0, 0, 0, NOW(), 0, ?, ?)`,
		contentID,
		userID,
		userID,
		userID,
	).Error; err != nil {
		t.Fatalf("seed content row: %v", err)
	}
}

func newCommentLogicTestSvcCtx(db *gorm.DB, users map[int64]*userservice.UserInfo) *svc.ServiceContext {
	return &svc.ServiceContext{
		MysqlDb: db,
		UserRpc: &fakeCommentUserService{users: users},
	}
}

type fakeCommentUserService struct {
	users map[int64]*userservice.UserInfo
}

func (f *fakeCommentUserService) Register(context.Context, *userservice.RegisterReq, ...grpc.CallOption) (*userservice.RegisterRes, error) {
	return nil, errors.New("unexpected Register call")
}

func (f *fakeCommentUserService) Login(context.Context, *userservice.LoginReq, ...grpc.CallOption) (*userservice.LoginRes, error) {
	return nil, errors.New("unexpected Login call")
}

func (f *fakeCommentUserService) Logout(context.Context, *userservice.LogoutReq, ...grpc.CallOption) (*userservice.LogoutRes, error) {
	return nil, errors.New("unexpected Logout call")
}

func (f *fakeCommentUserService) GetMe(context.Context, *userservice.GetMeReq, ...grpc.CallOption) (*userservice.GetMeRes, error) {
	return nil, errors.New("unexpected GetMe call")
}

func (f *fakeCommentUserService) GetUser(context.Context, *userservice.GetUserReq, ...grpc.CallOption) (*userservice.GetUserRes, error) {
	return nil, errors.New("unexpected GetUser call")
}

func (f *fakeCommentUserService) GetUserProfile(context.Context, *userservice.GetUserProfileReq, ...grpc.CallOption) (*userservice.GetUserProfileRes, error) {
	return nil, errors.New("unexpected GetUserProfile call")
}

func (f *fakeCommentUserService) BatchGetUser(_ context.Context, in *userservice.BatchGetUserReq, _ ...grpc.CallOption) (*userservice.BatchGetUserRes, error) {
	users := make([]*userservice.UserInfo, 0, len(in.GetUserIds()))
	for _, userID := range in.GetUserIds() {
		if user := f.users[userID]; user != nil {
			users = append(users, user)
		}
	}
	return &userservice.BatchGetUserRes{Users: users}, nil
}
