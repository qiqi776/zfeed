CREATE TABLE IF NOT EXISTS `zfeed_favorite` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL,
  `scene` TINYINT NOT NULL DEFAULT 0,
  `status` TINYINT NOT NULL COMMENT '10=active,20=cancel',
  `content_id` BIGINT NOT NULL,
  `content_user_id` BIGINT NOT NULL DEFAULT 0,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_scene_content` (`user_id`, `scene`, `content_id`),
  KEY `idx_user_created` (`user_id`, `created_at` DESC),
  KEY `idx_scene_content_status` (`scene`, `content_id`, `status`),
  KEY `idx_scene_content_user` (`scene`, `content_user_id`),
  KEY `idx_user_status_scene_id` (`user_id`, `status`, `scene`, `id`),
  KEY `idx_content_id` (`content_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND column_name = 'scene'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD COLUMN `scene` TINYINT NOT NULL DEFAULT 0 AFTER `user_id`'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND column_name = 'content_user_id'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD COLUMN `content_user_id` BIGINT NOT NULL DEFAULT 0 AFTER `content_id`'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'idx_scene_content_status'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD KEY `idx_scene_content_status` (`scene`, `content_id`, `status`)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'idx_scene_content_user'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD KEY `idx_scene_content_user` (`scene`, `content_user_id`)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'idx_user_status_scene_id'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD KEY `idx_user_status_scene_id` (`user_id`, `status`, `scene`, `id`)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'idx_content_id'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD KEY `idx_content_id` (`content_id`)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'uk_user_content'
    ),
    'ALTER TABLE `zfeed_favorite` DROP INDEX `uk_user_content`',
    'SELECT 1'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl := IF (
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_favorite'
        AND index_name = 'uk_user_scene_content'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_favorite` ADD UNIQUE KEY `uk_user_scene_content` (`user_id`, `scene`, `content_id`)'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
