package followservicelogic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gztracetest "github.com/zeromicro/go-zero/core/trace/tracetest"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/model"
	"zfeed/app/rpc/interaction/internal/svc"
)

var _ contentservice.ContentService = (*fakeContentService)(nil)

type fakeContentService struct {
	backfillFunc func(ctx context.Context, in *contentpb.BackfillFollowInboxReq, opts ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error)
}

func (f *fakeContentService) PublishArticle(ctx context.Context, in *contentpb.ArticlePublishReq, opts ...grpc.CallOption) (*contentpb.ArticlePublishRes, error) {
	return nil, errors.New("unexpected PublishArticle call")
}

func (f *fakeContentService) PublishVideo(ctx context.Context, in *contentpb.VideoPublishReq, opts ...grpc.CallOption) (*contentpb.VideoPublishRes, error) {
	return nil, errors.New("unexpected PublishVideo call")
}

func (f *fakeContentService) BackfillFollowInbox(ctx context.Context, in *contentpb.BackfillFollowInboxReq, opts ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error) {
	if f.backfillFunc == nil {
		return nil, errors.New("unexpected BackfillFollowInbox call")
	}
	return f.backfillFunc(ctx, in, opts...)
}

func (f *fakeContentService) GetUploadCredentials(context.Context, *contentpb.GetUploadCredentialsReq, ...grpc.CallOption) (*contentpb.GetUploadCredentialsRes, error) {
	return nil, errors.New("unexpected GetUploadCredentials call")
}

func (f *fakeContentService) GetContentDetail(context.Context, *contentpb.GetContentDetailReq, ...grpc.CallOption) (*contentpb.GetContentDetailRes, error) {
	return nil, errors.New("unexpected GetContentDetail call")
}

func (f *fakeContentService) EditArticle(context.Context, *contentpb.EditArticleReq, ...grpc.CallOption) (*contentpb.EditArticleRes, error) {
	return nil, errors.New("unexpected EditArticle call")
}

func (f *fakeContentService) EditVideo(context.Context, *contentpb.EditVideoReq, ...grpc.CallOption) (*contentpb.EditVideoRes, error) {
	return nil, errors.New("unexpected EditVideo call")
}

func (f *fakeContentService) DeleteContent(context.Context, *contentpb.DeleteContentReq, ...grpc.CallOption) (*contentpb.DeleteContentRes, error) {
	return nil, errors.New("unexpected DeleteContent call")
}

func newFollowTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedFollow{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestFollowFlow(t *testing.T) {
	db := newFollowTestDB(t)
	backfillCh := make(chan *contentpb.BackfillFollowInboxReq, 1)

	svcCtx := &svc.ServiceContext{
		MysqlDb: db,
		ContentRpc: &fakeContentService{
			backfillFunc: func(ctx context.Context, in *contentpb.BackfillFollowInboxReq, _ ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error) {
				backfillCh <- in
				return &contentpb.BackfillFollowInboxRes{AddedCount: 2}, nil
			},
		},
	}

	followLogic := NewFollowUserLogic(context.Background(), svcCtx)
	unfollowLogic := NewUnfollowUserLogic(context.Background(), svcCtx)
	listLogic := NewListFolloweesLogic(context.Background(), svcCtx)
	summaryLogic := NewGetFollowSummaryLogic(context.Background(), svcCtx)

	var (
		userID       int64 = 1001
		followUserID int64 = 2002
	)

	resp, err := followLogic.FollowUser(&interaction.FollowUserReq{
		UserId:       userID,
		FollowUserId: followUserID,
	})
	if err != nil {
		t.Fatalf("FollowUser returned error: %v", err)
	}
	if !resp.GetIsFollowed() {
		t.Fatal("FollowUser returned is_followed=false, want true")
	}

	select {
	case req := <-backfillCh:
		if req.GetFollowerId() != userID || req.GetFolloweeId() != followUserID || req.GetLimit() != backfillFollowInboxLimit {
			t.Fatalf("backfill req = %+v, want follower=%d followee=%d limit=%d", req, userID, followUserID, backfillFollowInboxLimit)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async backfill call")
	}

	var row model.ZfeedFollow
	if err := db.Where("user_id = ? AND follow_user_id = ?", userID, followUserID).Take(&row).Error; err != nil {
		t.Fatalf("query follow row after follow: %v", err)
	}
	if row.Status != 10 {
		t.Fatalf("follow row status after follow = %d, want 10", row.Status)
	}

	listResp, err := listLogic.ListFollowees(&interaction.ListFolloweesReq{
		UserId:   userID,
		Cursor:   0,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListFollowees returned error: %v", err)
	}
	if len(listResp.GetFollowUserIds()) != 1 || listResp.GetFollowUserIds()[0] != followUserID {
		t.Fatalf("followees = %v, want [%d]", listResp.GetFollowUserIds(), followUserID)
	}

	summaryResp, err := summaryLogic.GetFollowSummary(&interaction.GetFollowSummaryReq{
		UserId:   followUserID,
		ViewerId: &userID,
	})
	if err != nil {
		t.Fatalf("GetFollowSummary returned error: %v", err)
	}
	if summaryResp.GetFollowerCount() != 1 || !summaryResp.GetIsFollowing() {
		t.Fatalf("follow summary after follow = %+v, want follower_count=1 is_following=true", summaryResp)
	}

	resp2, err := unfollowLogic.UnfollowUser(&interaction.UnfollowUserReq{
		UserId:       userID,
		FollowUserId: followUserID,
	})
	if err != nil {
		t.Fatalf("UnfollowUser returned error: %v", err)
	}
	if resp2.GetIsFollowed() {
		t.Fatal("UnfollowUser returned is_followed=true, want false")
	}

	if err := db.Where("user_id = ? AND follow_user_id = ?", userID, followUserID).Take(&row).Error; err != nil {
		t.Fatalf("query follow row after unfollow: %v", err)
	}
	if row.Status != 20 {
		t.Fatalf("follow row status after unfollow = %d, want 20", row.Status)
	}

	summaryResp, err = summaryLogic.GetFollowSummary(&interaction.GetFollowSummaryReq{
		UserId:   followUserID,
		ViewerId: &userID,
	})
	if err != nil {
		t.Fatalf("GetFollowSummary after unfollow returned error: %v", err)
	}
	if summaryResp.GetFollowerCount() != 0 || summaryResp.GetIsFollowing() {
		t.Fatalf("follow summary after unfollow = %+v, want follower_count=0 is_following=false", summaryResp)
	}
}

func TestFollowTrace(t *testing.T) {
	db := newFollowTestDB(t)
	backfillCtxCh := make(chan context.Context, 1)
	gztracetest.NewInMemoryExporter(t)

	svcCtx := &svc.ServiceContext{
		MysqlDb: db,
		ContentRpc: &fakeContentService{
			backfillFunc: func(ctx context.Context, in *contentpb.BackfillFollowInboxReq, _ ...grpc.CallOption) (*contentpb.BackfillFollowInboxRes, error) {
				backfillCtxCh <- ctx
				return &contentpb.BackfillFollowInboxRes{AddedCount: 1}, nil
			},
		},
	}

	parentCtx, parentSpan := otel.Tracer("follow-test").Start(context.Background(), "follow-request")
	defer parentSpan.End()

	logic := NewFollowUserLogic(parentCtx, svcCtx)
	_, err := logic.FollowUser(&interaction.FollowUserReq{
		UserId:       3001,
		FollowUserId: 4002,
	})
	if err != nil {
		t.Fatalf("FollowUser returned error: %v", err)
	}

	select {
	case backfillCtx := <-backfillCtxCh:
		parentSpanCtx := oteltrace.SpanContextFromContext(parentCtx)
		backfillSpanCtx := oteltrace.SpanContextFromContext(backfillCtx)
		if !backfillSpanCtx.IsValid() {
			t.Fatal("backfill context does not contain a valid trace span")
		}
		if backfillSpanCtx.TraceID() != parentSpanCtx.TraceID() {
			t.Fatalf("backfill trace id = %s, want %s", backfillSpanCtx.TraceID(), parentSpanCtx.TraceID())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async backfill trace context")
	}
}
