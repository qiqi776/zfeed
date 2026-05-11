CREATE TABLE IF NOT EXISTS `zfeed_user` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL DEFAULT '',
  `nickname` VARCHAR(64) NOT NULL DEFAULT '',
  `avatar` VARCHAR(512) NOT NULL DEFAULT '',
  `bio` VARCHAR(255) NOT NULL DEFAULT '',
  `mobile` VARCHAR(32) NOT NULL DEFAULT '',
  `email` VARCHAR(128) NOT NULL DEFAULT '',
  `password_hash` VARCHAR(255) NOT NULL DEFAULT '',
  `password_salt` VARCHAR(64) NOT NULL DEFAULT '',
  `gender` TINYINT NOT NULL DEFAULT 0,
  `birthday` DATE DEFAULT NULL,
  `status` INT NOT NULL DEFAULT 10,
  `is_deleted` TINYINT NOT NULL DEFAULT 0,
  `active_mobile` VARCHAR(32) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `mobile`, NULL)) STORED,
  `active_email` VARCHAR(128) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `email`, NULL)) STORED,
  `created_by` BIGINT NOT NULL DEFAULT 0,
  `updated_by` BIGINT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_active_mobile` (`active_mobile`),
  UNIQUE KEY `uk_active_email` (`active_email`),
  KEY `idx_mobile_active` (`mobile`, `is_deleted`),
  KEY `idx_email_active` (`email`, `is_deleted`),
  KEY `idx_status_deleted` (`status`, `is_deleted`),
  FULLTEXT KEY `ft_user_search` (`nickname`, `bio`, `mobile`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_user'
        AND column_name = 'active_mobile'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD COLUMN `active_mobile` VARCHAR(32) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `mobile`, NULL)) STORED AFTER `is_deleted`'
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
        AND table_name = 'zfeed_user'
        AND index_name = 'idx_mobile_active'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD KEY `idx_mobile_active` (`mobile`, `is_deleted`)'
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
        AND table_name = 'zfeed_user'
        AND index_name = 'idx_email_active'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD KEY `idx_email_active` (`email`, `is_deleted`)'
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
        AND table_name = 'zfeed_user'
        AND column_name = 'active_email'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD COLUMN `active_email` VARCHAR(128) GENERATED ALWAYS AS (IF(`is_deleted` = 0, `email`, NULL)) STORED AFTER `active_mobile`'
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
        AND table_name = 'zfeed_user'
        AND index_name = 'uk_active_mobile'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD UNIQUE KEY `uk_active_mobile` (`active_mobile`)'
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
        AND table_name = 'zfeed_user'
        AND index_name = 'uk_active_email'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD UNIQUE KEY `uk_active_email` (`active_email`)'
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
        AND table_name = 'zfeed_user'
        AND index_name = 'uk_mobile'
    ),
    'ALTER TABLE `zfeed_user` DROP INDEX `uk_mobile`',
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
        AND table_name = 'zfeed_user'
        AND index_name = 'uk_email'
    ),
    'ALTER TABLE `zfeed_user` DROP INDEX `uk_email`',
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
        AND table_name = 'zfeed_user'
        AND index_name = 'ft_user_search'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD FULLTEXT KEY `ft_user_search` (`nickname`, `bio`, `mobile`)'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
