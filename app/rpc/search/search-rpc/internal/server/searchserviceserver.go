package server

import (
	"context"

	"zfeed/app/rpc/search/search"
	"zfeed/app/rpc/search/search-rpc/internal/logic"
	"zfeed/app/rpc/search/search-rpc/internal/svc"
)

type SearchServiceServer struct {
	svcCtx *svc.ServiceContext
	search.UnimplementedSearchServiceServer
}

func NewSearchServiceServer(svcCtx *svc.ServiceContext) *SearchServiceServer {
	return &SearchServiceServer{
		svcCtx: svcCtx,
	}
}

func (s *SearchServiceServer) SearchUsers(ctx context.Context, in *search.SearchUsersReq) (*search.SearchUsersRes, error) {
	l := logic.NewSearchUsersLogic(ctx, s.svcCtx)
	return l.SearchUsers(in)
}

func (s *SearchServiceServer) SearchContents(ctx context.Context, in *search.SearchContentsReq) (*search.SearchContentsRes, error) {
	l := logic.NewSearchContentsLogic(ctx, s.svcCtx)
	return l.SearchContents(in)
}
