DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `name` varchar(50) NOT NULL DEFAULT '',
 `password` varchar(100) NOT NULL DEFAULT '',
 `nickname` varchar(50) NOT NULL DEFAULT '',
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 `deleted_at` timestamp NULL DEFAULT NULL,
 `avator` varchar(100) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
TRUNCATE TABLE `user`;
INSERT INTO `user` (`id`, `name`, `password`, `nickname`, `created_at`, `updated_at`, `deleted_at`, `avator`) VALUE
(1, 'agent', '$2a$10$r/URu06JLWL898kj7ojHp.kH58kVN8xOjveZnGnBZsNl.bMghdw/O', 'Open Source LiveChat Support', '2020-06-27 19:32:41', '2020-07-04 09:32:20', NULL, '/static/images/4.jpg');

DROP TABLE IF EXISTS `role`;
CREATE TABLE `role` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `name` varchar(50) NOT NULL DEFAULT '',
 `method` varchar(255) NOT NULL DEFAULT '*',
 `path` varchar(1024) NOT NULL DEFAULT '*',
 PRIMARY KEY (`id`),
 UNIQUE KEY `idx_role_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
INSERT INTO `role` (`id`, `name`, `method`, `path`) VALUES
(1, 'super_admin', '*', '*'),
(2, 'manager', '*', '*'),
(3, 'agent', '*', '*'),
(4, 'auditor', '*', '*');

DROP TABLE IF EXISTS `user_role`;
CREATE TABLE `user_role` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `user_id` varchar(50) NOT NULL DEFAULT '',
 `role_id` int(11) NOT NULL DEFAULT '0',
 PRIMARY KEY (`id`),
 UNIQUE KEY `idx_user_role_user_id` (`user_id`),
 KEY `idx_user_role_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
INSERT INTO `user_role` (`id`, `user_id`, `role_id`) VALUES
(1, '1', 1);

DROP TABLE IF EXISTS `audit_log`;
CREATE TABLE `audit_log` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `actor_user_id` int(11) NOT NULL DEFAULT '0',
 `actor_name` varchar(100) NOT NULL DEFAULT '',
 `actor_role` varchar(50) NOT NULL DEFAULT '',
 `action` varchar(100) NOT NULL DEFAULT '',
 `target_type` varchar(100) NOT NULL DEFAULT '',
 `target_id` varchar(100) NOT NULL DEFAULT '',
 `before_data` text,
 `after_data` text,
 `client_ip` varchar(100) NOT NULL DEFAULT '',
 `request_id` varchar(100) NOT NULL DEFAULT '',
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY (`id`),
 KEY `idx_audit_action` (`action`),
 KEY `idx_audit_target` (`target_type`, `target_id`),
 KEY `idx_audit_actor` (`actor_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `conversation_session`;
CREATE TABLE `conversation_session` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `session_id` varchar(100) NOT NULL DEFAULT '',
 `visitor_id` varchar(100) NOT NULL DEFAULT '',
 `visitor_name` varchar(100) NOT NULL DEFAULT '',
 `owner_id` varchar(100) NOT NULL DEFAULT '',
 `sticky_owner_id` varchar(100) NOT NULL DEFAULT '',
 `route_status` varchar(50) NOT NULL DEFAULT '',
 `queue_name` varchar(100) NOT NULL DEFAULT '',
 `preferred_skill` varchar(100) NOT NULL DEFAULT '',
 `source_entry` varchar(100) NOT NULL DEFAULT '',
 `served_by_type` varchar(50) NOT NULL DEFAULT '',
 `last_route_reason` varchar(255) NOT NULL DEFAULT '',
 `queue_entered_at` timestamp NULL DEFAULT NULL,
 `last_assign_attempt_at` timestamp NULL DEFAULT NULL,
 `last_assigned_at` timestamp NULL DEFAULT NULL,
 `last_transfer_at` timestamp NULL DEFAULT NULL,
 `last_activity_at` timestamp NULL DEFAULT NULL,
 `closed_at` timestamp NULL DEFAULT NULL,
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `idx_conversation_session_id` (`session_id`),
 KEY `idx_conversation_visitor_id` (`visitor_id`),
 KEY `idx_conversation_route_status` (`route_status`),
 KEY `idx_conversation_owner_id` (`owner_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `session_summary`;
CREATE TABLE `session_summary` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `session_id` varchar(100) NOT NULL DEFAULT '',
 `visitor_id` varchar(100) NOT NULL DEFAULT '',
 `display_name` varchar(100) NOT NULL DEFAULT '',
 `avatar` varchar(500) NOT NULL DEFAULT '',
 `owner_id` varchar(100) NOT NULL DEFAULT '',
 `sticky_owner_id` varchar(100) NOT NULL DEFAULT '',
 `route_status` varchar(50) NOT NULL DEFAULT '',
 `queue_name` varchar(100) NOT NULL DEFAULT '',
 `preferred_skill` varchar(100) NOT NULL DEFAULT '',
 `last_message` varchar(2048) NOT NULL DEFAULT '',
 `last_message_at` timestamp NULL DEFAULT NULL,
 `unread_count` int(11) NOT NULL DEFAULT '0',
 `last_route_reason` varchar(255) NOT NULL DEFAULT '',
 `visitor_status` tinyint(4) NOT NULL DEFAULT '0',
 `queue_entered_at` timestamp NULL DEFAULT NULL,
 `last_assign_attempt_at` timestamp NULL DEFAULT NULL,
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `idx_session_summary_session_id` (`session_id`),
 KEY `idx_session_summary_owner_id` (`owner_id`),
 KEY `idx_session_summary_route_status` (`route_status`),
 KEY `idx_session_summary_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `outbox_event`;
CREATE TABLE `outbox_event` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `event_type` varchar(100) NOT NULL DEFAULT '',
 `aggregate_type` varchar(100) NOT NULL DEFAULT '',
 `aggregate_id` varchar(100) NOT NULL DEFAULT '',
 `payload` text,
 `status` varchar(50) NOT NULL DEFAULT 'pending',
 `attempts` int(11) NOT NULL DEFAULT '0',
 `last_error` text,
 `next_retry_at` timestamp NULL DEFAULT NULL,
 `published_at` timestamp NULL DEFAULT NULL,
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 PRIMARY KEY (`id`),
 KEY `idx_outbox_status` (`status`),
 KEY `idx_outbox_event_type` (`event_type`),
 KEY `idx_outbox_next_retry_at` (`next_retry_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `visitor`;
CREATE TABLE `visitor` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `name` varchar(50) NOT NULL DEFAULT '',
 `avator` varchar(500) NOT NULL DEFAULT '',
 `source_ip` varchar(50) NOT NULL DEFAULT '',
 `to_id` varchar(50) NOT NULL DEFAULT '',
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 `deleted_at` timestamp NULL DEFAULT NULL,
 `visitor_id` varchar(100) NOT NULL DEFAULT '',
 `status` tinyint(4) NOT NULL DEFAULT '0',
 `refer` varchar(500) NOT NULL DEFAULT '',
 `last_message` varchar(500) NOT NULL DEFAULT '',
 `city` varchar(100) NOT NULL DEFAULT '',
 `client_ip` varchar(100) NOT NULL DEFAULT '',
 `extra` varchar(2048) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 UNIQUE KEY `visitor_id` (`visitor_id`),
 KEY `to_id` (`to_id`),
 KEY `idx_update` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `message`;
CREATE TABLE `message` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `kefu_id` varchar(100) NOT NULL DEFAULT '',
 `visitor_id` varchar(100) NOT NULL DEFAULT '',
 `content` varchar(2048) NOT NULL DEFAULT '',
 `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `updated_at` timestamp NULL DEFAULT NULL,
 `deleted_at` timestamp NULL DEFAULT NULL,
 `mes_type` enum('kefu','visitor') NOT NULL DEFAULT 'visitor',
 `status` enum('read','unread') NOT NULL DEFAULT 'unread',
 PRIMARY KEY (`id`),
 KEY `kefu_id` (`kefu_id`),
 KEY `visitor_id` (`visitor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

DROP TABLE IF EXISTS `ipblack`;
CREATE TABLE `ipblack` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `ip` varchar(100) NOT NULL DEFAULT '',
 `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
 `kefu_id` varchar(100) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 UNIQUE KEY `ip` (`ip`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `config`;
CREATE TABLE `config` (
 `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
 `conf_name` varchar(255) NOT NULL DEFAULT '',
 `conf_key` varchar(255) NOT NULL DEFAULT '',
 `conf_value` varchar(255) NOT NULL DEFAULT '',
 `user_id` varchar(500) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 KEY `conf_key` (`conf_key`),
 KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Announcement', 'AllNotice', 'Open source customer support system at your service','agent');
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Offline Message', 'OfflineMessage', 'I am currently offline and will reply to you later!','agent');
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Welcome Message', 'WelcomeMessage', 'How may I help you?','agent');
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Email Address (SMTP)', 'NoticeEmailSmtp', '','agent');
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Email Account', 'NoticeEmailAddress', '','agent');
INSERT INTO `config` (`id`, `conf_name`, `conf_key`, `conf_value`, `user_id`) VALUES
(NULL, 'Email Password (SMTP)', 'NoticeEmailPassword', '','agent');


DROP TABLE IF EXISTS `reply_group`;
CREATE TABLE `reply_group` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `group_name` varchar(50) NOT NULL DEFAULT '',
 `user_id` varchar(50) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
INSERT INTO `reply_group` (`id`, `group_name`, `user_id`) VALUES
(NULL, 'Frequently Asked Questions', 'agent');


DROP TABLE IF EXISTS `reply_item`;
CREATE TABLE `reply_item` (
 `id` int(11) NOT NULL AUTO_INCREMENT,
 `content` varchar(1024) NOT NULL DEFAULT '',
 `group_id` int(11) NOT NULL DEFAULT '0',
 `user_id` varchar(50) NOT NULL DEFAULT '',
 `item_name` varchar(50) NOT NULL DEFAULT '',
 PRIMARY KEY (`id`),
 KEY `user_id` (`user_id`),
 KEY `group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
