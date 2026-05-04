package server

import (
	"context"

	"zfeed/app/rpc/content/content"
	contentlogic "zfeed/app/rpc/content/internal/logic/content"
	"zfeed/app/rpc/content/internal/svc"
)

type ContentServiceServer struct {
	svcCtx *svc.ServiceContext
	content.UnimplementedContentServiceServer
}

func NewContentServiceServer(svcCtx *svc.ServiceContext) *ContentServiceServer {
	return &ContentServiceServer{
		svcCtx: svcCtx,
	}
}

func (s *ContentServiceServer) PublishArticle(ctx context.Context, in *content.ArticlePublishReq) (*content.ArticlePublishRes, error) {
	l := contentlogic.NewPublishArticleLogic(ctx, s.svcCtx)
	return l.PublishArticle(in)
}

func (s *ContentServiceServer) PublishVideo(ctx context.Context, in *content.VideoPublishReq) (*content.VideoPublishRes, error) {
	l := contentlogic.NewPublishVideoLogic(ctx, s.svcCtx)
	return l.PublishVideo(in)
}

func (s *ContentServiceServer) BackfillFollowInbox(ctx context.Context, in *content.BackfillFollowInboxReq) (*content.BackfillFollowInboxRes, error) {
	l := contentlogic.NewBackfillFollowInboxLogic(ctx, s.svcCtx)
	return l.BackfillFollowInbox(in)
}

func (s *ContentServiceServer) CleanupFollowInbox(ctx context.Context, in *content.CleanupFollowInboxReq) (*content.CleanupFollowInboxRes, error) {
	l := contentlogic.NewCleanupFollowInboxLogic(ctx, s.svcCtx)
	return l.CleanupFollowInbox(in)
}

func (s *ContentServiceServer) GetUploadCredentials(ctx context.Context, in *content.GetUploadCredentialsReq) (*content.GetUploadCredentialsRes, error) {
	l := contentlogic.NewGetUploadCredentialsLogic(ctx, s.svcCtx)
	return l.GetUploadCredentials(in)
}

func (s *ContentServiceServer) GetContentDetail(ctx context.Context, in *content.GetContentDetailReq) (*content.GetContentDetailRes, error) {
	l := contentlogic.NewGetContentDetailLogic(ctx, s.svcCtx)
	return l.GetContentDetail(in)
}

func (s *ContentServiceServer) EditArticle(ctx context.Context, in *content.EditArticleReq) (*content.EditArticleRes, error) {
	l := contentlogic.NewEditArticleLogic(ctx, s.svcCtx)
	return l.EditArticle(in)
}

func (s *ContentServiceServer) EditVideo(ctx context.Context, in *content.EditVideoReq) (*content.EditVideoRes, error) {
	l := contentlogic.NewEditVideoLogic(ctx, s.svcCtx)
	return l.EditVideo(in)
}

func (s *ContentServiceServer) DeleteContent(ctx context.Context, in *content.DeleteContentReq) (*content.DeleteContentRes, error) {
	l := contentlogic.NewDeleteContentLogic(ctx, s.svcCtx)
	return l.DeleteContent(in)
}
