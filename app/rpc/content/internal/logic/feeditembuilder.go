package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	contentpb "zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	userservice "zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
)

type FeedItemBuilder struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewFeedItemBuilder(ctx context.Context, svcCtx *svc.ServiceContext) *FeedItemBuilder {
	return &FeedItemBuilder{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
		articleRepo: repositories.NewArticleRepository(ctx, svcCtx.MysqlDb),
		videoRepo:   repositories.NewVideoRepository(ctx, svcCtx.MysqlDb),
	}
}

func (b *FeedItemBuilder) LoadContentsByIDs(ids []int64) ([]*model.ZfeedContent, error) {
	contentMap, err := b.contentRepo.BatchGetPublishedByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(b.ctx, err, errorx.NewMsg("查询内容失败"))
	}

	contents := make([]*model.ZfeedContent, 0, len(ids))
	for _, id := range ids {
		if row, ok := contentMap[id]; ok && row != nil {
			contents = append(contents, row)
		}
	}
	return contents, nil
}

func (b *FeedItemBuilder) BuildContentItems(contents []*model.ZfeedContent, viewerID *int64) ([]*contentpb.ContentItem, error) {
	_ = viewerID

	articleMap, videoMap, err := b.buildBriefMaps(contents)
	if err != nil {
		return nil, err
	}
	authorMap, err := b.buildAuthorMap(contents)
	if err != nil {
		return nil, err
	}

	items := make([]*contentpb.ContentItem, 0, len(contents))
	for _, row := range contents {
		if row == nil || row.ID <= 0 {
			continue
		}

		item := &contentpb.ContentItem{
			ContentId:   row.ID,
			ContentType: contentpb.ContentType(row.ContentType),
			AuthorId:    row.UserID,
			LikeCount:   row.LikeCount,
		}
		if row.PublishedAt != nil {
			item.PublishedAt = row.PublishedAt.Unix()
		}

		if author, ok := authorMap[row.UserID]; ok && author != nil {
			item.AuthorName = author.GetNickname()
			item.AuthorAvatar = author.GetAvatar()
		}

		switch contentpb.ContentType(row.ContentType) {
		case contentpb.ContentType_CONTENT_TYPE_ARTICLE:
			if article, ok := articleMap[row.ID]; ok && article != nil {
				item.Title = article.Title
				item.CoverUrl = article.Cover
			}
		case contentpb.ContentType_CONTENT_TYPE_VIDEO:
			if video, ok := videoMap[row.ID]; ok && video != nil {
				item.Title = video.Title
				item.CoverUrl = video.CoverURL
			}
		}

		items = append(items, item)
	}
	return items, nil
}

func ContentItemsToFollowItems(items []*contentpb.ContentItem) []*contentpb.FollowFeedItem {
	result := make([]*contentpb.FollowFeedItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, &contentpb.FollowFeedItem{
			ContentId:    item.GetContentId(),
			ContentType:  item.GetContentType(),
			AuthorId:     item.GetAuthorId(),
			AuthorName:   item.GetAuthorName(),
			AuthorAvatar: item.GetAuthorAvatar(),
			Title:        item.GetTitle(),
			CoverUrl:     item.GetCoverUrl(),
			PublishedAt:  item.GetPublishedAt(),
			IsLiked:      item.GetIsLiked(),
			LikeCount:    item.GetLikeCount(),
		})
	}
	return result
}

func (b *FeedItemBuilder) buildBriefMaps(contents []*model.ZfeedContent) (map[int64]*model.ZfeedArticle, map[int64]*model.ZfeedVideo, error) {
	articleIDs := make([]int64, 0)
	videoIDs := make([]int64, 0)
	for _, row := range contents {
		if row == nil || row.ID <= 0 {
			continue
		}
		switch contentpb.ContentType(row.ContentType) {
		case contentpb.ContentType_CONTENT_TYPE_ARTICLE:
			articleIDs = append(articleIDs, row.ID)
		case contentpb.ContentType_CONTENT_TYPE_VIDEO:
			videoIDs = append(videoIDs, row.ID)
		}
	}

	articleMap, err := b.articleRepo.BatchGetBriefByContentIDs(articleIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(b.ctx, err, errorx.NewMsg("查询文章摘要失败"))
	}
	videoMap, err := b.videoRepo.BatchGetBriefByContentIDs(videoIDs)
	if err != nil {
		return nil, nil, errorx.Wrap(b.ctx, err, errorx.NewMsg("查询视频摘要失败"))
	}
	return articleMap, videoMap, nil
}

func (b *FeedItemBuilder) buildAuthorMap(contents []*model.ZfeedContent) (map[int64]*userservice.UserInfo, error) {
	result := make(map[int64]*userservice.UserInfo)
	if b.svcCtx.UserRpc == nil || len(contents) == 0 {
		return result, nil
	}

	seen := make(map[int64]struct{}, len(contents))
	authorIDs := make([]int64, 0, len(contents))
	for _, row := range contents {
		if row == nil || row.UserID <= 0 {
			continue
		}
		if _, ok := seen[row.UserID]; ok {
			continue
		}
		seen[row.UserID] = struct{}{}
		authorIDs = append(authorIDs, row.UserID)
	}
	if len(authorIDs) == 0 {
		return result, nil
	}

	resp, err := b.svcCtx.UserRpc.BatchGetUser(b.ctx, &userservice.BatchGetUserReq{UserIds: authorIDs})
	if err != nil {
		return nil, errorx.Wrap(b.ctx, err, errorx.NewMsg("查询作者信息失败"))
	}
	if resp == nil {
		return result, nil
	}
	for _, user := range resp.GetUsers() {
		if user == nil || user.GetUserId() <= 0 {
			continue
		}
		result[user.GetUserId()] = user
	}
	return result, nil
}
