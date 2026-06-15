CREATE TABLE IF NOT EXISTS `zfeed_rec_metric_daily` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `metric_date` DATE NOT NULL,
  `variant_id` VARCHAR(64) NOT NULL DEFAULT 'control',
  `source` VARCHAR(32) NOT NULL DEFAULT 'unknown',
  `exposure_count` BIGINT NOT NULL DEFAULT 0,
  `click_count` BIGINT NOT NULL DEFAULT 0,
  `dwell_count` BIGINT NOT NULL DEFAULT 0,
  `dwell_ms_sum` BIGINT NOT NULL DEFAULT 0,
  `like_count` BIGINT NOT NULL DEFAULT 0,
  `favorite_count` BIGINT NOT NULL DEFAULT 0,
  `comment_count` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updated_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rec_metric_day_variant_source` (`metric_date`, `variant_id`, `source`),
  KEY `idx_variant_date` (`variant_id`, `metric_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
