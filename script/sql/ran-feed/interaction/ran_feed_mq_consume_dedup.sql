CREATE TABLE IF NOT EXISTS `ran_feed_mq_consume_dedup` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `consumer` VARCHAR(64) NOT NULL,
  `event_id` VARCHAR(64) NOT NULL,
  `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_consumer_event` (`consumer`, `event_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;