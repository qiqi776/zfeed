package logic

import (
	"context"
	"errors"
	"time"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const defaultContentDetailCountTimeout = 200 * time.Millisecond

const (
	contentTypeArticle       = int32(content.ContentType_CONTENT_TYPE_ARTICLE)
	contentTypeVideo         = int32(content.ContentType_CONTENT_TYPE_VIDEO)
	contentStatusPublish     = int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)
	contentVisibilityPublic  = int32(content.Visibility_VISIBILITY_PUBLIC)
	contentVisibilityPrivate = int32(content.Visibility_VISIBILITY_PRIVATE)
)

type GetContentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type contentBaseRow struct {
	ID            int64      `gorm:"column:id"`
	UserID        int64      `gorm:"column:user_id"`
	ContentType   int32      `gorm:"column:content_type"`
	LikeCount     int64      `gorm:"column:like_count"`
	FavoriteCount int64      `gorm:"column:favorite_count"`
	CommentCount  int64      `gorm:"column:comment_count"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
}

type contentArticleRow struct {
	ContentID   int64   `gorm:"column:content_id"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	Cover       string  `gorm:"column:cover"`
	Content     string  `gorm:"column:content"`
}

type contentVideoRow struct {
	ContentID   int64   `gorm:"column:content_id"`
	Title       string  `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	OriginURL   string  `gorm:"column:origin_url"`
	CoverURL    string  `gorm:"column:cover_url"`
	Duration    int32   `gorm:"column:duration"`
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContentDetailLogic {
	return &GetContentDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetContentDetailLogic) GetContentDetail(in *content.GetContentDetailReq) (*content.GetContentDetailRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	viewerID := int64(0)
	if in.ViewerId != nil && in.GetViewerId() > 0 {
		viewerID = in.GetViewerId()
	}

	contentRow, err := l.queryContent(in.GetContentId(), viewerID)
	if err != nil {
		return nil, err
	}

	detail := &content.ContentDetail{
		ContentId:     contentRow.ID,
		ContentType:   content.ContentType(contentRow.ContentType),
		AuthorId:      contentRow.UserID,
		AuthorName:    "用户",
		PublishedAt:   toUnix(contentRow.PublishedAt),
		LikeCount:     contentRow.LikeCount,
		FavoriteCount: contentRow.FavoriteCount,
		CommentCount:  contentRow.CommentCount,
	}

	switch detail.GetContentType() {
	case content.ContentType_CONTENT_TYPE_ARTICLE:
		article, err := l.queryArticle(contentRow.ID)
		if err != nil {
			return nil, err
		}
		detail.Title = article.Title
		detail.Description = valueOrEmpty(article.Description)
		detail.CoverUrl = article.Cover
		detail.ArticleContent = article.Content
	case content.ContentType_CONTENT_TYPE_VIDEO:
		video, err := l.queryVideo(contentRow.ID)
		if err != nil {
			return nil, err
		}
		detail.Title = video.Title
		detail.Description = valueOrEmpty(video.Description)
		detail.CoverUrl = video.CoverURL
		detail.VideoUrl = video.OriginURL
		detail.VideoDuration = video.Duration
	default:
		return nil, errorx.NewBadRequest("内容类型错误")
	}

	l.fillAuthor(detail)
	l.fillCounts(detail)
	if viewerID > 0 {
		l.fillViewerState(detail, viewerID)
	}

	return &content.GetContentDetailRes{Detail: detail}, nil
}

func (l *GetContentDetailLogic) queryContent(contentID int64, viewerID int64) (*contentBaseRow, error) {
	var row contentBaseRow
	query := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_content").
		Select("id", "user_id", "content_type", "like_count", "favorite_count", "comment_count", "published_at").
		Where("id = ? AND status = ? AND is_deleted = 0", contentID, int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED))
	if viewerID > 0 {
		query = query.Where(
			"(visibility = ? OR (visibility = ? AND user_id = ?))",
			int32(content.Visibility_VISIBILITY_PUBLIC),
			int32(content.Visibility_VISIBILITY_PRIVATE),
			viewerID,
		)
	} else {
		query = query.Where("visibility = ?", int32(content.Visibility_VISIBILITY_PUBLIC))
	}

	if err := query.Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryArticle(contentID int64) (*contentArticleRow, error) {
	var row contentArticleRow
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_article").
		Where("content_id = ? AND is_deleted = 0", contentID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryVideo(contentID int64) (*contentVideoRow, error) {
	var row contentVideoRow
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_video").
		Where("content_id = ? AND is_deleted = 0", contentID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) fillAuthor(detail *content.ContentDetail) {
	if detail == nil || detail.GetAuthorId() <= 0 || l.svcCtx == nil || l.svcCtx.UserRpc == nil {
		return
	}

	resp, err := l.svcCtx.UserRpc.GetUserProfile(l.ctx, &userservice.GetUserProfileReq{UserId: detail.GetAuthorId()})
	if err != nil {
		l.Errorf("query author profile failed, author_id=%d, err=%v", detail.GetAuthorId(), err)
		return
	}
	if resp.GetUserProfile() == nil {
		return
	}

	detail.AuthorName = resp.GetUserProfile().GetNickname()
	detail.AuthorAvatar = resp.GetUserProfile().GetAvatar()
}

func (l *GetContentDetailLogic) fillCounts(detail *content.ContentDetail) {
	if detail == nil || l.svcCtx == nil || l.svcCtx.CountRpc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailCountTimeout)
	defer cancel()

	resp, err := l.svcCtx.CountRpc.BatchGetCount(ctx, &counterservice.BatchGetCountReq{
		Keys: []*counterservice.CountKey{
			{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: detail.GetContentId()},
			{BizType: count.BizType_FAVORITE, TargetType: count.TargetType_CONTENT, TargetId: detail.GetContentId()},
			{BizType: count.BizType_COMMENT, TargetType: count.TargetType_CONTENT, TargetId: detail.GetContentId()},
		},
	})
	if err != nil {
		l.Errorf("query content counts failed, content_id=%d, err=%v", detail.GetContentId(), err)
		return
	}

	for _, item := range resp.GetItems() {
		if item == nil || item.GetKey() == nil {
			continue
		}
		switch item.GetKey().GetBizType() {
		case count.BizType_LIKE:
			detail.LikeCount = item.GetValue()
		case count.BizType_FAVORITE:
			detail.FavoriteCount = item.GetValue()
		case count.BizType_COMMENT:
			detail.CommentCount = item.GetValue()
		}
	}
}

func (l *GetContentDetailLogic) fillViewerState(detail *content.ContentDetail, viewerID int64) {
	if detail == nil || viewerID <= 0 {
		return
	}

	scene, ok := sceneByContentType(detail.GetContentType())
	if !ok {
		return
	}

	if l.svcCtx != nil && l.svcCtx.LikeRpc != nil {
		resp, err := l.svcCtx.LikeRpc.QueryLikeInfo(l.ctx, &likeservice.QueryLikeInfoReq{
			UserId:    viewerID,
			ContentId: detail.GetContentId(),
			Scene:     scene,
		})
		if err != nil {
			l.Errorf("query like info failed, viewer_id=%d, content_id=%d, err=%v", viewerID, detail.GetContentId(), err)
		} else if resp != nil {
			detail.IsLiked = resp.GetIsLiked()
			if resp.GetLikeCount() > 0 {
				detail.LikeCount = resp.GetLikeCount()
			}
		}
	}

	if l.svcCtx != nil && l.svcCtx.FavoriteRpc != nil {
		resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteInfo(l.ctx, &favoriteservice.QueryFavoriteInfoReq{
			UserId:    viewerID,
			ContentId: detail.GetContentId(),
			Scene:     scene,
		})
		if err != nil {
			l.Errorf("query favorite info failed, viewer_id=%d, content_id=%d, err=%v", viewerID, detail.GetContentId(), err)
		} else if resp != nil {
			detail.IsFavorited = resp.GetIsFavorited()
			if resp.GetFavoriteCount() > 0 {
				detail.FavoriteCount = resp.GetFavoriteCount()
			}
		}
	}

	if l.svcCtx != nil && l.svcCtx.FollowRpc != nil {
		resp, err := l.svcCtx.FollowRpc.GetFollowSummary(l.ctx, &followservice.GetFollowSummaryReq{
			UserId:   detail.GetAuthorId(),
			ViewerId: &viewerID,
		})
		if err != nil {
			l.Errorf("query follow summary failed, viewer_id=%d, author_id=%d, err=%v", viewerID, detail.GetAuthorId(), err)
		} else if resp != nil {
			detail.IsFollowingAuthor = resp.GetIsFollowing()
		}
	}
}

func sceneByContentType(contentType content.ContentType) (interactionpb.Scene, bool) {
	switch contentType {
	case content.ContentType_CONTENT_TYPE_ARTICLE:
		return interactionpb.Scene_ARTICLE, true
	case content.ContentType_CONTENT_TYPE_VIDEO:
		return interactionpb.Scene_VIDEO, true
	default:
		return interactionpb.Scene_SCENE_UNKNOWN, false
	}
}

func toUnix(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
