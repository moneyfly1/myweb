-- 初始化邮件模板
-- 用于用户分析页面的联系用户功能

-- 订阅即将到期提醒模板
INSERT OR REPLACE INTO email_templates (name, subject, content, variables, is_active, created_at, updated_at)
VALUES (
  'subscription_expiring',
  '【重要提醒】您的订阅即将到期',
  '尊敬的用户 {username}，

您好！

我们注意到您的订阅服务即将在 {expire_date} 到期（剩余 {days_left} 天）。为了确保您的服务不受影响，请及时续费。

续费优惠：
- 现在续费可享受 9 折优惠
- 续费 3 个月及以上可享受 8.5 折优惠

如有任何问题，请随时联系我们。

祝好！
CBoard 团队',
  '{username}, {email}, {expire_date}, {days_left}',
  1,
  datetime('now'),
  datetime('now')
);

-- 订阅已到期通知模板
INSERT OR REPLACE INTO email_templates (name, subject, content, variables, is_active, created_at, updated_at)
VALUES (
  'subscription_expired',
  '【服务到期通知】您的订阅已到期',
  '尊敬的用户 {username}，

您好！

您的订阅服务已于 {expire_date} 到期。为了继续使用我们的服务，请尽快续费。

续费福利：
- 立即续费可获得额外 3 天免费使用时间
- 续费 6 个月及以上可享受 8 折优惠

我们期待继续为您服务！

祝好！
CBoard 团队',
  '{username}, {email}, {expire_date}',
  1,
  datetime('now'),
  datetime('now')
);

-- 流失用户召回模板
INSERT OR REPLACE INTO email_templates (name, subject, content, variables, is_active, created_at, updated_at)
VALUES (
  'user_recall',
  '我们想念您！特别优惠等您来领',
  '尊敬的用户 {username}，

您好！

我们注意到您已经有一段时间没有使用我们的服务了。我们非常想念您！

为了欢迎您回来，我们准备了特别优惠：
- 续费任意套餐可享受 7 折优惠
- 赠送 7 天免费试用时间
- 优先体验新功能

这个优惠仅限 7 天内有效，不要错过哦！

期待您的归来！

祝好！
CBoard 团队',
  '{username}, {email}',
  1,
  datetime('now'),
  datetime('now')
);

-- 查看已创建的模板
SELECT id, name, subject, is_active FROM email_templates WHERE name IN ('subscription_expiring', 'subscription_expired', 'user_recall');
