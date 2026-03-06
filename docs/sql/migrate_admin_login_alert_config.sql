-- 迁移脚本：将 admin_abnormal_login_alert_enabled 从 security 类别移到 admin_notification 类别
-- 执行时间：2026-03-06

-- 1. 更新已存在的配置项的类别
UPDATE system_configs
SET category = 'admin_notification'
WHERE key = 'admin_abnormal_login_alert_enabled'
  AND category = 'security';

-- 2. 如果不存在，则创建默认配置（值为 true）
INSERT INTO system_configs (category, `key`, value, type, display_name, description, is_public, sort_order, created_at, updated_at)
SELECT 'admin_notification', 'admin_abnormal_login_alert_enabled', 'true', 'boolean', '管理员账户异常登录告警', '开启后，管理员在新设备或异地登录时会收到邮件与站内告警', 0, 100, NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM system_configs
    WHERE key = 'admin_abnormal_login_alert_enabled'
      AND category = 'admin_notification'
);

-- 验证迁移结果
SELECT category, `key`, value, display_name
FROM system_configs
WHERE `key` = 'admin_abnormal_login_alert_enabled';
