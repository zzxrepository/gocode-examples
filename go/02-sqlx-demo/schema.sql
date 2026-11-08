-- 独立教程示例，仅创建 person 表。重复初始化不重置已有记录。
CREATE TABLE IF NOT EXISTS person (
    user_id BIGINT NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    age INT NOT NULL,
    email VARCHAR(128) NULL,
    PRIMARY KEY (user_id),
    UNIQUE KEY uk_person_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO person (username, age, email)
VALUES ('alice', 20, 'alice@example.com'), ('bob', 21, NULL);
