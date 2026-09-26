-- 仅创建课程自己的数据库与表；不覆盖已有业务表。
CREATE DATABASE IF NOT EXISTS kafka_order_demo;
USE kafka_order_demo;
CREATE TABLE IF NOT EXISTS processed_events (
  consumer_name VARCHAR(160) NOT NULL,
  event_id VARCHAR(160) NOT NULL,
  PRIMARY KEY (consumer_name,event_id)
);
CREATE TABLE IF NOT EXISTS order_projection (
  consumer_name VARCHAR(160) NOT NULL,
  order_id VARCHAR(160) NOT NULL,
  status VARCHAR(32) NOT NULL,
  version INT NOT NULL,
  amount_cents BIGINT NOT NULL,
  PRIMARY KEY (consumer_name,order_id)
);

CREATE TABLE IF NOT EXISTS source_orders (
 order_id VARCHAR(160) PRIMARY KEY,
 status VARCHAR(32) NOT NULL
);
CREATE TABLE IF NOT EXISTS order_outbox (
 id BIGINT AUTO_INCREMENT PRIMARY KEY,
 event_id VARCHAR(160) NOT NULL UNIQUE,
 topic VARCHAR(249) NOT NULL,
 payload JSON NOT NULL,
 published BOOLEAN NOT NULL DEFAULT FALSE
);
