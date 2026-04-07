package model

import "time"

type ZfeedCountValue struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	BizType    int32     `gorm:"column:biz_type;uniqueIndex:uk_biz_target;index:idx_target"`
	TargetType int32     `gorm:"column:target_type;uniqueIndex:uk_biz_target;index:idx_target"`
	TargetID   int64     `gorm:"column:target_id;uniqueIndex:uk_biz_target;index:idx_target"`
	Value      int64     `gorm:"column:value"`
	Version    int64     `gorm:"column:version"`
	OwnerID    int64     `gorm:"column:owner_id;index:idx_owner"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (ZfeedCountValue) TableName() string {
	return "zfeed_count_value"
}

type ZfeedMqConsumeDedup struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Consumer  string    `gorm:"column:consumer;uniqueIndex:uniq_consumer_event"`
	EventID   string    `gorm:"column:event_id;uniqueIndex:uniq_consumer_event"`
	CreatedAt time.Time `gorm:"column:created_at;index:idx_created_at"`
}

func (ZfeedMqConsumeDedup) TableName() string {
	return "zfeed_mq_consume_dedup"
}
