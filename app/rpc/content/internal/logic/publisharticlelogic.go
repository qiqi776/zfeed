package logic

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	luautils "zfeed/app/rpc/content/internal/common/utils/lua"
	"zfeed/app/rpc/content/internal/do"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/errorx"
)

type PublishArticleLogic struct {
	ctx             context.Context
	svcCtx          *svc.ServiceContext
	logx.Logger
	contentRepo     repositories.ContentRepository
	articleRepo     repositories.ArticleRepository
}

func NewPublishArticleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishArticleLogic {
	return &PublishArticleLogic{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *PublishArticleLogic) PublishArticle(in *contentpb.ArticlePublishReq) (*contentpb.ArticlePublishRes, error) {
	if in == nil || in.GetUserId() <= 0 || strings.TrimSpace(in.GetTitle()) == "" || strings.TrimSpace(in.GetCover()) == "" || strings.TrimSpace(in.GetContent()) == "" {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetVisibility() == contentpb.Visibility_VISIBILITY_UNKNOWN {
		return nil, errorx.NewMsg("参数错误")
	}

	now := time.Now()
	var contentID int64

	err := l.svcCtx.MysqlDb.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		contentRepo := l.contentRepo.WithTx(tx)
		articleRepo := l.articleRepo.WithTx(tx)

		id, err := contentRepo.CreateContent(&do.ContentDO{
			UserID:        in.GetUserId(),
			ContentType:   int32(contentpb.ContentType_CONTENT_TYPE_ARTICLE),
			Status:        int32(contentpb.ContentStatus_CONTENT_STATUS_PUBLISHED),
			Visibility:    int32(in.GetVisibility()),
			LikeCount:     0,
			FavoriteCount: 0,
			CommentCount:  0,
			PublishedAt:   &now,
			IsDeleted:     0,
			CreatedBy:     in.GetUserId(),
			UpdatedBy:     in.GetUserId(),
		})
		if err != nil {
			return err
		}
		contentID = id

		return articleRepo.CreateArticle(&do.ArticleDO{
			ContentID:   contentID,
			Title:       strings.TrimSpace(in.GetTitle()),
			Description: in.Description,
			Cover:       strings.TrimSpace(in.GetCover()),
			Content:     in.GetContent(),
			IsDeleted:   0,
		})
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("发布文章失败"))
	}

	l.tryUpdateUserPublishZSet(in.GetUserId(), contentID)

	return &contentpb.ArticlePublishRes{
		ContentId: contentID,
	}, nil
}

func (l *PublishArticleLogic) tryUpdateUserPublishZSet(userID, contentID int64) {
	key := redisconsts.BuildUserPublishKey(userID)
	contentIDStr := strconv.FormatInt(contentID, 10)

	_, err := l.svcCtx.Redis.EvalCtx(
		l.ctx,
		luautils.UpdateUserPublishZSetScript,
		[]string{key},
		strconv.Itoa(redisconsts.RedisUserPublishKeepLatestN),
		contentIDStr, contentIDStr,
	)
	if err != nil {
		l.Errorf("update user publish zset failed, user_id=%d, content_id=%d, err=%v", userID, contentID, err)
	}
}
