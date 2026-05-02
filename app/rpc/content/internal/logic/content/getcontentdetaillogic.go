package contentlogic

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
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

const (
	defaultContentDetailCountTimeout  = 200 * time.Millisecond
	defaultContentDetailAuthorTimeout = 150 * time.Millisecond
	defaultContentDetailStateTimeout  = 150 * time.Millisecond
)

const (
	contentTypeArticle       = int32(content.ContentType_CONTENT_TYPE_ARTICLE)
	contentTypeVideo         = int32(content.ContentType_CONTENT_TYPE_VIDEO)
	contentStatusPublish     = int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)
	contentVisibilityPublic  = int32(content.Visibility_VISIBILITY_PUBLIC)
	contentVisibilityPrivate = int32(content.Visibility_VISIBILITY_PRIVATE)
)

type ContentDetailLogic struct {
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

type authorResult struct {
	name   string
	avatar string
	ok     bool
}

type countResult struct {
	likeCount     int64
	favoriteCount int64
	commentCount  int64
	ok            bool
}

type likeStateResult struct {
	isLiked   bool
	likeCount int64
	ok        bool
}

type favoriteStateResult struct {
	isFavorited   bool
	favoriteCount int64
	ok            bool
}

type followStateResult struct {
	isFollowing bool
	ok          bool
}

type viewerStateResult struct {
	like     likeStateResult
	favorite favoriteStateResult
	follow   followStateResult
}

func NewGetContentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ContentDetailLogic {
	return &ContentDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ContentDetailLogic) GetContentDetail(in *content.GetContentDetailReq) (*content.GetContentDetailRes, error) {
	if in == nil || in.GetContentId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	viewerID := int64(0)
	if in.ViewerId != nil && in.GetViewerId() > 0 {
		viewerID = in.GetViewerId()
	}

	detail, err := l.buildBaseDetail(in.GetContentId(), viewerID)
	if err != nil {
		return nil, err
	}

	authorRes, countRes, viewerRes := l.enrichDetail(detail, viewerID)
	l.applyAuthor(detail, authorRes)
	l.applyCounts(detail, countRes)
	l.applyViewerState(detail, viewerRes, viewerID)

	return &content.GetContentDetailRes{Detail: detail}, nil
}

func (l *ContentDetailLogic) buildBaseDetail(contentID, viewerID int64) (*content.ContentDetail, error) {
	contentRow, err := l.queryContent(contentID, viewerID)
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

	return detail, nil
}

func (l *ContentDetailLogic) enrichDetail(detail *content.ContentDetail, viewerID int64) (authorResult, countResult, viewerStateResult) {
	var (
		authorRes authorResult
		countRes  countResult
		viewerRes viewerStateResult
		eg        errgroup.Group
	)

	eg.Go(func() error {
		authorRes = l.queryAuthor(detail.GetAuthorId())
		return nil
	})

	eg.Go(func() error {
		countRes = l.queryCounts(detail.GetContentId())
		return nil
	})

	if viewerID > 0 {
		eg.Go(func() error {
			viewerRes = l.queryViewerState(
				viewerID,
				detail.GetContentId(),
				detail.GetAuthorId(),
				detail.GetContentType(),
			)
			return nil
		})
	}

	_ = eg.Wait()
	return authorRes, countRes, viewerRes
}

func (l *ContentDetailLogic) queryContent(contentID int64, viewerID int64) (*contentBaseRow, error) {
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

func (l *ContentDetailLogic) queryArticle(contentID int64) (*contentArticleRow, error) {
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

func (l *ContentDetailLogic) queryVideo(contentID int64) (*contentVideoRow, error) {
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

func (l *ContentDetailLogic) queryAuthor(authorID int64) authorResult {
	if authorID <= 0 || l.svcCtx == nil || l.svcCtx.UserRpc == nil {
		return authorResult{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailAuthorTimeout)
	defer cancel()

	resp, err := l.svcCtx.UserRpc.GetUserProfile(ctx, &userservice.GetUserProfileReq{UserId: authorID})
	if err != nil {
		l.Errorf("query author profile failed, author_id=%d, err=%v", authorID, err)
		return authorResult{}
	}
	if resp.GetUserProfile() == nil {
		return authorResult{}
	}

	return authorResult{
		name:   resp.GetUserProfile().GetNickname(),
		avatar: resp.GetUserProfile().GetAvatar(),
		ok:     true,
	}
}

func (l *ContentDetailLogic) applyAuthor(detail *content.ContentDetail, res authorResult) {
	if detail == nil || !res.ok {
		return
	}
	detail.AuthorName = res.name
	detail.AuthorAvatar = res.avatar
}

func (l *ContentDetailLogic) queryCounts(contentID int64) countResult {
	if contentID <= 0 || l.svcCtx == nil || l.svcCtx.CountRpc == nil {
		return countResult{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailCountTimeout)
	defer cancel()

	resp, err := l.svcCtx.CountRpc.BatchGetCount(ctx, &counterservice.BatchGetCountReq{
		Keys: []*counterservice.CountKey{
			{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
			{BizType: count.BizType_FAVORITE, TargetType: count.TargetType_CONTENT, TargetId: contentID},
			{BizType: count.BizType_COMMENT, TargetType: count.TargetType_CONTENT, TargetId: contentID},
		},
	})
	if err != nil {
		l.Errorf("query content counts failed, content_id=%d, err=%v", contentID, err)
		return countResult{}
	}

	res := countResult{ok: true}
	for _, item := range resp.GetItems() {
		if item == nil || item.GetKey() == nil {
			continue
		}
		switch item.GetKey().GetBizType() {
		case count.BizType_LIKE:
			res.likeCount = item.GetValue()
		case count.BizType_FAVORITE:
			res.favoriteCount = item.GetValue()
		case count.BizType_COMMENT:
			res.commentCount = item.GetValue()
		}
	}
	return res
}

func (l *ContentDetailLogic) applyCounts(detail *content.ContentDetail, res countResult) {
	if detail == nil || !res.ok {
		return
	}
	detail.LikeCount = res.likeCount
	detail.FavoriteCount = res.favoriteCount
	detail.CommentCount = res.commentCount
}

func (l *ContentDetailLogic) queryViewerState(viewerID, contentID, authorID int64, contentType content.ContentType) viewerStateResult {
	if viewerID <= 0 {
		return viewerStateResult{}
	}

	scene, ok := sceneByContentType(contentType)
	if !ok {
		return viewerStateResult{}
	}

	var (
		likeRes     likeStateResult
		favoriteRes favoriteStateResult
		followRes   followStateResult
		eg          errgroup.Group
	)

	eg.Go(func() error {
		likeRes = l.queryLikeState(viewerID, contentID, scene)
		return nil
	})

	eg.Go(func() error {
		favoriteRes = l.queryFavoriteState(viewerID, contentID, scene)
		return nil
	})

	eg.Go(func() error {
		followRes = l.queryFollowState(viewerID, authorID)
		return nil
	})

	_ = eg.Wait()

	return viewerStateResult{
		like:     likeRes,
		favorite: favoriteRes,
		follow:   followRes,
	}
}

func (l *ContentDetailLogic) queryLikeState(viewerID, contentID int64, scene interactionpb.Scene) likeStateResult {
	if l.svcCtx == nil || l.svcCtx.LikeRpc == nil {
		return likeStateResult{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailStateTimeout)
	defer cancel()

	resp, err := l.svcCtx.LikeRpc.QueryLikeInfo(ctx, &likeservice.QueryLikeInfoReq{
		UserId:    viewerID,
		ContentId: contentID,
		Scene:     scene,
	})
	if err != nil {
		l.Errorf("query like info failed, viewer_id=%d, content_id=%d, err=%v", viewerID, contentID, err)
		return likeStateResult{}
	}
	if resp == nil {
		return likeStateResult{}
	}

	return likeStateResult{
		isLiked:   resp.GetIsLiked(),
		likeCount: resp.GetLikeCount(),
		ok:        true,
	}
}

func (l *ContentDetailLogic) queryFavoriteState(viewerID, contentID int64, scene interactionpb.Scene) favoriteStateResult {
	if l.svcCtx == nil || l.svcCtx.FavoriteRpc == nil {
		return favoriteStateResult{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailStateTimeout)
	defer cancel()

	resp, err := l.svcCtx.FavoriteRpc.QueryFavoriteInfo(ctx, &favoriteservice.QueryFavoriteInfoReq{
		UserId:    viewerID,
		ContentId: contentID,
		Scene:     scene,
	})
	if err != nil {
		l.Errorf("query favorite info failed, viewer_id=%d, content_id=%d, err=%v", viewerID, contentID, err)
		return favoriteStateResult{}
	}
	if resp == nil {
		return favoriteStateResult{}
	}

	return favoriteStateResult{
		isFavorited:   resp.GetIsFavorited(),
		favoriteCount: resp.GetFavoriteCount(),
		ok:            true,
	}
}

func (l *ContentDetailLogic) queryFollowState(viewerID, authorID int64) followStateResult {
	if l.svcCtx == nil || l.svcCtx.FollowRpc == nil || authorID <= 0 {
		return followStateResult{}
	}

	ctx, cancel := context.WithTimeout(l.ctx, defaultContentDetailStateTimeout)
	defer cancel()

	resp, err := l.svcCtx.FollowRpc.GetFollowSummary(ctx, &followservice.GetFollowSummaryReq{
		UserId:   authorID,
		ViewerId: &viewerID,
	})
	if err != nil {
		l.Errorf("query follow summary failed, viewer_id=%d, author_id=%d, err=%v", viewerID, authorID, err)
		return followStateResult{}
	}
	if resp == nil {
		return followStateResult{}
	}

	return followStateResult{
		isFollowing: resp.GetIsFollowing(),
		ok:          true,
	}
}

func (l *ContentDetailLogic) applyViewerState(detail *content.ContentDetail, res viewerStateResult, viewerID int64) {
	if detail == nil || viewerID <= 0 {
		return
	}

	if res.like.ok {
		detail.IsLiked = res.like.isLiked
		if res.like.likeCount > 0 {
			detail.LikeCount = res.like.likeCount
		}
	}

	if res.favorite.ok {
		detail.IsFavorited = res.favorite.isFavorited
		if res.favorite.favoriteCount > 0 {
			detail.FavoriteCount = res.favorite.favoriteCount
		}
	}

	if res.follow.ok {
		detail.IsFollowingAuthor = res.follow.isFollowing
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
