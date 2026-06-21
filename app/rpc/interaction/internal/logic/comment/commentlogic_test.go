package commentlogic

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/app/rpc/interaction/internal/testutil/mysqltest"
	"zfeed/app/rpc/user/client/userservice"
)

const (
	commentLogicTestMinID int64 = 911000
	commentLogicTestMaxID int64 = 911999
)

type commentTestContent struct {
	ID        int64 `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64 `gorm:"column:user_id"`
	IsDeleted int32 `gorm:"column:is_deleted"`
}

func (commentTestContent) TableName() string {
	return "zfeed_content"
}

type commentTestComment struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	ContentID     int64     `gorm:"column:content_id"`
	ContentUserID int64     `gorm:"column:content_user_id"`
	UserID        int64     `gorm:"column:user_id"`
	ReplyToUserID int64     `gorm:"column:reply_to_user_id"`
	ParentID      int64     `gorm:"column:parent_id"`
	RootID        int64     `gorm:"column:root_id"`
	Comment       string    `gorm:"column:comment"`
	Status        int32     `gorm:"column:status"`
	Version       int32     `gorm:"column:version"`
	ReplyCount    int64     `gorm:"column:reply_count"`
	IsDeleted     int32     `gorm:"column:is_deleted"`
	CreatedBy     int64     `gorm:"column:created_by"`
	UpdatedBy     int64     `gorm:"column:updated_by"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (commentTestComment) TableName() string {
	return "zfeed_comment"
}

type fakeCommentUserActionProducer struct {
	events []event.UserActionEvent
	err    error
}

func (p *fakeCommentUserActionProducer) SendUserAction(_ context.Context, action event.UserActionEvent) error {
	p.events = append(p.events, action)
	return p.err
}

func TestCommentEmitsUserActionAfterSuccessfulCreate(t *testing.T) {
	db := newCommentUserActionTestDB(t)

	const (
		userID        int64 = 1001
		contentID     int64 = 9001
		contentUserID int64 = 2001
	)
	seedCommentTestContent(t, db, contentID, contentUserID)

	userActionProducer := &fakeCommentUserActionProducer{}
	resp, err := NewCommentLogic(context.Background(), &svc.ServiceContext{
		MysqlDb:            db,
		UserActionProducer: userActionProducer,
	}).Comment(&interaction.CommentReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "user action comment",
	})
	if err != nil {
		t.Fatalf("Comment returned error: %v", err)
	}
	if resp.GetCommentId() <= 0 {
		t.Fatalf("comment_id = %d, want positive", resp.GetCommentId())
	}
	if len(userActionProducer.events) != 1 {
		t.Fatalf("user action events = %+v, want one event", userActionProducer.events)
	}
	if got := userActionProducer.events[0]; got.Action != event.UserActionComment ||
		got.UserID != userID ||
		got.TargetID != contentID ||
		got.ContentID != contentID ||
		got.ContentUserID != contentUserID ||
		got.Scene != interaction.Scene_ARTICLE.String() {
		t.Fatalf("comment user action = %+v", got)
	}
}

func TestCommentIgnoresUserActionFailure(t *testing.T) {
	db := newCommentUserActionTestDB(t)

	const (
		userID        int64 = 1002
		contentID     int64 = 9002
		contentUserID int64 = 2002
	)
	seedCommentTestContent(t, db, contentID, contentUserID)

	userActionProducer := &fakeCommentUserActionProducer{err: errors.New("kafka unavailable")}
	resp, err := NewCommentLogic(context.Background(), &svc.ServiceContext{
		MysqlDb:            db,
		UserActionProducer: userActionProducer,
	}).Comment(&interaction.CommentReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "best effort user action",
	})
	if err != nil {
		t.Fatalf("Comment returned error: %v", err)
	}
	if resp.GetCommentId() <= 0 {
		t.Fatalf("comment_id = %d, want positive", resp.GetCommentId())
	}
	if len(userActionProducer.events) != 1 {
		t.Fatalf("user action events = %+v, want one attempted event", userActionProducer.events)
	}
}

func TestCommentWriteErrors(t *testing.T) {
	ctx := context.Background()
	contentID := int64(9011)
	contentUserID := int64(2011)
	userID := int64(1011)

	tests := []struct {
		name string
		repo *fakeCommentRepository
		req  *interaction.CommentReq
	}{
		{
			name: "create error",
			repo: &fakeCommentRepository{createErr: errors.New("create comment failed")},
			req: &interaction.CommentReq{
				UserId:        userID,
				ContentId:     contentID,
				ContentUserId: contentUserID,
				Scene:         interaction.Scene_ARTICLE,
				Comment:       "create should fail",
			},
		},
		{
			name: "reply count error",
			repo: &fakeCommentRepository{
				createdID: 701,
				incErr:    errors.New("increment reply count failed"),
				getByID: map[int64]*do.CommentDO{
					700: {ID: 700, ContentID: contentID, UserID: 1700, RootID: 0},
				},
			},
			req: &interaction.CommentReq{
				UserId:        userID,
				ContentId:     contentID,
				ContentUserId: contentUserID,
				Scene:         interaction.Scene_ARTICLE,
				Comment:       "reply count should fail",
				ParentId:      700,
				RootId:        700,
				ReplyToUserId: 1700,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userActionProducer := &fakeCommentUserActionProducer{}
			logic := NewCommentLogic(ctx, &svc.ServiceContext{
				MysqlDb:            newCommentUserActionTestDB(t),
				UserActionProducer: userActionProducer,
			})
			logic.contentRepo = &fakeContentRepository{authorID: contentUserID}
			logic.commentRepo = tt.repo

			resp, err := logic.Comment(tt.req)
			if err == nil {
				t.Fatal("expected Comment error")
			}
			if resp != nil {
				t.Fatalf("Comment response = %+v, want nil", resp)
			}
			if len(userActionProducer.events) != 0 {
				t.Fatalf("user action events = %+v, want none", userActionProducer.events)
			}
		})
	}
}

func TestCommentAuthorError(t *testing.T) {
	userActionProducer := &fakeCommentUserActionProducer{}
	logic := NewCommentLogic(context.Background(), &svc.ServiceContext{
		MysqlDb:            newCommentUserActionTestDB(t),
		UserActionProducer: userActionProducer,
	})
	logic.contentRepo = &fakeContentRepository{err: errors.New("author query failed")}
	logic.commentRepo = &fakeCommentRepository{}

	resp, err := logic.Comment(&interaction.CommentReq{
		UserId:        1012,
		ContentId:     9012,
		ContentUserId: 2012,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "author lookup should fail",
	})
	if err == nil {
		t.Fatal("expected Comment error")
	}
	if resp != nil {
		t.Fatalf("Comment response = %+v, want nil", resp)
	}
	if len(userActionProducer.events) != 0 {
		t.Fatalf("user action events = %+v, want none", userActionProducer.events)
	}
}

func TestCommentThreadError(t *testing.T) {
	ctx := context.Background()
	const (
		contentID     int64 = 9013
		contentUserID int64 = 2013
		userID        int64 = 1013
		parentID      int64 = 3013
	)
	repo := &fakeCommentRepository{getByID: map[int64]*do.CommentDO{}}
	userActionProducer := &fakeCommentUserActionProducer{}
	logic := NewCommentLogic(ctx, &svc.ServiceContext{
		MysqlDb:            newCommentUserActionTestDB(t),
		UserActionProducer: userActionProducer,
	})
	logic.contentRepo = &fakeContentRepository{authorID: contentUserID}
	logic.commentRepo = repo

	resp, err := logic.Comment(&interaction.CommentReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "missing parent should fail",
		ParentId:      parentID,
		RootId:        parentID,
		ReplyToUserId: 4013,
	})
	if err == nil {
		t.Fatal("expected Comment error")
	}
	if resp != nil {
		t.Fatalf("Comment response = %+v, want nil", resp)
	}
	if repo.createCalls != 0 {
		t.Fatalf("Create calls = %d, want 0", repo.createCalls)
	}
	if len(userActionProducer.events) != 0 {
		t.Fatalf("user action events = %+v, want none", userActionProducer.events)
	}
}

func TestCommentRejects(t *testing.T) {
	db := newCommentUserActionTestDB(t)
	const (
		contentID     int64 = 9301
		contentUserID int64 = 9401
		userID        int64 = 9501
	)
	seedCommentTestContent(t, db, contentID, contentUserID)

	logic := NewCommentLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	tests := []struct {
		name string
		req  *interaction.CommentReq
	}{
		{name: "nil request"},
		{name: "missing user", req: &interaction.CommentReq{ContentId: contentID, ContentUserId: contentUserID, Scene: interaction.Scene_ARTICLE, Comment: "ok"}},
		{name: "missing content", req: &interaction.CommentReq{UserId: userID, ContentUserId: contentUserID, Scene: interaction.Scene_ARTICLE, Comment: "ok"}},
		{name: "missing author", req: &interaction.CommentReq{UserId: userID, ContentId: contentID, Scene: interaction.Scene_ARTICLE, Comment: "ok"}},
		{name: "unknown scene", req: &interaction.CommentReq{UserId: userID, ContentId: contentID, ContentUserId: contentUserID, Scene: interaction.Scene_SCENE_UNKNOWN, Comment: "ok"}},
		{name: "blank text", req: &interaction.CommentReq{UserId: userID, ContentId: contentID, ContentUserId: contentUserID, Scene: interaction.Scene_ARTICLE, Comment: "  \t\n  "}},
		{name: "overlong text", req: &interaction.CommentReq{UserId: userID, ContentId: contentID, ContentUserId: contentUserID, Scene: interaction.Scene_ARTICLE, Comment: strings.Repeat("x", 256)}},
		{name: "content not found", req: &interaction.CommentReq{UserId: userID, ContentId: contentID + 1, ContentUserId: contentUserID, Scene: interaction.Scene_ARTICLE, Comment: "ok"}},
		{name: "author mismatch", req: &interaction.CommentReq{UserId: userID, ContentId: contentID, ContentUserId: contentUserID + 1, Scene: interaction.Scene_ARTICLE, Comment: "ok"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := logic.Comment(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	var count int64
	if err := db.Model(&commentTestComment{}).Count(&count).Error; err != nil {
		t.Fatalf("count comments: %v", err)
	}
	if count != 0 {
		t.Fatalf("comment count = %d, want 0", count)
	}
}

func TestResolveThread(t *testing.T) {
	ctx := context.Background()
	logic := &CommentLogic{ctx: ctx}
	const contentID int64 = 9601
	root := &do.CommentDO{ID: 100, ContentID: contentID, UserID: 200, RootID: 0}
	reply := &do.CommentDO{ID: 101, ContentID: contentID, UserID: 201, ParentID: 100, RootID: 100}

	tests := []struct {
		name        string
		repo        *fakeCommentRepository
		parentID    int64
		rootID      int64
		replyToID   int64
		wantParent  int64
		wantRoot    int64
		wantReplyTo int64
		wantError   bool
	}{
		{
			name: "root comment",
			repo: &fakeCommentRepository{},
		},
		{
			name:      "missing parent",
			repo:      &fakeCommentRepository{},
			rootID:    root.ID,
			wantError: true,
		},
		{
			name:        "direct reply",
			repo:        &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: root}},
			parentID:    root.ID,
			wantParent:  root.ID,
			wantRoot:    root.ID,
			wantReplyTo: root.UserID,
		},
		{
			name:        "nested reply",
			repo:        &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: root, reply.ID: reply}},
			parentID:    reply.ID,
			rootID:      root.ID,
			replyToID:   reply.UserID,
			wantParent:  reply.ID,
			wantRoot:    root.ID,
			wantReplyTo: reply.UserID,
		},
		{
			name:      "parent not found",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{}},
			parentID:  root.ID,
			wantError: true,
		},
		{
			name:      "wrong content",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: {ID: root.ID, ContentID: contentID + 1, UserID: root.UserID}}},
			parentID:  root.ID,
			wantError: true,
		},
		{
			name:      "wrong root",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: root}},
			parentID:  root.ID,
			rootID:    root.ID + 1,
			wantError: true,
		},
		{
			name:      "wrong reply target",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: root}},
			parentID:  root.ID,
			replyToID: root.UserID + 1,
			wantError: true,
		},
		{
			name:      "root not found",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{reply.ID: reply}},
			parentID:  reply.ID,
			rootID:    root.ID,
			replyToID: reply.UserID,
			wantError: true,
		},
		{
			name:      "bad root",
			repo:      &fakeCommentRepository{getByID: map[int64]*do.CommentDO{root.ID: {ID: root.ID, ContentID: contentID, UserID: root.UserID, RootID: 999}, reply.ID: reply}},
			parentID:  reply.ID,
			rootID:    root.ID,
			replyToID: reply.UserID,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentID, rootID, replyToID, err := logic.resolveThread(tt.repo, contentID, tt.parentID, tt.rootID, tt.replyToID)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveThread returned error: %v", err)
			}
			if parentID != tt.wantParent || rootID != tt.wantRoot || replyToID != tt.wantReplyTo {
				t.Fatalf("resolved = (%d,%d,%d), want (%d,%d,%d)", parentID, rootID, replyToID, tt.wantParent, tt.wantRoot, tt.wantReplyTo)
			}
		})
	}
}

func TestResolveThreadRepoError(t *testing.T) {
	ctx := context.Background()
	logic := &CommentLogic{ctx: ctx}
	const contentID int64 = 9602
	root := &do.CommentDO{ID: 200, ContentID: contentID, UserID: 300, RootID: 0}
	reply := &do.CommentDO{ID: 201, ContentID: contentID, UserID: 301, ParentID: 200, RootID: 200}

	parentErr := errors.New("parent query failed")
	rootErr := errors.New("root query failed")
	tests := []struct {
		name    string
		repo    *fakeCommentRepository
		parent  int64
		root    int64
		replyTo int64
		wantErr error
	}{
		{
			name:    "parent query error",
			repo:    &fakeCommentRepository{getErrByID: map[int64]error{reply.ID: parentErr}},
			parent:  reply.ID,
			root:    root.ID,
			replyTo: reply.UserID,
			wantErr: parentErr,
		},
		{
			name: "root query error",
			repo: &fakeCommentRepository{
				getByID: map[int64]*do.CommentDO{reply.ID: reply},
				getErrByID: map[int64]error{
					root.ID: rootErr,
				},
			},
			parent:  reply.ID,
			root:    root.ID,
			replyTo: reply.UserID,
			wantErr: rootErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentID, rootID, replyToID, err := logic.resolveThread(tt.repo, contentID, tt.parent, tt.root, tt.replyTo)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("resolveThread error = %v, want %v", err, tt.wantErr)
			}
			if parentID != 0 || rootID != 0 || replyToID != 0 {
				t.Fatalf("resolved = (%d,%d,%d), want zeros", parentID, rootID, replyToID)
			}
		})
	}
}

func TestCommentCacheRoot(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	const (
		userID        int64 = 9701
		contentID     int64 = 9702
		contentUserID int64 = 9703
	)
	seedCommentTestContent(t, db, contentID, contentUserID)

	resp, err := NewCommentLogic(ctx, &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{
			users: map[int64]*userservice.UserInfo{
				userID: {UserId: userID, Nickname: "root-user", Avatar: "root.png"},
			},
		},
	}).Comment(&interaction.CommentReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       " cached root ",
	})
	if err != nil {
		t.Fatalf("Comment returned error: %v", err)
	}

	commentID := resp.GetCommentId()
	listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	members, err := store.ZMembers(listKey)
	if err != nil {
		t.Fatalf("read list cache: %v", err)
	}
	if !reflect.DeepEqual(members, []string{strconv.FormatInt(commentID, 10)}) {
		t.Fatalf("list members = %v, want [%d]", members, commentID)
	}

	raw, err := redisClient.GetCtx(ctx, rediskey.BuildCommentItemKey(strconv.FormatInt(commentID, 10)))
	if err != nil {
		t.Fatalf("read item cache: %v", err)
	}
	item, err := unmarshalCommentItem(raw)
	if err != nil {
		t.Fatalf("unmarshal item cache: %v", err)
	}
	if item.GetCommentId() != commentID || item.GetComment() != "cached root" || item.GetUserName() != "root-user" || item.GetUserAvatar() != "root.png" {
		t.Fatalf("cached item = %+v", item)
	}
}

func TestCommentCacheReply(t *testing.T) {
	ctx := context.Background()
	db := newCommentUserActionTestDB(t)
	store, redisClient := newCommentCacheRedis(t)
	const (
		contentID     int64 = 9801
		contentUserID int64 = 9802
		rootUserID    int64 = 9803
		replyUserID   int64 = 9804
		nestedUserID  int64 = 9805
	)
	seedCommentTestContent(t, db, contentID, contentUserID)
	svcCtx := &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
		UserRpc: &fakeCommentUserService{},
	}

	rootRes, err := NewCommentLogic(ctx, svcCtx).Comment(&interaction.CommentReq{
		UserId:        rootUserID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "root",
	})
	if err != nil {
		t.Fatalf("create root: %v", err)
	}
	rootID := rootRes.GetCommentId()

	replyRes, err := NewCommentLogic(ctx, svcCtx).Comment(&interaction.CommentReq{
		UserId:        replyUserID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "reply",
		ParentId:      rootID,
		RootId:        rootID,
		ReplyToUserId: rootUserID,
	})
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	replyID := replyRes.GetCommentId()

	listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
	replyKey := rediskey.BuildCommentReplyKey(strconv.FormatInt(rootID, 10))
	rootItemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(rootID, 10))
	parentItemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(replyID, 10))
	store.ZAdd(listKey, float64(rootID), strconv.FormatInt(rootID, 10))
	store.ZAdd(replyKey, float64(replyID), strconv.FormatInt(replyID, 10))
	store.Set(rootItemKey, "cached-root")
	store.Set(parentItemKey, "cached-parent")

	_, err = NewCommentLogic(ctx, svcCtx).Comment(&interaction.CommentReq{
		UserId:        nestedUserID,
		ContentId:     contentID,
		ContentUserId: contentUserID,
		Scene:         interaction.Scene_ARTICLE,
		Comment:       "nested",
		ParentId:      replyID,
		RootId:        rootID,
		ReplyToUserId: replyUserID,
	})
	if err != nil {
		t.Fatalf("create nested reply: %v", err)
	}

	for _, key := range []string{listKey, replyKey, rootItemKey, parentItemKey} {
		if store.Exists(key) {
			t.Fatalf("cache key %s still exists after reply create", key)
		}
	}

	root := requireDeleteCommentRow(t, db, rootID)
	if root.ReplyCount != 2 {
		t.Fatalf("root reply_count = %d, want 2", root.ReplyCount)
	}
}

func TestCommentCacheSkips(t *testing.T) {
	ctx := context.Background()
	const (
		commentID int64 = 990101
		contentID int64 = 990102
		userID    int64 = 990103
	)

	NewCommentLogic(ctx, &svc.ServiceContext{}).updateCommentCacheAfterCreate(
		commentID,
		contentID,
		interaction.Scene_ARTICLE,
		0,
		0,
	)

	_, redisClient := newCommentCacheRedis(t)
	NewCommentLogic(ctx, &svc.ServiceContext{
		Redis: redisClient,
	}).updateCommentCacheAfterCreate(
		0,
		contentID,
		interaction.Scene_ARTICLE,
		0,
		0,
	)

	tests := []struct {
		name    string
		repo    *fakeCommentRepository
		userRPC *fakeCommentUserService
	}{
		{
			name: "repo error",
			repo: &fakeCommentRepository{
				getErr: errors.New("comment lookup failed"),
			},
			userRPC: &fakeCommentUserService{},
		},
		{
			name:    "missing comment",
			repo:    &fakeCommentRepository{getByID: map[int64]*do.CommentDO{}},
			userRPC: &fakeCommentUserService{},
		},
		{
			name: "user error",
			repo: &fakeCommentRepository{
				getByID: map[int64]*do.CommentDO{
					commentID: {
						ID:        commentID,
						ContentID: contentID,
						UserID:    userID,
						Comment:   "root",
						Status:    repositories.CommentStatusNormal,
					},
				},
			},
			userRPC: &fakeCommentUserService{err: errors.New("user lookup failed")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, redisClient := newCommentCacheRedis(t)
			logic := NewCommentLogic(ctx, &svc.ServiceContext{
				Redis:   redisClient,
				UserRpc: tt.userRPC,
			})
			logic.commentRepo = tt.repo

			logic.updateCommentCacheAfterCreate(
				commentID,
				contentID,
				interaction.Scene_ARTICLE,
				0,
				0,
			)

			listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(contentID, 10))
			itemKey := rediskey.BuildCommentItemKey(strconv.FormatInt(commentID, 10))
			if store.Exists(listKey) || store.Exists(itemKey) {
				t.Fatalf("unexpected cache write: list=%v item=%v", store.Exists(listKey), store.Exists(itemKey))
			}
		})
	}
}

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

func newCommentUserActionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&commentTestContent{}, &commentTestComment{}); err != nil {
		t.Fatalf("auto migrate comment user action tables: %v", err)
	}
	return db
}

func seedCommentTestContent(t *testing.T, db *gorm.DB, contentID, userID int64) {
	t.Helper()

	if err := db.Create(&commentTestContent{
		ID:     contentID,
		UserID: userID,
	}).Error; err != nil {
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
	users      map[int64]*userservice.UserInfo
	extraUsers []*userservice.UserInfo
	err        error
	failErr    error
	failures   int
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

func (f *fakeCommentUserService) UpdateProfile(context.Context, *userservice.UpdateProfileReq, ...grpc.CallOption) (*userservice.UpdateProfileRes, error) {
	return nil, errors.New("unexpected UpdateProfile call")
}

func (f *fakeCommentUserService) GetUser(context.Context, *userservice.GetUserReq, ...grpc.CallOption) (*userservice.GetUserRes, error) {
	return nil, errors.New("unexpected GetUser call")
}

func (f *fakeCommentUserService) GetUserProfile(context.Context, *userservice.GetUserProfileReq, ...grpc.CallOption) (*userservice.GetUserProfileRes, error) {
	return nil, errors.New("unexpected GetUserProfile call")
}

func (f *fakeCommentUserService) BatchGetUser(_ context.Context, in *userservice.BatchGetUserReq, _ ...grpc.CallOption) (*userservice.BatchGetUserRes, error) {
	if f.failures > 0 {
		f.failures--
		if f.failErr != nil {
			return nil, f.failErr
		}
		return nil, errors.New("transient user rpc failure")
	}
	if f.err != nil {
		return nil, f.err
	}
	users := make([]*userservice.UserInfo, 0, len(in.GetUserIds()))
	for _, userID := range in.GetUserIds() {
		if user := f.users[userID]; user != nil {
			users = append(users, user)
		}
	}
	users = append(users, f.extraUsers...)
	return &userservice.BatchGetUserRes{Users: users}, nil
}
