package logic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/svc"
	likeservice "zfeed/app/rpc/interaction/client/likeservice"
)

var _ likeservice.LikeService = (*fakeLikeService)(nil)

type fakeLikeService struct {
	batchQueryIsLikedFunc func(ctx context.Context, in *likeservice.BatchQueryIsLikedReq, opts ...grpc.CallOption) (*likeservice.BatchQueryIsLikedRes, error)
}

func (f *fakeLikeService) Like(context.Context, *likeservice.LikeReq, ...grpc.CallOption) (*likeservice.LikeRes, error) {
	return nil, errors.New("unexpected Like call")
}

func (f *fakeLikeService) Unlike(context.Context, *likeservice.UnlikeReq, ...grpc.CallOption) (*likeservice.UnlikeRes, error) {
	return nil, errors.New("unexpected Unlike call")
}

func (f *fakeLikeService) QueryLikeInfo(context.Context, *likeservice.QueryLikeInfoReq, ...grpc.CallOption) (*likeservice.QueryLikeInfoRes, error) {
	return nil, errors.New("unexpected QueryLikeInfo call")
}

func (f *fakeLikeService) BatchQueryLikeInfo(context.Context, *likeservice.BatchQueryLikeInfoReq, ...grpc.CallOption) (*likeservice.BatchQueryLikeInfoRes, error) {
	return nil, errors.New("unexpected BatchQueryLikeInfo call")
}

func (f *fakeLikeService) BatchQueryIsLiked(ctx context.Context, in *likeservice.BatchQueryIsLikedReq, opts ...grpc.CallOption) (*likeservice.BatchQueryIsLikedRes, error) {
	if f.batchQueryIsLikedFunc == nil {
		return nil, errors.New("unexpected BatchQueryIsLiked call")
	}
	return f.batchQueryIsLikedFunc(ctx, in, opts...)
}

func newFeedBuilderTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedContent{}, &model.ZfeedArticle{}, &model.ZfeedVideo{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestFeedItemBuilderBuildContentItemsIncludesLikedState(t *testing.T) {
	db := newFeedBuilderTestDB(t)

	if err := db.Create(&model.ZfeedArticle{
		ContentID: 1001,
		Title:     "article-1001",
		Cover:     "cover-a-1001",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}
	if err := db.Create(&model.ZfeedVideo{
		ContentID: 1002,
		Title:     "video-1002",
		CoverURL:  "cover-v-1002",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed video: %v", err)
	}

	builder := NewFeedItemBuilder(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		LikeRpc: &fakeLikeService{
			batchQueryIsLikedFunc: func(ctx context.Context, in *likeservice.BatchQueryIsLikedReq, opts ...grpc.CallOption) (*likeservice.BatchQueryIsLikedRes, error) {
				return &likeservice.BatchQueryIsLikedRes{
					IsLikedInfos: []*likeservice.IsLikedInfo{
						{ContentId: 1001, IsLiked: true},
						{ContentId: 1002, IsLiked: false},
					},
				}, nil
			},
		},
	})

	viewerID := int64(42)
	items, err := builder.BuildContentItems([]*model.ZfeedContent{
		{ID: 1001, UserID: 2001, ContentType: int32(contentpb.ContentType_CONTENT_TYPE_ARTICLE)},
		{ID: 1002, UserID: 2002, ContentType: int32(contentpb.ContentType_CONTENT_TYPE_VIDEO)},
	}, &viewerID)
	if err != nil {
		t.Fatalf("BuildContentItems returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if !items[0].GetIsLiked() {
		t.Fatal("first item is_liked = false, want true")
	}
	if items[1].GetIsLiked() {
		t.Fatal("second item is_liked = true, want false")
	}
}
