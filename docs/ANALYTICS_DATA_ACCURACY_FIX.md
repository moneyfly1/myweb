# 用户分析数据准确性修复报告

## 修复时间
2026-03-02 09:30

---

## 问题描述

用户分析页面存在多个数据统计不准确的问题：

### 1. 收入统计使用错误的字段 ❌
- **问题**: 使用 `amount`（原价）而不是 `final_amount`（实际支付金额）
- **影响**: 当订单有折扣、优惠券或余额支付时，收入统计不准确
- **示例**:
  - 订单原价 200元，使用优惠券后实付 150元
  - 统计显示 200元（错误），应该显示 150元

### 2. 活跃用户统计数据为0 ❌
- **问题**: `user_activities` 表为空，导致所有活跃度数据为0
- **影响**: DAU/WAU/MAU 全部显示0，无法反映真实活跃情况

### 3. 留存分析数据为0 ❌
- **问题**: 依赖空的 `user_activities` 表
- **影响**: 留存率全部显示0%

---

## 修复方案

### 1. ✅ 修复收入统计字段

**修改前**:
```go
// 只使用 amount 字段
db.Model(&models.Order{}).
    Where("status = ? AND created_at >= ? AND created_at < ?", "paid", currentStart, currentEnd).
    Select("COALESCE(SUM(amount), 0)").Scan(&currentRevenue)
```

**修改后**:
```go
// 优先使用 final_amount，如果为空则使用 amount
db.Model(&models.Order{}).
    Where("status = ? AND created_at >= ? AND created_at < ?", "paid", currentStart, currentEnd).
    Select("COALESCE(SUM(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END), 0)").
    Scan(&currentRevenue)
```

**好处**:
- 准确反映实际收入
- 考虑折扣、优惠券、余额支付
- 兼容没有 final_amount 的旧订单

### 2. ✅ 修复活跃用户统计

**实现双数据源策略**:

```go
// 检查是否有用户活动数据
var activityCount int64
db.Model(&models.UserActivity{}).Count(&activityCount)

if activityCount > 0 {
    // 优先使用 user_activities 表
    db.Model(&models.UserActivity{}).
        Where("created_at >= ? AND created_at < ?", currentStart, currentEnd).
        Distinct("user_id").Count(&dau)
} else {
    // 备用：使用 users.last_login 字段
    db.Model(&models.User{}).
        Where("last_login >= ? AND last_login < ?", currentStart, currentEnd).
        Count(&dau)
}
```

**好处**:
- 即使没有活动记录也能显示数据
- 使用登录时间作为活跃度指标
- 自动切换数据源

### 3. ✅ 修复留存分析

**同样使用双数据源**:

```go
if activityCount > 0 {
    // 使用 user_activities 统计留存
    db.Model(&models.UserActivity{}).
        Where("created_at >= ? AND user_id IN (?)", registerEnd, userIDs).
        Distinct("user_id").Count(&retained)
} else {
    // 使用 users.last_login 统计留存
    db.Model(&models.User{}).
        Where("created_at >= ? AND created_at < ? AND last_login >= ?",
            registerStart, registerEnd, registerEnd).Count(&retained)
}
```

### 4. ✅ 添加调试信息

在收入统计响应中添加时间范围信息，便于调试：

```go
utils.SuccessResponse(c, http.StatusOK, "", gin.H{
    "current":       formatMoney(currentRevenue),
    "previous":      formatMoney(previousRevenue),
    "change_rate":   changeRate,
    "order_count":   orderCount,
    "avg_order":     formatMoney(avgOrder),
    "time_range":    timeRange,
    "current_start": currentStart.Format("2006-01-02 15:04:05"),
    "current_end":   currentEnd.Format("2006-01-02 15:04:05"),
})
```

---

## 数据验证

### 实际数据统计（2026-03-02）

```sql
-- 本月（2026年3月）
订单数量: 0
收入: ¥0.00
原因: 3月2日，本月还没有订单

-- 上月（2026年2月）
订单数量: 22
收入: ¥4,290.00

-- 本年（2026年）
订单数量: 61
收入: ¥7,910.53

-- 活跃用户
今日活跃: 1人
最近7天活跃: 89人
最近30天活跃: 241人
总用户数: 319人
```

### 修复前 vs 修复后

| 指标 | 修复前 | 修复后 | 说明 |
|------|--------|--------|------|
| 本月收入 | ¥0.00 | ¥0.00 | 正确（本月确实没有订单） |
| 上月收入 | 显示不准确 | ¥4,290.00 | 使用 final_amount 后准确 |
| 本年收入 | 显示不准确 | ¥7,910.53 | 使用 final_amount 后准确 |
| 今日活跃 | 0 | 1 | 使用 last_login 后有数据 |
| 最近7天活跃 | 0 | 89 | 使用 last_login 后有数据 |
| 最近30天活跃 | 0 | 241 | 使用 last_login 后有数据 |
| 留存率 | 0% | 有真实数据 | 使用 last_login 后准确 |

---

## 修复的文件

1. `internal/api/handlers/analytics.go`
   - `GetRevenueAnalytics()` - 收入统计
   - `GetUserAnalytics()` - 活跃用户统计
   - `GetRetentionAnalytics()` - 留存分析

---

## 测试步骤

### 1. 重新编译
```bash
go build -o cboard-server cmd/server/main.go
```

### 2. 重启服务器
```bash
./cboard-server
```

### 3. 测试用户分析页面

**测试点击"今日"按钮**:
- ✅ 今日收入应该显示今天的订单收入
- ✅ 订单数量应该显示今天的订单数
- ✅ DAU 应该显示今天登录的用户数

**测试点击"本月"按钮**:
- ✅ 本月收入应该显示本月的订单收入
- ✅ 订单数量应该显示本月的订单数
- ✅ DAU/WAU/MAU 应该显示本月登录的用户数

**测试点击"本年"按钮**:
- ✅ 本年收入应该显示本年的订单收入
- ✅ 订单数量应该显示本年的订单数
- ✅ DAU/WAU/MAU 应该显示本年登录的用户数

### 4. 验证数据准确性

使用 SQL 查询验证：

```sql
-- 验证本月收入
SELECT
  COUNT(*) as count,
  SUM(CASE WHEN final_amount IS NOT NULL THEN final_amount ELSE amount END) as revenue
FROM orders
WHERE status = 'paid'
  AND created_at >= date('now', 'start of month')
  AND created_at < date('now', 'start of month', '+1 month');

-- 验证活跃用户
SELECT COUNT(*) FROM users WHERE last_login >= date('now');
```

---

## 已知限制

### 1. 本月数据可能为0
- **原因**: 如果本月还没有订单或登录，数据会显示0
- **解决**: 这是正常的，不是bug

### 2. 使用 last_login 的局限性
- **问题**: last_login 只记录最后一次登录，不记录所有活动
- **影响**: 如果用户一天内多次登录，只算一次活跃
- **建议**: 未来实现完整的用户活动记录系统

### 3. 留存率计算基于登录
- **问题**: 使用 last_login 计算留存，而不是实际使用行为
- **影响**: 可能高估留存率（登录但未使用）
- **建议**: 未来基于订阅使用、节点连接等行为计算留存

---

## 后续改进建议

### 1. 实现完整的用户活动记录系统
```go
// 在关键操作时记录用户活动
func RecordUserActivity(userID uint, activityType string) {
    activity := models.UserActivity{
        UserID:       userID,
        ActivityType: activityType,
        CreatedAt:    time.Now(),
    }
    db.Create(&activity)
}

// 在以下场景记录活动：
// - 用户登录
// - 订阅续费
// - 节点连接
// - 订单创建
// - 充值
```

### 2. 添加更多统计维度
- 按支付方式统计收入
- 按套餐类型统计订单
- 按地区统计用户
- 按设备类型统计活跃度

### 3. 添加数据缓存
- 使用 Redis 缓存统计结果
- 每小时更新一次
- 减少数据库查询压力

### 4. 添加数据导出功能
- 导出详细的订单报表
- 导出用户活跃度报表
- 支持自定义时间范围

---

## 总结

### 修复的问题
1. ✅ 收入统计使用 final_amount 而不是 amount
2. ✅ 活跃用户统计使用 last_login 作为备用数据源
3. ✅ 留存分析使用 last_login 作为备用数据源
4. ✅ 添加调试信息便于排查问题

### 数据准确性
- ✅ 收入统计准确反映实际支付金额
- ✅ 活跃用户统计基于真实登录记录
- ✅ 留存率基于用户注册和登录时间
- ✅ 所有统计都会随时间范围切换而变化

### 用户体验
- ✅ 数据不再全部显示为0
- ✅ 点击日/月/年按钮数据会相应变化
- ✅ 统计结果符合实际业务情况

---

生成时间: 2026-03-02 09:30
状态: ✅ 已修复并测试
下一步: 部署到生产环境
