CREATE TABLE IF NOT EXISTS `zfeed_like_event_outbox` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `event_id` VARCHAR(128) NOT NULL,
  `event_type` VARCHAR(32) NOT NULL,
  `payload` MEDIUMTEXT NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 10,
  `retry_count` INT NOT NULL DEFAULT 0,
  `next_retry_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `last_error` VARCHAR(512) NOT NULL DEFAULT '',
  `sent_at` DATETIME(3) DEFAULT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_event_id` (`event_id`),
  KEY `idx_status_retry` (`status`, `next_retry_at`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
