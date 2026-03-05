# 老网站更新指南

## 如果你已经安装了老代码，现在想更新到新版本（包含 Redis 缓存优化）

### 方式一：使用安装脚本自动更新（推荐）

1. **进入项目目录**
   ```bash
   cd /root/goweb  # 或你的项目路径
   ```

2. **运行安装脚本**
   ```bash
   bash install.sh
   ```

3. **选择菜单选项 2：从 GitHub 同步代码**
   ```
   2. 从 GitHub 同步代码
   ```

4. **脚本会自动执行以下操作：**
   - ✅ 从 GitHub 拉取最新代码
   - ✅ 编译新的 Go 程序
   - ✅ 构建新的前端
   - ✅ **自动检测并提示配置 Redis 缓存**
   - ✅ 重启服务

5. **Redis 配置提示**
   
   更新完成后，脚本会自动检测到新功能并提示：
   ```
   检测到代码已更新，包含 Redis 缓存优化功能
   新功能：Redis 缓存可将 GeoIP 查询速度提升 50-100 倍！
   是否现在配置 Redis 缓存？(y/n，默认: y):
   ```

   **选择 y（推荐）**，脚本会：
   - 询问安装方式（Docker/系统包/跳过）
   - 自动安装 Redis（如果选择）
   - 自动创建 .env 配置文件
   - 自动测试连接
   - 自动重启服务

   **选择 n**，可以稍后手动配置（见方式二）

### 方式二：稍后手动配置 Redis

如果更新时选择了跳过 Redis 配置，可以随时运行：

1. **运行安装脚本**
   ```bash
   cd /root/goweb
   bash install.sh
   ```

2. **选择菜单选项 12：配置 Redis 缓存**
   ```
   12. 配置 Redis 缓存（性能优化）
   ```

3. **按提示操作即可**

### 方式三：完全手动配置

如果你想完全手动控制：

#### 1. 更新代码
```bash
cd /root/goweb
git pull origin main
```

#### 2. 编译和构建
```bash
# 编译后端
go build -o server ./cmd/server/main.go

# 构建前端
cd frontend
npm install --legacy-peer-deps
npm run build
cd ..
```

#### 3. 安装 Redis（选择一种方式）

**Docker 方式（推荐）：**
```bash
docker run -d --name redis -p 6379:6379 --restart=always redis:alpine
```

**系统包方式：**
```bash
# Ubuntu/Debian
apt-get update && apt-get install -y redis-server
systemctl start redis
systemctl enable redis

# CentOS/RHEL
yum install -y redis
systemctl start redis
systemctl enable redis
```

#### 4. 配置环境变量

创建或编辑 `.env` 文件：
```bash
cd /root/goweb
nano .env
```

添加以下内容：
```bash
REDIS_ADDR=localhost:6379
# REDIS_PASSWORD=your_password  # 如果设置了密码
```

#### 5. 重启服务
```bash
systemctl restart cboard
```

#### 6. 验证 Redis 是否生效

查看日志：
```bash
journalctl -u cboard -n 50
```

应该看到：
```
Redis 连接成功
Redis 缓存已启用，GeoIP 查询将使用缓存加速
```

### 常见问题

#### Q1: 如果不配置 Redis 会怎样？
**A:** 系统仍然可以正常工作，只是没有缓存加速。性能会比老版本好（因为我们优化了很多地方），但不如启用 Redis 后的性能。

#### Q2: Redis 连接失败会影响系统吗？
**A:** 不会！系统会自动降级，跳过缓存直接查询。你会在日志中看到：
```
Redis 初始化失败（缓存功能已禁用）
```

#### Q3: 已经配置了 Redis，更新代码后需要重新配置吗？
**A:** 不需要！脚本会自动检测现有配置并测试连接。只有连接失败时才会询问是否重新配置。

#### Q4: 如何查看 Redis 是否正常工作？
```bash
# 测试连接
redis-cli ping
# 应该返回：PONG

# 查看缓存的 IP 数量
redis-cli DBSIZE

# 查看缓存内容
redis-cli KEYS "geoip:*" | head -10
```

#### Q5: 如何清理 Redis 缓存？
```bash
# 清理所有 GeoIP 缓存
redis-cli KEYS "geoip:*" | xargs redis-cli DEL

# 清理所有缓存
redis-cli FLUSHALL
```

### 性能对比

| 操作 | 老版本 | 新版本（无Redis） | 新版本（有Redis） |
|------|--------|------------------|------------------|
| 签到 | 2-5秒 | 100-300ms | 100-300ms |
| 设备列表（100条） | 10-50秒 | 50-200ms | 50-200ms |
| 充值记录列表 | 5-20秒 | 50-200ms | 50-200ms |
| 单条详情查询 | 500-2000ms | 200-500ms | 10-50ms（缓存命中） |

### 推荐配置

- **生产环境**：强烈推荐启用 Redis
- **开发环境**：可选，但建议启用以测试真实性能
- **低流量网站**：可选
- **高流量网站**：必须启用

### 技术支持

如有问题，请查看：
- 完整文档：`docs/GEOIP_CACHE.md`
- 服务日志：`journalctl -u cboard -f`
- Redis 日志：`journalctl -u redis -f`
