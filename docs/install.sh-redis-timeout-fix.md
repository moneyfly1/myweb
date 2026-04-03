# install.sh Redis 重启超时修复说明

## 问题描述

在运行 `install.sh` 脚本选择选项 11（从 GitHub 同步代码并重新构建）时，脚本会卡在 Redis 重启步骤，无法继续执行后续操作。

## 问题原因

`systemctl restart redis` 命令在某些情况下可能会挂起或长时间无响应，导致脚本阻塞。

## 解决方案

### 1. 添加了带超时的 Redis 重启函数

在脚本中新增了 `restart_redis_with_timeout()` 函数，具有以下特性：

- **10秒超时机制**：如果 Redis 重启超过 10 秒，自动终止并继续执行
- **非阻塞设计**：即使 Redis 重启失败，脚本也会继续执行后续步骤
- **友好提示**：超时或失败时会提示用户手动重启 Redis

### 2. 函数实现原理

```bash
restart_redis_with_timeout() {
    local timeout_seconds=10
    
    # 在后台执行重启命令
    (systemctl restart redis ...) &
    local restart_pid=$!
    
    # 等待最多 10 秒
    local count=0
    while kill -0 $restart_pid 2>/dev/null; do
        if [ $count -ge $timeout_seconds ]; then
            # 超时，杀死进程
            kill -9 $restart_pid 2>/dev/null
            warn "⚠️  Redis 重启超时（10秒），继续执行后续步骤"
            warn "💡 请稍后手动重启 Redis: sudo systemctl restart redis"
            return 1
        fi
        sleep 1
        ((count++))
    done
    
    # 检查 Redis 是否正常运行
    if redis-cli ping &> /dev/null; then
        log "✅ Redis 服务已重启并运行正常"
        return 0
    else
        warn "⚠️  Redis 重启完成但无法连接，请手动检查"
        return 1
    fi
}
```

### 3. 修改位置

已在以下 6 个位置替换了 Redis 重启逻辑：

1. **第 443 行** - 全自动部署中的 Redis 重启
2. **第 549 行** - 进程清理后的 Redis 重启
3. **第 570 行** - 深度清理中的 Redis 重启
4. **第 796 行** - GitHub 同步后的 Redis 重启（选项 11）
5. **第 867 行** - 强制重启服务中的 Redis 重启（选项 3）
6. **第 901 行** - 标准重启服务中的 Redis 重启（选项 8）

## 使用说明

### 正常情况

当 Redis 重启成功时，会显示：
```
[时间] 正在重启 Redis 服务...
[时间] ✅ Redis 服务已重启并运行正常
```

### 超时情况

当 Redis 重启超过 10 秒时，会显示：
```
[时间] 正在重启 Redis 服务...
[WARN] ⚠️  Redis 重启超时（10秒），继续执行后续步骤
[WARN] 💡 请稍后手动重启 Redis: sudo systemctl restart redis
```

脚本会继续执行后续步骤，不会卡住。

### 手动重启 Redis

如果看到超时提示，可以在脚本执行完成后手动重启 Redis：

```bash
# 方法 1：使用 systemctl
sudo systemctl restart redis

# 方法 2：使用 service
sudo service redis restart

# 检查 Redis 状态
sudo systemctl status redis
redis-cli ping
```

## 测试验证

已通过以下测试：

1. ✅ 脚本语法检查通过
2. ✅ 所有 6 处 Redis 重启都已替换为带超时的函数
3. ✅ 超时机制正常工作（10秒后自动终止）
4. ✅ 脚本在 Redis 重启失败时仍能继续执行

## 备份文件

原始文件已备份为：`install.sh.backup`

如需恢复原始版本：
```bash
cp install.sh.backup install.sh
```

## 更新日期

2026/04/04

## 相关问题

如果遇到其他问题，请检查：

1. Redis 服务是否正常安装：`which redis-cli`
2. Redis 服务状态：`systemctl status redis`
3. Redis 配置文件：`/etc/redis/redis.conf`
4. 系统日志：`journalctl -u redis -n 50`
