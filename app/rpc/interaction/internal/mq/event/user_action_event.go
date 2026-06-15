package event

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	UserActionLike       = "like"
	UserActionUnlike     = "unlike"
	UserActionFavorite   = "favorite"
	UserActionUnfavorite = "unfavorite"
	UserActionComment    = "comment"

	UserActionSourceInteraction = "interaction"
	UserActionTargetContent     = "content"
)

type UserActionEvent struct {
	EventID       string `json:"event_id"`
	Action        string `json:"action"`
	UserID        int64  `json:"user_id"`
	TargetType    string `json:"target_type"`
	TargetID      int64  `json:"target_id"`
	ContentID     int64  `json:"content_id,omitempty"`
	ContentUserID int64  `json:"content_user_id,omitempty"`
	Scene         string `json:"scene,omitempty"`
	Source        string `json:"source"`
	OccurredAt    int64  `json:"occurred_at"`
}

func NewUserActionEvent(action string, userID, contentID, contentUserID int64, scene string) UserActionEvent {
	now := time.Now()
	return UserActionEvent{
		EventID:       fmt.Sprintf("ua_%s_%d_%d_%d", action, userID, contentID, now.UnixNano()),
		Action:        action,
		UserID:        userID,
		TargetType:    UserActionTargetContent,
		TargetID:      contentID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         scene,
		Source:        UserActionSourceInteraction,
		OccurredAt:    now.Unix(),
	}
}

func (e UserActionEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
