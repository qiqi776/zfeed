package contentlogic

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	"zfeed/pkg/errorx"
)

func TestViewerStateDoesNotOverwriteCounts(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestContent{
		ID:            6101,
		UserID:        2101,
		ContentType:   contentTypeArticle,
		Status:        contentStatusPublish,
		Visibility:    contentVisibilityPublic,
		LikeCount:     9,
		FavoriteCount: 7,
		CommentCount:  3,
		PublishedAt:   &now,
		IsDeleted:     0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}
	if err := db.Create(&contentServiceTestArticle{
		ContentID: 6101,
		Title:     "public article",
		Cover:     "cover",
		Content:   "body",
		IsDeleted: 0,
	}).Error; err != nil {
		t.Fatalf("seed article: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		LikeRpc: &stubLikeService{
			queryLikeInfo: func(_ context.Context, in *likeservice.QueryLikeInfoReq, _ ...grpc.CallOption) (*likeservice.QueryLikeInfoRes, error) {
				if in.GetContentId() != 6101 || in.GetUserId() != 3101 {
					t.Fatalf("unexpected like request: %+v", in)
				}
				return &likeservice.QueryLikeInfoRes{
					ContentId: 6101,
					LikeCount: 0,
					IsLiked:   false,
				}, nil
			},
		},
		FavoriteRpc: &stubFavoriteService{
			queryFavoriteInfo: func(_ context.Context, in *favoriteservice.QueryFavoriteInfoReq, _ ...grpc.CallOption) (*favoriteservice.QueryFavoriteInfoRes, error) {
				if in.GetContentId() != 6101 || in.GetUserId() != 3101 {
					t.Fatalf("unexpected favorite request: %+v", in)
				}
				return &favoriteservice.QueryFavoriteInfoRes{
					ContentId:     6101,
					FavoriteCount: 0,
					IsFavorited:   false,
				}, nil
			},
		},
	})

	resp, err := logic.GetContentDetail(&contentpb.GetContentDetailReq{
		ContentId: 6101,
		ViewerId:  int64Ptr(3101),
	})
	if err != nil {
		t.Fatalf("GetContentDetail returned error: %v", err)
	}
	if resp.GetDetail() == nil {
		t.Fatal("expected detail")
	}
	if resp.GetDetail().GetLikeCount() != 9 {
		t.Fatalf("like_count = %d, want 9", resp.GetDetail().GetLikeCount())
	}
	if resp.GetDetail().GetFavoriteCount() != 7 {
		t.Fatalf("favorite_count = %d, want 7", resp.GetDetail().GetFavoriteCount())
	}
	if resp.GetDetail().GetCommentCount() != 3 {
		t.Fatalf("comment_count = %d, want 3", resp.GetDetail().GetCommentCount())
	}
	if resp.GetDetail().GetIsLiked() {
		t.Fatal("expected is_liked false")
	}
	if resp.GetDetail().GetIsFavorited() {
		t.Fatal("expected is_favorited false")
	}
}

func TestUnknownType(t *testing.T) {
	db := newContentServiceTestDB(t)
	now := time.Unix(1_700_000_000, 0)
	if err := db.Create(&contentServiceTestContent{
		ID:          6102,
		UserID:      2102,
		ContentType: 999,
		Status:      contentStatusPublish,
		Visibility:  contentVisibilityPublic,
		PublishedAt: &now,
		IsDeleted:   0,
	}).Error; err != nil {
		t.Fatalf("seed content: %v", err)
	}

	logic := NewGetContentDetailLogic(context.Background(), &svc.ServiceContext{MysqlDb: db})
	_, err := logic.GetContentDetail(&contentpb.GetContentDetailReq{ContentId: 6102})
	if err == nil {
		t.Fatal("expected error for unknown content type")
	}

	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("expected biz error, got %T", err)
	}
	if bizErr.HTTPStatus() != 500 {
		t.Fatalf("status = %d, want 500", bizErr.HTTPStatus())
	}
}
