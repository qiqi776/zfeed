package changeevent

import "time"

type ChangeEvent struct {
	EventID   string
	Source    string
	Table     string
	Operation string
	Timestamp time.Time
	Current   map[string]any
	Previous  map[string]any
}
