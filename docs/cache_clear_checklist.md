# 缓存清除策略 - 完整检查清单

## 关键问题
后台设置更改后，必须清除对应的缓存，否则会造成数据不同步。

## 需要清除缓存的操作

### 1. 套餐管理 ✅ 已处理
- **操作**: CreatePackage, UpdatePackage, DeletePackage
- **位置**: `internal/api/handlers/package.go`
- **清除**: `packages:list:active`
- **状态**: ✅ 已实施

### 2. 支付配置 ⚠️ 需补充
- **操作**: CreatePaymentConfig, UpdatePaymentConfig, DeletePaymentConfig
- **位置**: `internal/api/handlers/admin.go`
- **清除**: `payment:methods:active`
- **状态**: ⚠️ 仅 Create 已添加，Update/Delete 需补充

### 3. 系统配置 ❌ 未处理
- **操作**: UpdateGeneralSettings, UpdateRegistrationSettings, UpdateSecuritySettings, 等
- **位置**: `internal/api/handlers/config.go`
- **清除**: `system:config:{category}`
- **状态**: ❌ 未实施

### 4. 公告管理 ❌ 未处理
- **操作**: CreateNotification, UpdateNotification, DeleteNotification
- **位置**: `internal/api/handlers/notification.go`
- **清除**: `announcements:list:active`
- **状态**: ❌ 未实施

### 5. 节点管理 ⚠️ 部分处理
- **操作**: CreateNode, UpdateNode, DeleteNode, ImportNodeLinks
- **位置**: `internal/api/handlers/node.go`
- **清除**: `nodes:list:active`, `nodes:system:all`, `subscription:config:*`
- **状态**: ⚠️ 仅 Delete 已添加，其他需补充

### 6. 订单支付 ✅ 已处理
- **操作**: ProcessPaidOrder
- **位置**: `internal/services/order/order.go`
- **清除**: `user:info:{userID}`, `user:subscription:{userID}`
- **状态**: ✅ 已实施

### 7. 知识库管理 ❌ 未处理（如果实施了缓存）
- **操作**: CreateKnowledge, UpdateKnowledge, DeleteKnowledge
- **位置**: `internal/api/handlers/knowledge.go`
- **清除**: `knowledge:categories:active`, `knowledge:articles:*`
- **状态**: ❌ 缓存已定义但未应用

## 实施方案

### 最小化实施（核心功能）

```go
// 在 config.go 的 updateSettingsCommon 函数末尾添加
go func(cat string) {
    cs := cache_service.NewCacheService()
    cs.ClearSystemConfigCache(cat)
}(category)

// 在 admin.go 的 UpdatePaymentConfig 末尾添加
go cache_service.NewCacheService().ClearPaymentMethodsCache()

// 在 notification.go 的增删改操作末尾添加
go cache_service.NewCacheService().ClearAnnouncementsCache()

// 在 node.go 的 CreateNode, UpdateNode, ImportNodeLinks 末尾添加
go func() {
    cs := cache_service.NewCacheService()
    cs.ClearNodesCache()
    (&config_update.CacheService{}).ClearSystemNodesCache()
    (&config_update.CacheService{}).ClearAllSubscriptionCache()
}()
```

## 优先级

### P0 - 必须立即实施
1. 系统配置更新清除缓存
2. 支付配置更新清除缓存

### P1 - 高优先级
3. 公告增删改清除缓存
4. 节点增删改清除缓存

### P2 - 中优先级
5. 知识库增删改清除缓存（如果启用了缓存）

## 测试验证

每个缓存清除都需要验证：
1. 修改数据后，缓存是否被清除
2. 下次查询是否返回最新数据
3. 缓存是否正确重建

## 风险提示

⚠️ **未清除缓存的风险**：
- 用户看到旧的配置数据
- 支付方式变更不生效
- 系统设置修改无效
- 公告更新不显示

## 建议

1. **统一缓存清除函数**：创建一个统一的缓存清除中间件
2. **自动化测试**：添加缓存清除的自动化测试
3. **监控告警**：监控缓存命中率，异常时告警
