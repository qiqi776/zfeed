CREATE TABLE IF NOT EXISTS `zfeed_content` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL DEFAULT 0,
  `content_type` INT NOT NULL DEFAULT 0,
  `status` INT NOT NULL DEFAULT 0,
  `visibility` INT NOT NULL DEFAULT 0,
  `like_count` BIGINT NOT NULL DEFAULT 0 COMMENT 'legacy denormalized field, count service remains the primary truth',
  `favorite_count` BIGINT NOT NULL DEFAULT 0 COMMENT 'legacy denormalized field, count service remains the primary truth',
  `comment_count` BIGINT NOT NULL DEFAULT 0 COMMENT 'legacy denormalized field, count service remains the primary truth',
  `hot_score` DOUBLE NOT NULL DEFAULT 0 COMMENT 'calculated hot score for ranking maintenance jobs',
  `last_hot_score_at` DATETIME DEFAULT NULL COMMENT 'last hot score refresh time',
  `published_at` DATETIME DEFAULT NULL,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_publish_list` (`user_id`, `status`, `visibility`, `is_deleted`, `id`),
  KEY `idx_user_publish_time` (`user_id`, `status`, `visibility`, `is_deleted`, `published_at`, `id`),
  KEY `idx_public_feed` (`status`, `visibility`, `is_deleted`, `id`),
  KEY `idx_public_published` (`status`, `visibility`, `is_deleted`, `published_at`, `id`),
  KEY `idx_hot_score` (`hot_score`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_content'
        AND index_name = 'idx_public_feed'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_content` ADD KEY `idx_public_feed` (`status`, `visibility`, `is_deleted`, `id`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_content'
        AND index_name = 'idx_public_published'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_content` ADD KEY `idx_public_published` (`status`, `visibility`, `is_deleted`, `published_at`, `id`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
