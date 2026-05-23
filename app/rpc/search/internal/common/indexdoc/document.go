package indexdoc

type ContentDocument struct {
	ContentID    int64   `json:"content_id"`
	ContentType  int32   `json:"content_type"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	AuthorID     int64   `json:"author_id"`
	AuthorName   string  `json:"author_name"`
	AuthorAvatar string  `json:"author_avatar"`
	PublishedAt  int64   `json:"published_at"`
	Visibility   int32   `json:"visibility"`
	Status       int32   `json:"status"`
	HotScore     float64 `json:"hot_score"`
}

type UserDocument struct {
	UserID            int64  `json:"user_id"`
	Nickname          string `json:"nickname"`
	Bio               string `json:"bio"`
	MobileSearchField string `json:"mobile_search_field"`
	Status            int32  `json:"status"`
}
