CREATE TABLE IF NOT EXISTS `zfeed_favorite_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(128) NOT NULL,
  `event_type` VARCHAR(32) NOT NULL,
  `scene` TINYINT NOT NULL DEFAULT 0,
  `user_id` BIGINT NOT NULL,
  `content_id` BIGINT NOT NULL,
  `content_user_id` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_event_id` (`event_id`),
  KEY `idx_user_created` (`user_id`, `created_at`),
  KEY `idx_content_created` (`content_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
