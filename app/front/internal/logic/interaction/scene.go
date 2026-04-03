package interaction

import (
	"strings"

	interactionpb "zfeed/app/rpc/interaction/interaction"
	"zfeed/pkg/errorx"
)

func parseScene(raw string) (interactionpb.Scene, error) {
	key := strings.ToUpper(strings.TrimSpace(raw))
	val, ok := interactionpb.Scene_value[key]
	if !ok {
		return interactionpb.Scene_SCENE_UNKNOWN, errorx.NewMsg("场景参数错误")
	}
	scene := interactionpb.Scene(val)
	if scene == interactionpb.Scene_SCENE_UNKNOWN {
		return interactionpb.Scene_SCENE_UNKNOWN, errorx.NewMsg("场景参数错误")
	}
	return scene, nil
}
