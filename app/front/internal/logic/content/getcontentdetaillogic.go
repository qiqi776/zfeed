// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"errors"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/app/rpc/count/count"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetContentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetContentDetailLogic {
	return &GetContentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetContentDetailLogic) GetContentDetail(req *types.GetContentDetailReq) (resp *types.GetContentDetailRes, err error) {
	if req == nil || req.ContentId == nil || *req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	contentRow, err := l.queryContent(*req.ContentId)
	if err != nil {
		return nil, err
	}

	author, err := l.queryAuthor(contentRow.UserID)
	if err != nil {
		return nil, err
	}

	detail := types.ContentDetail{
		ContentId:     contentRow.ID,
		ContentType:   contentRow.ContentType,
		AuthorId:      contentRow.UserID,
		AuthorName:    author.Nickname,
		AuthorAvatar:  author.Avatar,
		PublishedAt:   toUnix(contentRow.PublishedAt),
		LikeCount:     contentRow.LikeCount,
		FavoriteCount: contentRow.FavoriteCount,
		CommentCount:  contentRow.CommentCount,
	}

	switch contentRow.ContentType {
	case contentTypeArticle:
		article, queryErr := l.queryArticle(contentRow.ID)
		if queryErr != nil {
			return nil, queryErr
		}
		detail.Title = article.Title
		detail.Description = valueOrEmpty(article.Description)
		detail.CoverUrl = article.Cover
		detail.ArticleContent = article.Content
	case contentTypeVideo:
		video, queryErr := l.queryVideo(contentRow.ID)
		if queryErr != nil {
			return nil, queryErr
		}
		detail.Title = video.Title
		detail.Description = valueOrEmpty(video.Description)
		detail.CoverUrl = video.CoverURL
		detail.VideoUrl = video.OriginURL
		detail.VideoDuration = video.Duration
	default:
		return nil, errorx.NewBadRequest("内容类型错误")
	}

	if counts, countErr := l.queryCounts(contentRow.ID); countErr != nil {
		l.Errorf("query content counts failed, content_id=%d, err=%v", contentRow.ID, countErr)
	} else {
		detail.LikeCount = counts.LikeCount
		detail.FavoriteCount = counts.FavoriteCount
		detail.CommentCount = counts.CommentCount
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	if viewerID > 0 {
		isLiked, likeErr := l.queryIsLiked(viewerID, contentRow.ID)
		if likeErr != nil {
			return nil, likeErr
		}
		isFavorited, favoriteErr := l.queryIsFavorited(viewerID, contentRow.ID)
		if favoriteErr != nil {
			return nil, favoriteErr
		}
		isFollowing, followErr := l.queryIsFollowing(viewerID, contentRow.UserID)
		if followErr != nil {
			return nil, followErr
		}

		detail.IsLiked = isLiked
		detail.IsFavorited = isFavorited
		detail.IsFollowingAuthor = isFollowing
	}

	return &types.GetContentDetailRes{Detail: detail}, nil
}

const (
	contentTypeArticle      = 10
	contentTypeVideo        = 20
	contentStatusPublish    = 30
	contentVisibilityPublic = 10
	likeStatusActive        = 10
	favoriteStatusActive    = 10
	followStatusActive      = 10
)

var defaultContentCountTimeout = 200 * time.Millisecond

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

type contentAuthorRow struct {
	ID       int64  `gorm:"column:id"`
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
}

type contentCounts struct {
	LikeCount     int64
	FavoriteCount int64
	CommentCount  int64
}

func (l *GetContentDetailLogic) queryContent(contentID int64) (*contentBaseRow, error) {
	var row contentBaseRow
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_content").
		Select("id", "user_id", "content_type", "like_count", "favorite_count", "comment_count", "published_at").
		Where("id = ? AND status = ? AND visibility = ? AND is_deleted = 0", contentID, contentStatusPublish, contentVisibilityPublic).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryArticle(contentID int64) (*contentArticleRow, error) {
	var row contentArticleRow
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_article").
		Where("content_id = ? AND is_deleted = 0", contentID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryVideo(contentID int64) (*contentVideoRow, error) {
	var row contentVideoRow
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_video").
		Where("content_id = ? AND is_deleted = 0", contentID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.NewNotFound("内容不存在")
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryAuthor(userID int64) (*contentAuthorRow, error) {
	var row contentAuthorRow
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Select("id", "nickname", "avatar").
		Where("id = ? AND is_deleted = 0", userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &contentAuthorRow{
				ID:       userID,
				Nickname: "用户",
			}, nil
		}
		return nil, err
	}
	return &row, nil
}

func (l *GetContentDetailLogic) queryCounts(contentID int64) (*contentCounts, error) {
	timeout := defaultContentCountTimeout
	if l.svcCtx.Config.CountRPCTimeoutMs > 0 {
		timeout = time.Duration(l.svcCtx.Config.CountRPCTimeoutMs) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(l.ctx, timeout)
	defer cancel()

	resp, err := l.svcCtx.CountRpc.BatchGetCount(ctx, &count.BatchGetCountReq{
		Keys: []*count.CountKey{
			{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
			{BizType: count.BizType_FAVORITE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
			{BizType: count.BizType_COMMENT, TargetType: count.TargetType_CONTENT, TargetId: contentID},
		},
	})
	if err != nil {
		return nil, err
	}

	result := &contentCounts{}
	for _, item := range resp.GetItems() {
		if item == nil || item.GetKey() == nil {
			continue
		}
		switch item.GetKey().GetBizType() {
		case count.BizType_LIKE:
			result.LikeCount = item.GetValue()
		case count.BizType_FAVORITE:
			result.FavoriteCount = item.GetValue()
		case count.BizType_COMMENT:
			result.CommentCount = item.GetValue()
		}
	}
	return result, nil
}

func (l *GetContentDetailLogic) queryIsLiked(userID int64, contentID int64) (bool, error) {
	var countValue int64
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_like").
		Where("user_id = ? AND content_id = ? AND status = ? AND is_deleted = 0", userID, contentID, likeStatusActive).
		Count(&countValue).Error
	if err != nil {
		return false, err
	}
	return countValue > 0, nil
}

func (l *GetContentDetailLogic) queryIsFavorited(userID int64, contentID int64) (bool, error) {
	var countValue int64
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_favorite").
		Where("user_id = ? AND content_id = ? AND status = ?", userID, contentID, favoriteStatusActive).
		Count(&countValue).Error
	if err != nil {
		return false, err
	}
	return countValue > 0, nil
}

func (l *GetContentDetailLogic) queryIsFollowing(viewerID int64, authorID int64) (bool, error) {
	if viewerID <= 0 || authorID <= 0 || viewerID == authorID {
		return false, nil
	}

	var countValue int64
	err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_follow").
		Where("user_id = ? AND follow_user_id = ? AND status = ? AND is_deleted = 0", viewerID, authorID, followStatusActive).
		Count(&countValue).Error
	if err != nil {
		return false, err
	}
	return countValue > 0, nil
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
