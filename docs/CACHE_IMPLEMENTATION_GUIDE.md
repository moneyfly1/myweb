# 缓存实施指南

## 已创建的缓存服务

### 位置
`internal/services/cache_service/cache_service.go`

### 功能
- 用户信息缓存
- 套餐列表缓存
- 公告列表缓存
- 系统配置缓存

## 使用方法

### 1. 套餐列表缓存示例

**优化前**：
```go
func GetPackages(c *gin.Context) {
    db := database.GetDB()
    
    var packages []models.Package
    db.Where("is_active = ?", true).Order("sort_order ASC").Find(&packages)
    
    // ... 格式化数据
    utils.SuccessResponse(c, http.StatusOK, "", result)
}
```

**优化后**：
```go
func GetPackages(c *gin.Context) {
    cacheService := cache_service.NewCacheService()
    
    // 尝试从缓存获取
    if cached, ok := cacheService.GetPackagesCache(); ok {
        utils.SuccessResponse(c, http.StatusOK, "", cached)
        return
    }
    
    // 缓存未命中，查询数据库
    db := database.GetDB()
    var packages []models.Package
    db.Where("is_active = ?", true).Order("sort_order ASC").Find(&packages)
    
    // 格式化数据
    result := make([]map[string]interface{}, 0)
    for _, pkg := range packages {
        result = append(result, map[string]interface{}{
            "id":             pkg.ID,
            "name":           pkg.Name,
            "price":          pkg.Price,
            // ... 其他字段
        })
    }
    
    // 异步写入缓存
    go cacheService.SetPackagesCache(result)
    
    utils.SuccessResponse(c, http.StatusOK, "", result)
}
```

**清除缓存**（套餐变更时）：
```go
func UpdatePackage(c *gin.Context) {
    // ... 更新套餐逻辑
    
    // 清除缓存
    cacheService := cache_service.NewCacheService()
    cacheService.ClearPackagesCache()
    
    utils.SuccessResponse(c, http.StatusOK, "更新成功", nil)
}
```

### 2. 用户信息缓存示例

**优化前**：
```go
func GetUserInfo(c *gin.Context) {
    user, _ := middleware.GetCurrentUser(c)
    
    db := database.GetDB()
    var freshUser models.User
    db.Preload("UserLevel").First(&freshUser, user.ID)
    
    // ... 返回用户信息
}
```

**优化后**：
```go
func GetUserInfo(c *gin.Context) {
    user, _ := middleware.GetCurrentUser(c)
    cacheService := cache_service.NewCacheService()
    
    // 尝试从缓存获取
    if cached, ok := cacheService.GetUserCache(user.ID); ok {
        utils.SuccessResponse(c, http.StatusOK, "", cached)
        return
    }
    
    // 缓存未命中，查询数据库
    db := database.GetDB()
    var freshUser models.User
    db.Preload("UserLevel").First(&freshUser, user.ID)
    
    // 格式化数据
    userInfo := map[string]interface{}{
        "id":       freshUser.ID,
        "username": freshUser.Username,
        "email":    freshUser.Email,
        "balance":  freshUser.Balance,
        // ... 其他字段
    }
    
    // 异步写入缓存
    go cacheService.SetUserCache(user.ID, userInfo)
    
    utils.SuccessResponse(c, http.StatusOK, "", userInfo)
}
```

**清除缓存**（用户信息变更时）：
```go
func UpdateUserInfo(c *gin.Context) {
    user, _ := middleware.GetCurrentUser(c)
    
    // ... 更新用户信息逻辑
    
    // 清除缓存
    cacheService := cache_service.NewCacheService()
    cacheService.ClearUserCache(user.ID)
    
    utils.SuccessResponse(c, http.StatusOK, "更新成功", nil)
}
```

## 需要添加缓存的位置

### 高优先级

1. **套餐列表** - `internal/api/handlers/package.go`
   - `GetPackages()` - 添加缓存读取
   - `CreatePackage()` - 添加缓存清除
   - `UpdatePackage()` - 添加缓存清除
   - `DeletePackage()` - 添加缓存清除

2. **用户信息** - `internal/api/handlers/dashboard.go`
   - `GetUserDashboard()` - 添加用户信息缓存
   - `internal/api/handlers/user.go` 中所有获取用户信息的地方

3. **公告列表** - `internal/api/handlers/announcement.go`
   - `GetAnnouncements()` - 添加缓存读取
   - `CreateAnnouncement()` - 添加缓存清除
   - `UpdateAnnouncement()` - 添加缓存清除
   - `DeleteAnnouncement()` - 添加缓存清除

### 中优先级

4. **系统配置** - `internal/api/handlers/config.go`
   - 所有读取配置的地方添加缓存
   - 所有更新配置的地方清除缓存

## 缓存失效策略

### 自动失效（TTL）
- 用户信息：10分钟
- 套餐列表：30分钟
- 公告列表：10分钟
- 系统配置：1小时

### 主动失效（数据变更时）
```go
// 示例：套餐变更时清除缓存
cacheService := cache_service.NewCacheService()
cacheService.ClearPackagesCache()
```

## 性能监控

### 添加缓存命中率日志
```go
if cached, ok := cacheService.GetPackagesCache(); ok {
    // 缓存命中
    log.Printf("Cache HIT: packages:list:active")
    utils.SuccessResponse(c, http.StatusOK, "", cached)
    return
}
// 缓存未命中
log.Printf("Cache MISS: packages:list:active")
```

## 注意事项

1. **缓存数据格式**
   - 使用 `map[string]interface{}` 或 `[]map[string]interface{}`
   - 确保数据可以 JSON 序列化

2. **异步写入缓存**
   - 使用 `go` 关键字异步写入，不阻塞响应
   - 写入失败不影响主流程

3. **缓存清除时机**
   - 数据创建时：清除列表缓存
   - 数据更新时：清除详情缓存和列表缓存
   - 数据删除时：清除详情缓存和列表缓存

4. **降级策略**
   - Redis 不可用时自动降级
   - 缓存数据异常时自动删除并重新查询

## 实施步骤

### 第一步：套餐列表缓存（最简单，效果最明显）
1. 修改 `GetPackages()` 添加缓存读取
2. 修改 `CreatePackage()`, `UpdatePackage()`, `DeletePackage()` 添加缓存清除
3. 测试验证

### 第二步：用户信息缓存（影响最大）
1. 修改 `GetUserDashboard()` 添加缓存
2. 修改所有用户信息变更的地方添加缓存清除
3. 测试验证

### 第三步：公告列表缓存
1. 修改 `GetAnnouncements()` 添加缓存
2. 修改公告增删改添加缓存清除
3. 测试验证

## 测试方法

### 1. 功能测试
```bash
# 第一次请求（缓存未命中）
curl http://localhost:8080/api/v1/packages

# 第二次请求（缓存命中，应该更快）
curl http://localhost:8080/api/v1/packages
```

### 2. 性能测试
```bash
# 使用 ab 测试
ab -n 1000 -c 10 http://localhost:8080/api/v1/packages

# 对比优化前后的响应时间
```

### 3. 缓存验证
```bash
# 查看 Redis 中的缓存
redis-cli KEYS "packages:*"
redis-cli GET "packages:list:active"

# 查看缓存过期时间
redis-cli TTL "packages:list:active"
```

## 预期效果

### 套餐列表
- 优化前：50-100ms
- 优化后：1-5ms（缓存命中）
- 提升：10-100倍

### 用户信息
- 优化前：30-80ms
- 优化后：1-5ms（缓存命中）
- 提升：6-80倍

### 公告列表
- 优化前：20-50ms
- 优化后：1-5ms（缓存命中）
- 提升：4-50倍

## 下一步

完成第一阶段后，可以继续实施：
- 统计数据缓存
- 邀请码缓存
- 其他低频数据缓存
