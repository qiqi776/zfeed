package event

import "encoding/json"

type EventType string

const (
	EventTypeLike   EventType = "like"
	EventTypeCancel EventType = "cancel_like"
)

type LikeEvent struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	UserID        int64     `json:"user_id"`
	ContentID     int64     `json:"content_id"`
	ContentUserID int64     `json:"content_user_id"`
	Scene         string    `json:"scene"`
	Timestamp     int64     `json:"timestamp"`
}

func (e *LikeEvent) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func UnmarshalLikeEvent(raw string) (*LikeEvent, error) {
	var evt LikeEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		return nil, err
	}
	return &evt, nil
}
