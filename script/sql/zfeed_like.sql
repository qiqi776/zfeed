CREATE TABLE IF NOT EXISTS `zfeed_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `scene` TINYINT NOT NULL DEFAULT 0,
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
  UNIQUE KEY `uk_user_scene_content` (`user_id`, `scene`, `content_id`),
  KEY `idx_scene_content` (`scene`, `content_id`),
  KEY `idx_scene_content_user` (`scene`, `content_user_id`),
  KEY `idx_user_scene_status` (`user_id`, `scene`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_like'
        AND column_name = 'scene'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_like` ADD COLUMN `scene` TINYINT NOT NULL DEFAULT 0 AFTER `user_id`'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

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

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_like'
        AND index_name = 'uk_user_content'
    ),
    'ALTER TABLE `zfeed_like` DROP INDEX `uk_user_content`',
    'SELECT 1'
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
        AND table_name = 'zfeed_like'
        AND index_name = 'uk_user_scene_content'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_like` ADD UNIQUE KEY `uk_user_scene_content` (`user_id`, `scene`, `content_id`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
