package contentservicelogic

import (
	"context"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BackfillFollowInboxLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BackfillFollowInboxLogic) BackfillFollowInbox(in *content.BackfillFollowInboxReq) (*content.BackfillFollowInboxRes, error) {
	// todo: add your logic here and delete this line

	return &content.BackfillFollowInboxRes{}, nil
}
