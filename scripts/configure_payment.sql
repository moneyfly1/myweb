-- 码支付和易支付配置 SQL 脚本
-- 根据截图信息自动配置

-- 商户信息（从截图获取）
-- PID: REDACTED_PID
-- KEY: REDACTED_MERCHANT_KEY
-- 网关: https://REDACTED_GATEWAY/xpay/epay/

-- ============================================
-- 注意: 请先修改下面的域名配置
-- ============================================
SET @domain = 'example.com';  -- 修改为您的实际域名，例如: 'yourdomain.com' 或 'localhost:8080'
SET @use_https = TRUE;         -- 如果是本地环境(localhost)，改为 FALSE

-- ============================================
-- 自动生成回调地址
-- ============================================
SET @protocol = IF(@use_https, 'https://', 'http://');
SET @codepay_notify_url = CONCAT(@protocol, @domain, '/api/v1/payment/notify/codepay');
SET @yipay_notify_url = CONCAT(@protocol, @domain, '/api/v1/payment/notify/yipay');
SET @return_url = CONCAT(@protocol, @domain, '/payment/return');

-- 显示配置信息
SELECT '=== 配置信息 ===' AS info;
SELECT
    '商户ID' AS item, 'REDACTED_PID' AS value
UNION ALL SELECT
    '商户密钥', 'REDACTED_MERCHANT_KEY'
UNION ALL SELECT
    '网关地址', 'https://REDACTED_GATEWAY/xpay/epay/'
UNION ALL SELECT
    '码支付回调地址', @codepay_notify_url
UNION ALL SELECT
    '易支付回调地址', @yipay_notify_url
UNION ALL SELECT
    '同步返回地址', @return_url;

-- ============================================
-- 1. 配置码支付 (Codepay)
-- ============================================
INSERT INTO payment_configs (
    pay_type,
    app_id,
    merchant_private_key,
    notify_url,
    return_url,
    status,
    sort_order,
    config_json,
    created_at,
    updated_at
) VALUES (
    'codepay',
    'REDACTED_PID',
    'REDACTED_MERCHANT_KEY',
    @codepay_notify_url,
    @return_url,
    1,
    10,
    JSON_OBJECT(
        'gateway_url', 'https://REDACTED_GATEWAY/xpay/epay/',
        'api_url', 'https://REDACTED_GATEWAY/xpay/epay/mapi.php',
        'submit_url', 'https://REDACTED_GATEWAY/xpay/epay/submit.php',
        'notify_url', @codepay_notify_url,
        'supported_types', JSON_ARRAY('alipay', 'wxpay')
    ),
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE
    app_id = 'REDACTED_PID',
    merchant_private_key = 'REDACTED_MERCHANT_KEY',
    notify_url = @codepay_notify_url,
    return_url = @return_url,
    config_json = JSON_OBJECT(
        'gateway_url', 'https://REDACTED_GATEWAY/xpay/epay/',
        'api_url', 'https://REDACTED_GATEWAY/xpay/epay/mapi.php',
        'submit_url', 'https://REDACTED_GATEWAY/xpay/epay/submit.php',
        'notify_url', @codepay_notify_url,
        'supported_types', JSON_ARRAY('alipay', 'wxpay')
    ),
    status = 1,
    updated_at = NOW();

SELECT '码支付配置完成' AS result;

-- ============================================
-- 2. 配置易支付 (Yipay)
-- ============================================
INSERT INTO payment_configs (
    pay_type,
    app_id,
    merchant_private_key,
    notify_url,
    return_url,
    status,
    sort_order,
    config_json,
    created_at,
    updated_at
) VALUES (
    'yipay',
    'REDACTED_PID',
    'REDACTED_MERCHANT_KEY',
    @yipay_notify_url,
    @return_url,
    1,
    11,
    JSON_OBJECT(
        'gateway_url', 'https://REDACTED_GATEWAY/xpay/epay/',
        'api_url', 'https://REDACTED_GATEWAY/xpay/epay/mapi.php',
        'submit_url', 'https://REDACTED_GATEWAY/xpay/epay/submit.php',
        'notify_url', @yipay_notify_url,
        'sign_type', 'MD5',
        'supported_types', JSON_ARRAY('alipay', 'wxpay')
    ),
    NOW(),
    NOW()
) ON DUPLICATE KEY UPDATE
    app_id = 'REDACTED_PID',
    merchant_private_key = 'REDACTED_MERCHANT_KEY',
    notify_url = @yipay_notify_url,
    return_url = @return_url,
    config_json = JSON_OBJECT(
        'gateway_url', 'https://REDACTED_GATEWAY/xpay/epay/',
        'api_url', 'https://REDACTED_GATEWAY/xpay/epay/mapi.php',
        'submit_url', 'https://REDACTED_GATEWAY/xpay/epay/submit.php',
        'notify_url', @yipay_notify_url,
        'sign_type', 'MD5',
        'supported_types', JSON_ARRAY('alipay', 'wxpay')
    ),
    status = 1,
    updated_at = NOW();

SELECT '易支付配置完成' AS result;

-- ============================================
-- 3. 验证配置
-- ============================================
SELECT '=== 配置验证 ===' AS info;

SELECT
    id,
    pay_type AS '支付类型',
    app_id AS '商户ID',
    LEFT(merchant_private_key, 10) AS '密钥前10位',
    notify_url AS '回调地址',
    return_url AS '返回地址',
    status AS '状态',
    sort_order AS '排序',
    created_at AS '创建时间',
    updated_at AS '更新时间'
FROM payment_configs
WHERE pay_type IN ('codepay', 'yipay')
ORDER BY sort_order;

-- ============================================
-- 4. 清除支付方法缓存（如果使用了缓存）
-- ============================================
-- 注意: 如果您的系统使用了 Redis 缓存，需要手动清除缓存
-- redis-cli DEL payment_methods_cache

SELECT '=== 配置完成 ===' AS info;
SELECT '请重启应用服务以使配置生效' AS notice;
SELECT '请在码支付平台后台配置回调地址:' AS notice;
SELECT @codepay_notify_url AS '码支付异步通知地址';
SELECT @yipay_notify_url AS '易支付异步通知地址';
SELECT @return_url AS '同步返回地址';

-- ============================================
-- 使用说明
-- ============================================
-- 1. 修改上面的 @domain 变量为您的实际域名
-- 2. 如果是本地环境，将 @use_https 改为 FALSE
-- 3. 执行整个 SQL 脚本
-- 4. 在码支付平台后台配置回调地址
-- 5. 重启应用服务
-- 6. 使用测试脚本测试回调功能
-- ============================================
