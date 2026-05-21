SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_content` ADD COLUMN `hot_score` DOUBLE NOT NULL DEFAULT 0 AFTER `comment_count`'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_content'
    AND column_name = 'hot_score'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_content` ADD COLUMN `last_hot_score_at` DATETIME NULL AFTER `hot_score`'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_content'
    AND column_name = 'last_hot_score_at'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ddl = (
  SELECT IF(
    COUNT(1) > 0,
    'SELECT 1',
    'ALTER TABLE `zfeed_content` ADD KEY `idx_hot_score` (`hot_score`, `id`)'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'zfeed_content'
    AND index_name = 'idx_hot_score'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

