CREATE TABLE IF NOT EXISTS `zfeed_follow` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT 'follower user id',
  `follow_user_id` BIGINT NOT NULL COMMENT 'followee user id',
  `status` TINYINT NOT NULL COMMENT '10=follow,20=unfollow',
  `version` INT NOT NULL DEFAULT 1,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_follow_user` (`user_id`, `follow_user_id`),
  KEY `idx_user_status_follow` (`user_id`, `status`, `is_deleted`, `follow_user_id`),
  KEY `idx_follow_status_user` (`follow_user_id`, `status`, `is_deleted`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_follow'
        AND column_name = 'version'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_follow` ADD COLUMN `version` INT NOT NULL DEFAULT 1 AFTER `status`'
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
        AND table_name = 'zfeed_follow'
        AND index_name = 'idx_user_status_follow'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_follow` ADD KEY `idx_user_status_follow` (`user_id`, `status`, `is_deleted`, `follow_user_id`)'
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
        AND table_name = 'zfeed_follow'
        AND index_name = 'idx_follow_status_user'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_follow` ADD KEY `idx_follow_status_user` (`follow_user_id`, `status`, `is_deleted`, `user_id`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
