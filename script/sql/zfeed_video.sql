CREATE TABLE IF NOT EXISTS `zfeed_video` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `content_id` BIGINT NOT NULL,
  `title` VARCHAR(100) NOT NULL DEFAULT '',
  `description` VARCHAR(500) DEFAULT NULL,
  `origin_url` VARCHAR(1024) NOT NULL DEFAULT '',
  `cover_url` VARCHAR(1024) NOT NULL DEFAULT '',
  `duration` INT NOT NULL DEFAULT 0,
  `transcode_status` INT NOT NULL DEFAULT 10,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_content_id` (`content_id`),
  FULLTEXT KEY `ft_video_search` (`title`, `description`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.statistics
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_video'
        AND index_name = 'ft_video_search'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_video` ADD FULLTEXT KEY `ft_video_search` (`title`, `description`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
