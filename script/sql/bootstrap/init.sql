CREATE DATABASE IF NOT EXISTS `zfeed` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `xxl_job` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS 'zfeed'@'%' IDENTIFIED BY '123456';
GRANT ALL PRIVILEGES ON `zfeed`.* TO 'zfeed'@'%';
GRANT ALL PRIVILEGES ON `xxl_job`.* TO 'zfeed'@'%';
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO 'zfeed'@'%';
FLUSH PRIVILEGES;

USE `zfeed`;

SOURCE script/sql/zfeed_user.sql;
SOURCE script/sql/zfeed_content.sql;
SOURCE script/sql/zfeed_article.sql;
SOURCE script/sql/zfeed_video.sql;
SOURCE script/sql/zfeed_like.sql;
SOURCE script/sql/zfeed_favorite.sql;
SOURCE script/sql/zfeed_follow.sql;
SOURCE script/sql/zfeed_mq_consume_dedup.sql;
SOURCE script/sql/zfeed_comment.sql;
SOURCE script/sql/zfeed_comment_migrate.sql;
