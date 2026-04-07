package server

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/logic"
	"zfeed/app/rpc/content/internal/svc"
)

type FeedServiceServer struct {
	svcCtx *svc.ServiceContext
	content.UnimplementedFeedServiceServer
}

func NewFeedServiceServer(svcCtx *svc.ServiceContext) *FeedServiceServer {
	return &FeedServiceServer{svcCtx: svcCtx}
}

func (s *FeedServiceServer) RecommendFeed(ctx context.Context, in *content.RecommendFeedReq) (*content.RecommendFeedRes, error) {
	l := logic.NewRecommendFeedLogic(ctx, s.svcCtx)
	return l.RecommendFeed(in)
}

func (s *FeedServiceServer) FollowFeed(ctx context.Context, in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	l := logic.NewFollowFeedLogic(ctx, s.svcCtx)
	return l.FollowFeed(in)
}

func (s *FeedServiceServer) UserPublishFeed(ctx context.Context, in *content.UserPublishFeedReq) (*content.UserPublishFeedRes, error) {
	l := logic.NewUserPublishFeedLogic(ctx, s.svcCtx)
	return l.UserPublishFeed(in)
}

func (s *FeedServiceServer) UserFavoriteFeed(ctx context.Context, in *content.UserFavoriteFeedReq) (*content.UserFavoriteFeedRes, error) {
	l := logic.NewUserFavoriteFeedLogic(ctx, s.svcCtx)
	return l.UserFavoriteFeed(in)
}
