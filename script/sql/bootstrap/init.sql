CREATE DATABASE IF NOT EXISTS `zfeed` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `xxl_job` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS 'zfeed'@'%' IDENTIFIED BY '123456';
GRANT ALL PRIVILEGES ON `zfeed`.* TO 'zfeed'@'%';
GRANT ALL PRIVILEGES ON `xxl_job`.* TO 'zfeed'@'%';
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'zfeed'@'%';
FLUSH PRIVILEGES;

USE `zfeed`;

SOURCE /seed-sql/zfeed_user.sql;
SOURCE /seed-sql/zfeed_content.sql;
SOURCE /seed-sql/zfeed_article.sql;
SOURCE /seed-sql/zfeed_video.sql;
SOURCE /seed-sql/zfeed_like.sql;
SOURCE /seed-sql/zfeed_favorite.sql;
SOURCE /seed-sql/zfeed_follow.sql;
SOURCE /seed-sql/zfeed_mq_consume_dedup.sql;
SOURCE /seed-sql/zfeed_comment.sql;
SOURCE /seed-sql/zfeed_comment_migrate.sql;
SOURCE /seed-sql/zfeed_count_value.sql;
