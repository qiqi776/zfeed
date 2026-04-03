package do

type LikeDO struct {
	UserID        int64
	ContentID     int64
	ContentUserID int64
	Status        int32
	LastEventTs   int64
	CreatedBy     int64
	UpdatedBy     int64
}
