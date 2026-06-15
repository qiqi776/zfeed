package favoritelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/mq/event"
)

type userActionEmitter interface {
	SendUserAction(ctx context.Context, action event.UserActionEvent) error
}

func emitUserAction(
	ctx context.Context,
	logger logx.Logger,
	emitter userActionEmitter,
	action string,
	userID int64,
	contentID int64,
	contentUserID int64,
	scene interaction.Scene,
) {
	if emitter == nil || userID <= 0 || contentID <= 0 {
		return
	}

	err := emitter.SendUserAction(ctx, event.NewUserActionEvent(
		action,
		userID,
		contentID,
		contentUserID,
		scene.String(),
	))
	if err != nil {
		logger.Errorf(
			"emit user action failed, action=%s, user_id=%d, content_id=%d, err=%v",
			action,
			userID,
			contentID,
			err,
		)
	}
}
