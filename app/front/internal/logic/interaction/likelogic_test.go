package interaction

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	contentpb "zfeed/app/rpc/content/content"
	interactionpb "zfeed/app/rpc/interaction/interaction"
)

type fakeLikeService struct {
	likeReq *interactionpb.LikeReq
}

func (f *fakeLikeService) Like(
	_ context.Context,
	in *interactionpb.LikeReq,
	_ ...grpc.CallOption,
) (*interactionpb.LikeRes, error) {
	f.likeReq = in
	return &interactionpb.LikeRes{}, nil
}

func (f *fakeLikeService) Unlike(
	context.Context,
	*interactionpb.UnlikeReq,
	...grpc.CallOption,
) (*interactionpb.UnlikeRes, error) {
	return &interactionpb.UnlikeRes{}, nil
}

func (f *fakeLikeService) QueryLikeInfo(
	context.Context,
	*interactionpb.QueryLikeInfoReq,
	...grpc.CallOption,
) (*interactionpb.QueryLikeInfoRes, error) {
	return &interactionpb.QueryLikeInfoRes{}, nil
}

func (f *fakeLikeService) BatchLikeInfo(
	context.Context,
	*interactionpb.BatchLikeInfoReq,
	...grpc.CallOption,
) (*interactionpb.BatchLikeInfoRes, error) {
	return &interactionpb.BatchLikeInfoRes{}, nil
}

func (f *fakeLikeService) BatchIsLiked(
	context.Context,
	*interactionpb.BatchIsLikedReq,
	...grpc.CallOption,
) (*interactionpb.BatchIsLikedRes, error) {
	return &interactionpb.BatchIsLikedRes{}, nil
}

type fakeFavoriteService struct {
	favoriteReq *interactionpb.FavoriteReq
}

func (f *fakeFavoriteService) Favorite(
	_ context.Context,
	in *interactionpb.FavoriteReq,
	_ ...grpc.CallOption,
) (*interactionpb.FavoriteRes, error) {
	f.favoriteReq = in
	return &interactionpb.FavoriteRes{}, nil
}

func (f *fakeFavoriteService) RemoveFavorite(
	context.Context,
	*interactionpb.RemoveFavoriteReq,
	...grpc.CallOption,
) (*interactionpb.RemoveFavoriteRes, error) {
	return &interactionpb.RemoveFavoriteRes{}, nil
}

func (f *fakeFavoriteService) QueryFavoriteInfo(
	context.Context,
	*interactionpb.QueryFavoriteInfoReq,
	...grpc.CallOption,
) (*interactionpb.QueryFavoriteInfoRes, error) {
	return &interactionpb.QueryFavoriteInfoRes{}, nil
}

func (f *fakeFavoriteService) QueryFavoriteList(
	context.Context,
	*interactionpb.QueryFavoriteListReq,
	...grpc.CallOption,
) (*interactionpb.QueryFavoriteListRes, error) {
	return &interactionpb.QueryFavoriteListRes{}, nil
}

type fakeCommentService struct {
	commentReq *interactionpb.CommentReq
}

func (f *fakeCommentService) Comment(
	_ context.Context,
	in *interactionpb.CommentReq,
	_ ...grpc.CallOption,
) (*interactionpb.CommentRes, error) {
	f.commentReq = in
	return &interactionpb.CommentRes{CommentId: 3001}, nil
}

func (f *fakeCommentService) DeleteComment(
	context.Context,
	*interactionpb.DeleteCommentReq,
	...grpc.CallOption,
) (*interactionpb.DeleteCommentRes, error) {
	return &interactionpb.DeleteCommentRes{}, nil
}

func (f *fakeCommentService) QueryCommentList(
	context.Context,
	*interactionpb.QueryCommentListReq,
	...grpc.CallOption,
) (*interactionpb.QueryCommentListRes, error) {
	return &interactionpb.QueryCommentListRes{}, nil
}

func (f *fakeCommentService) QueryReplyList(
	context.Context,
	*interactionpb.QueryReplyListReq,
	...grpc.CallOption,
) (*interactionpb.QueryReplyListRes, error) {
	return &interactionpb.QueryReplyListRes{}, nil
}

func (f *fakeCommentService) BatchGetComments(
	context.Context,
	*interactionpb.BatchGetCommentsReq,
	...grpc.CallOption,
) (*interactionpb.BatchGetCommentsRes, error) {
	return &interactionpb.BatchGetCommentsRes{}, nil
}

func (f *fakeCommentService) RefillCommentCache(
	context.Context,
	*interactionpb.RefillCommentCacheReq,
	...grpc.CallOption,
) (*interactionpb.RefillCommentCacheRes, error) {
	return &interactionpb.RefillCommentCacheRes{}, nil
}

type fakeFeedService struct {
	trackReq *contentpb.EmitRecommendTrackReq
	err      error
}

func (f *fakeFeedService) RecommendFeed(
	context.Context,
	*contentpb.RecommendFeedReq,
	...grpc.CallOption,
) (*contentpb.RecommendFeedRes, error) {
	return &contentpb.RecommendFeedRes{}, nil
}

func (f *fakeFeedService) EmitRecommendTrack(
	_ context.Context,
	in *contentpb.EmitRecommendTrackReq,
	_ ...grpc.CallOption,
) (*contentpb.EmitRecommendTrackRes, error) {
	f.trackReq = in
	if f.err != nil {
		return nil, f.err
	}
	return &contentpb.EmitRecommendTrackRes{}, nil
}

func (f *fakeFeedService) FollowFeed(
	context.Context,
	*contentpb.FollowFeedReq,
	...grpc.CallOption,
) (*contentpb.FollowFeedRes, error) {
	return &contentpb.FollowFeedRes{}, nil
}

func (f *fakeFeedService) UserPublishFeed(
	context.Context,
	*contentpb.UserPublishFeedReq,
	...grpc.CallOption,
) (*contentpb.UserPublishFeedRes, error) {
	return &contentpb.UserPublishFeedRes{}, nil
}

func (f *fakeFeedService) UserFavoriteFeed(
	context.Context,
	*contentpb.UserFavoriteFeedReq,
	...grpc.CallOption,
) (*contentpb.UserFavoriteFeedRes, error) {
	return &contentpb.UserFavoriteFeedRes{}, nil
}

func TestLikeEmitsRecommendTrackAfterSuccess(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &fakeLikeService{}
	feedRPC := &fakeFeedService{}
	logic := NewLikeLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{
			LikeRpc: likeRPC,
			FeedRpc: feedRPC,
		},
	)

	resp, err := logic.Like(&types.LikeReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("Like returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Like returned nil response")
	}
	if likeRPC.likeReq == nil {
		t.Fatal("LikeRpc.Like was not called")
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}

	got := feedRPC.trackReq
	if got.GetUserId() != 1001 ||
		got.GetEventType() != "like" ||
		got.GetContentId() != contentID ||
		got.GetSource() != "interaction" {
		t.Fatalf("recommend track request = %+v", got)
	}
}

func TestLikeIgnoresRecommendTrackFailure(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	likeRPC := &fakeLikeService{}
	feedRPC := &fakeFeedService{err: errors.New("content rpc down")}
	logic := NewLikeLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{
			LikeRpc: likeRPC,
			FeedRpc: feedRPC,
		},
	)

	resp, err := logic.Like(&types.LikeReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("Like returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Like returned nil response")
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}
}

func TestFavoriteEmitsRecommendTrackAfterSuccess(t *testing.T) {
	contentID := int64(2001)
	scene := "ARTICLE"
	favoriteRPC := &fakeFavoriteService{}
	feedRPC := &fakeFeedService{}
	logic := NewFavoriteLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{
			FavoriteRpc: favoriteRPC,
			FeedRpc:     feedRPC,
		},
	)

	resp, err := logic.Favorite(&types.FavoriteReq{
		ContentId: &contentID,
		Scene:     &scene,
	})
	if err != nil {
		t.Fatalf("Favorite returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Favorite returned nil response")
	}
	if favoriteRPC.favoriteReq == nil {
		t.Fatal("FavoriteRpc.Favorite was not called")
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}

	got := feedRPC.trackReq
	if got.GetUserId() != 1001 ||
		got.GetEventType() != "favorite" ||
		got.GetContentId() != contentID ||
		got.GetSource() != "interaction" {
		t.Fatalf("recommend track request = %+v", got)
	}
}

func TestCommentEmitsRecommendTrackAfterSuccess(t *testing.T) {
	contentID := int64(2001)
	contentUserID := int64(3001)
	scene := "ARTICLE"
	comment := "nice post"
	commentRPC := &fakeCommentService{}
	feedRPC := &fakeFeedService{}
	logic := NewCommentLogic(
		context.WithValue(context.Background(), "user_id", int64(1001)),
		&svc.ServiceContext{
			CommentRpc: commentRPC,
			FeedRpc:    feedRPC,
		},
	)

	resp, err := logic.Comment(&types.CommentReq{
		ContentId:     &contentID,
		ContentUserId: &contentUserID,
		Scene:         &scene,
		Comment:       &comment,
	})
	if err != nil {
		t.Fatalf("Comment returned error: %v", err)
	}
	if resp == nil || resp.CommentId != 3001 {
		t.Fatalf("Comment response = %+v", resp)
	}
	if commentRPC.commentReq == nil {
		t.Fatal("CommentRpc.Comment was not called")
	}
	if feedRPC.trackReq == nil {
		t.Fatal("FeedRpc.EmitRecommendTrack was not called")
	}

	got := feedRPC.trackReq
	if got.GetUserId() != 1001 ||
		got.GetEventType() != "comment" ||
		got.GetContentId() != contentID ||
		got.GetSource() != "interaction" {
		t.Fatalf("recommend track request = %+v", got)
	}
}
