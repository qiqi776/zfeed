CREATE TABLE IF NOT EXISTS `zfeed_count_value` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `biz_type` INT NOT NULL COMMENT '10=like,20=favorite,30=comment,40=followed,41=following',
  `target_type` INT NOT NULL COMMENT '10=content,20=user',
  `target_id` BIGINT NOT NULL COMMENT 'content_id or user_id',
  `value` BIGINT NOT NULL DEFAULT 0,
  `version` BIGINT NOT NULL DEFAULT 0,
  `owner_id` BIGINT NOT NULL DEFAULT 0 COMMENT 'content owner id when target_type=content',
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_biz_target` (`biz_type`, `target_type`, `target_id`),
  KEY `idx_owner` (`owner_id`),
  KEY `idx_target` (`target_type`, `target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
