CREATE TABLE IF NOT EXISTS `zfeed_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `content_id` BIGINT NOT NULL,
  `content_user_id` BIGINT NOT NULL DEFAULT 0,
  `status` TINYINT NOT NULL COMMENT '10=like,20=cancel',
  `last_event_ts` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_content` (`user_id`, `content_id`),
  KEY `idx_content` (`content_id`),
  KEY `idx_content_user` (`content_user_id`),
  KEY `idx_user_status` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_like'
        AND column_name = 'last_event_ts'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_like` ADD COLUMN `last_event_ts` BIGINT NOT NULL DEFAULT 0 AFTER `status`'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
