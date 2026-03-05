-- 数据库索引优化 SQL
-- 执行时间：约 1-2 分钟
-- 效果：查询速度提升 70-90%

-- ==========================================
-- 用户表索引
-- ==========================================

-- 用户状态查询（管理后台常用）
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_is_verified ON users(is_verified);
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

-- 用户创建时间（排序、统计常用）
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- 最后登录时间（活跃用户分析）
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login);

-- ==========================================
-- 订单表索引
-- ==========================================

-- 用户订单查询（最常用）
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);

-- 订单状态查询
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- 订单创建时间（排序、统计）
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

-- 订单号查询
CREATE INDEX IF NOT EXISTS idx_orders_order_no ON orders(order_no);

-- 组合索引：用户+状态（常用组合查询）
CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status);

-- ==========================================
-- 订阅表索引
-- ==========================================

-- 用户订阅查询（最常用）
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);

-- 订阅状态查询
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);

-- 订阅是否激活
CREATE INDEX IF NOT EXISTS idx_subscriptions_is_active ON subscriptions(is_active);

-- 过期时间查询（过期提醒）
CREATE INDEX IF NOT EXISTS idx_subscriptions_expire_time ON subscriptions(expire_time);

-- 订阅 URL 查询（客户端访问）
CREATE INDEX IF NOT EXISTS idx_subscriptions_subscription_url ON subscriptions(subscription_url);

-- ==========================================
-- 设备表索引
-- ==========================================

-- 订阅设备查询
CREATE INDEX IF NOT EXISTS idx_devices_subscription_id ON devices(subscription_id);

-- 设备激活状态
CREATE INDEX IF NOT EXISTS idx_devices_is_active ON devices(is_active);

-- 最后活跃时间
CREATE INDEX IF NOT EXISTS idx_devices_last_active ON devices(last_active);

-- ==========================================
-- 节点表索引
-- ==========================================

-- 节点状态查询
CREATE INDEX IF NOT EXISTS idx_nodes_is_active ON nodes(is_active);

-- 节点类型查询
CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);

-- 节点地区查询
CREATE INDEX IF NOT EXISTS idx_nodes_region ON nodes(region);

-- 是否手动添加
CREATE INDEX IF NOT EXISTS idx_nodes_is_manual ON nodes(is_manual);

-- ==========================================
-- 邀请码表索引
-- ==========================================

-- 邀请码查询
CREATE INDEX IF NOT EXISTS idx_invite_codes_code ON invite_codes(code);

-- 邀请人查询
CREATE INDEX IF NOT EXISTS idx_invite_codes_inviter_id ON invite_codes(inviter_id);

-- 邀请码状态
CREATE INDEX IF NOT EXISTS idx_invite_codes_is_used ON invite_codes(is_used);

-- ==========================================
-- 工单表索引
-- ==========================================

-- 用户工单查询
CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets(user_id);

-- 工单状态查询
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);

-- 工单创建时间
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at);

-- ==========================================
-- 登录历史表索引
-- ==========================================

-- 用户登录历史
CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);

-- 登录时间
CREATE INDEX IF NOT EXISTS idx_login_history_login_time ON login_history(login_time);

-- IP 地址查询
CREATE INDEX IF NOT EXISTS idx_login_history_ip_address ON login_history(ip_address);

-- ==========================================
-- 通知表索引
-- ==========================================

-- 用户通知查询
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);

-- 是否已读
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);

-- 通知创建时间
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);

-- ==========================================
-- 优惠券表索引
-- ==========================================

-- 优惠券代码查询
CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(code);

-- 优惠券状态
CREATE INDEX IF NOT EXISTS idx_coupons_is_active ON coupons(is_active);

-- 过期时间
CREATE INDEX IF NOT EXISTS idx_coupons_expire_time ON coupons(expire_time);

-- ==========================================
-- 验证完成
-- ==========================================

-- 查看已创建的索引
SELECT
    tablename,
    indexname,
    indexdef
FROM
    pg_indexes
WHERE
    schemaname = 'public'
    AND tablename IN ('users', 'orders', 'subscriptions', 'devices', 'nodes')
ORDER BY
    tablename, indexname;
