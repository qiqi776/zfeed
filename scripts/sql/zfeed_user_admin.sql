-- Admin system migration
-- Adds role column to zfeed_user and creates admin audit log table

-- 1. Add role column (0=普通用户, 1=版主, 2=管理员)
SET @ddl := (
  SELECT IF(
    EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = DATABASE()
        AND table_name = 'zfeed_user'
        AND column_name = 'role'
    ),
    'SELECT 1',
    'ALTER TABLE `zfeed_user` ADD COLUMN `role` TINYINT NOT NULL DEFAULT 0 COMMENT ''0=普通用户 1=版主 2=管理员'' AFTER `is_deleted`'
  )
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 2. Create admin audit log table
CREATE TABLE IF NOT EXISTS `zfeed_admin_audit_log` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT NOT NULL,
  `action` VARCHAR(64) NOT NULL COMMENT 'disable_user|hide_content|delete_comment|...',
  `target_type` VARCHAR(32) NOT NULL COMMENT 'user|content|comment',
  `target_id` BIGINT NOT NULL DEFAULT 0,
  `detail` JSON DEFAULT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_admin` (`admin_id`, `created_at`),
  KEY `idx_target` (`target_type`, `target_id`),
  KEY `idx_action` (`action`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Set initial admin user (replace 1 with your admin user id)
-- UPDATE `zfeed_user` SET `role` = 2 WHERE `id` = 1;
