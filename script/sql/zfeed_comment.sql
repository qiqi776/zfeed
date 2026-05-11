CREATE TABLE IF NOT EXISTS `zfeed_comment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `content_id` BIGINT NOT NULL DEFAULT 0,
  `content_user_id` BIGINT NOT NULL DEFAULT 0,
  `user_id` BIGINT NOT NULL DEFAULT 0,
  `reply_to_user_id` BIGINT NOT NULL DEFAULT 0,
  `parent_id` BIGINT NOT NULL DEFAULT 0,
  `root_id` BIGINT NOT NULL DEFAULT 0,
  `comment` VARCHAR(255) NOT NULL DEFAULT '',
  `status` TINYINT NOT NULL DEFAULT 10 COMMENT '10=normal,20=deleted',
  `version` INT NOT NULL DEFAULT 1,
  `reply_count` BIGINT NOT NULL DEFAULT 0,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_content_root_list` (`content_id`, `root_id`, `is_deleted`, `id`),
  KEY `idx_content_root_status_list` (`content_id`, `root_id`, `is_deleted`, `status`, `id`),
  KEY `idx_root_reply_list` (`root_id`, `is_deleted`, `id`),
  KEY `idx_root_reply_status_list` (`root_id`, `is_deleted`, `status`, `id`),
  KEY `idx_parent_list` (`parent_id`, `is_deleted`, `id`),
  KEY `idx_content_user` (`content_user_id`),
  KEY `idx_user_comment_list` (`user_id`, `is_deleted`, `id`),
  KEY `idx_user_comment_status_list` (`user_id`, `is_deleted`, `status`, `id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_content_root_status_list` (`content_id`, `root_id`, `is_deleted`, `status`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_content_root_status_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_root_reply_status_list` (`root_id`, `is_deleted`, `status`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_root_reply_status_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_user_comment_status_list` (`user_id`, `is_deleted`, `status`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_user_comment_status_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
