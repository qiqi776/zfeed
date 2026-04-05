SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD COLUMN `content_user_id` BIGINT NOT NULL DEFAULT 0 AFTER `content_id`'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND column_name = 'content_user_id'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD COLUMN `version` INT NOT NULL DEFAULT 1 AFTER `status`'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND column_name = 'version'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_content_root_list` (`content_id`, `root_id`, `is_deleted`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_content_root_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_root_reply_list` (`root_id`, `is_deleted`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_root_reply_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_parent_list` (`parent_id`, `is_deleted`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_parent_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_content_user` (`content_user_id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_content_user'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_comment` ADD KEY `idx_user_comment_list` (`user_id`, `is_deleted`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_comment'
    AND index_name = 'idx_user_comment_list'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
