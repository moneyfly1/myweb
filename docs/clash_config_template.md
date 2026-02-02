# Clash 配置模板说明

## 概述

本系统使用 `/uploads/config/temp.yaml` 作为 Clash 配置模板，当客户订阅时，系统会自动：
1. 读取模板配置
2. 将实际的节点列表注入到配置中
3. 更新代理组的节点引用
4. 保留所有自定义规则

## 配置特性

### 基础配置
- **端口设置**: HTTP 代理端口 7890，SOCKS5 端口 7891
- **局域网共享**: 已启用，可以让同一网络内的其他设备使用
- **运行模式**: 规则模式（Rule）
- **外部控制**: 端口 9090，可用于 Clash Dashboard

### DNS 配置
- **增强模式**: fake-ip 模式，提供更好的性能
- **国内 DNS**: 使用腾讯 DNS (119.29.29.29) 和阿里 DNS (223.5.5.5)
- **国际 DNS**: Cloudflare 和 Google 的 DoH/DoT 服务
- **智能分流**: 根据 GeoIP 自动选择最佳 DNS

### 代理组

#### 1. 🚀 节点选择
- **类型**: 手动选择
- **用途**: 主要的节点选择组
- **选项**: 
  - ♻️ 自动选择
  - 🔰 故障转移
  - 🔮 负载均衡
  - DIRECT（直连）
  - 所有实际节点

#### 2. ♻️ 自动选择
- **类型**: url-test（自动测速）
- **用途**: 自动选择延迟最低的节点
- **测速地址**: http://www.gstatic.com/generate_204
- **测速间隔**: 300 秒
- **容差**: 50ms

#### 3. 🔰 故障转移
- **类型**: fallback
- **用途**: 当前节点不可用时自动切换到下一个
- **检测间隔**: 300 秒

#### 4. 🔮 负载均衡
- **类型**: load-balance
- **用途**: 将流量分散到多个节点
- **策略**: consistent-hashing（一致性哈希）

#### 5. 🎬 流媒体
- **类型**: 手动选择
- **用途**: 用于流媒体服务的通用组

#### 6. 🌍 国际媒体
- **类型**: 手动选择
- **用途**: YouTube, Netflix, Disney+, HBO, Spotify 等国际流媒体

#### 7. 📺 港台番剧
- **类型**: 手动选择
- **用途**: Bilibili, 巴哈姆特等港台地区内容

#### 8. 📲 Telegram
- **类型**: 手动选择
- **用途**: Telegram 消息服务

#### 9. Ⓜ️ 微软服务
- **类型**: 手动选择
- **用途**: Microsoft, Office, OneDrive, Xbox 等
- **默认**: 直连

#### 10. 🍎 苹果服务
- **类型**: 手动选择
- **用途**: Apple, iCloud, App Store 等
- **默认**: 直连

#### 11. 📢 谷歌服务
- **类型**: 手动选择
- **用途**: Google, Gmail, YouTube 等

#### 12. 🤖 OpenAI
- **类型**: 手动选择
- **用途**: OpenAI, ChatGPT, Claude AI 等 AI 服务

#### 13. 🎮 游戏平台
- **类型**: 手动选择
- **用途**: Steam, Epic, Blizzard, EA, Ubisoft 等
- **默认**: 直连

#### 14. 🎯 全球直连
- **类型**: 手动选择
- **用途**: 不需要代理的流量
- **默认**: DIRECT

#### 15. 🛑 广告拦截
- **类型**: 手动选择
- **用途**: 拦截广告和追踪
- **默认**: REJECT

#### 16. 🐟 漏网之鱼
- **类型**: 手动选择
- **用途**: 所有未匹配规则的流量
- **默认**: 使用节点选择

## 分流规则

### 规则优先级
规则按照从上到下的顺序匹配，第一个匹配的规则将被应用。

### 规则分类

#### 1. 本地网络
- 局域网地址（192.168.x.x, 10.x.x.x 等）
- 本地回环（127.0.0.1）
- **动作**: 直连

#### 2. AI 服务
- OpenAI (openai.com, chatgpt.com)
- Claude AI (anthropic.com, claude.ai)
- **动作**: 使用 🤖 OpenAI 组

#### 3. 即时通讯
- Telegram 及其 IP 段
- **动作**: 使用 📲 Telegram 组

#### 4. 国际流媒体
- YouTube
- Netflix
- Disney+
- HBO / HBO Max
- Spotify
- TikTok
- Twitch
- **动作**: 使用 🌍 国际媒体组

#### 5. 社交媒体
- Twitter / X
- Facebook / Instagram
- **动作**: 使用 🚀 节点选择

#### 6. 港台番剧
- Bilibili
- 巴哈姆特
- **动作**: 使用 📺 港台番剧组

#### 7. 科技服务
- Google 全系列服务 → 📢 谷歌服务
- Microsoft 全系列服务 → Ⓜ️ 微软服务
- Apple 全系列服务 → 🍎 苹果服务

#### 8. 游戏平台
- Steam
- Epic Games
- Battle.net / Blizzard
- EA / Origin
- Ubisoft
- PlayStation
- Riot Games
- **动作**: 使用 🎮 游戏平台组（默认直连）

#### 9. GitHub
- 所有 GitHub 相关域名
- **动作**: 使用 🚀 节点选择

#### 10. 广告拦截
- 常见广告关键词（adservice, analytics 等）
- 广告域名（doubleclick.net, googlesyndication.com 等）
- **动作**: 使用 🛑 广告拦截组（默认 REJECT）

#### 11. 中国大陆
- .cn 域名
- 百度、腾讯、阿里巴巴等中国服务
- GeoIP CN（中国 IP 段）
- **动作**: 使用 🎯 全球直连

#### 12. 漏网之鱼
- 所有未匹配的流量
- **动作**: 使用 🐟 漏网之鱼组

## 自定义配置

### 如何修改模板

1. 编辑 `/uploads/config/temp.yaml` 文件
2. 修改代理组配置或规则
3. 保存文件
4. 新的订阅请求将使用更新后的模板

### 注意事项

#### ⚠️ 不要修改的部分
- `proxies` 字段：这会被系统自动替换为实际节点
- 不要删除代理组中的占位配置

#### ✅ 可以修改的部分
- 基础配置（端口、模式等）
- DNS 配置
- 代理组的名称和类型
- 代理组的选项顺序
- 添加或删除规则
- 规则的优先级顺序

### 添加自定义规则

规则语法：
```yaml
rules:
  # 域名后缀匹配
  - DOMAIN-SUFFIX,example.com,🚀 节点选择
  
  # 域名关键词匹配
  - DOMAIN-KEYWORD,google,🚀 节点选择
  
  # 完整域名匹配
  - DOMAIN,www.example.com,🚀 节点选择
  
  # IP 地址段匹配
  - IP-CIDR,192.168.0.0/16,DIRECT,no-resolve
  
  # GeoIP 匹配
  - GEOIP,US,🚀 节点选择
  
  # 最终匹配（必须放在最后）
  - MATCH,🐟 漏网之鱼
```

### 支持的代理组类型

1. **select**: 手动选择
2. **url-test**: 自动测速选择最快节点
3. **fallback**: 故障转移
4. **load-balance**: 负载均衡

## 技术实现

### 后端处理流程

```go
// 位置: internal/services/config_update/config_update.go
// 函数: generateClashYAML()

1. 读取 uploads/config/temp.yaml 模板
2. 解析 YAML 配置
3. 将实际节点转换为代理列表
4. 替换 proxies 字段
5. 遍历 proxy-groups，注入节点名称到各组的 proxies 列表
   - url-test/fallback/load-balance: 只添加实际节点
   - select: 保留特殊选项（DIRECT/REJECT）、组名，并添加实际节点
6. 保留所有 rules
7. 序列化为 YAML 返回给客户端
```

### 支持的节点类型

- VMess
- VLess
- Trojan
- Shadowsocks (SS)
- ShadowsocksR (SSR)
- Hysteria
- Hysteria2
- TUIC

## 最佳实践

### 推荐配置

1. **日常使用**: 使用 ♻️ 自动选择，让系统自动选择最快节点
2. **稳定性优先**: 使用 🔰 故障转移，确保连接稳定
3. **流媒体解锁**: 针对不同服务选择合适的节点
4. **游戏加速**: 建议使用直连或专门的游戏节点
5. **广告拦截**: 启用 🛑 广告拦截可以提升浏览体验

### 性能优化建议

1. **DNS 优化**: 已配置 fake-ip 模式，提供最佳性能
2. **规则优化**: 常用规则放在前面，减少匹配时间
3. **代理组优化**: 
   - url-test 间隔不要太短（推荐 300 秒）
   - fallback 适合需要高可用的场景
   - load-balance 适合下载等大流量场景

## 故障排除

### 常见问题

1. **配置无法加载**
   - 检查 YAML 语法是否正确
   - 确保模板文件存在于 `/uploads/config/temp.yaml`
   - 查看系统日志了解详细错误

2. **节点不显示**
   - 后端会自动填充节点，模板中的占位节点会被替换
   - 确保订阅有效且包含节点

3. **规则不生效**
   - 检查规则优先级，越靠前优先级越高
   - 确保 MATCH 规则在最后

4. **DNS 解析问题**
   - 可以尝试将 fake-ip 改为 redir-host
   - 调整 nameserver 和 fallback 的 DNS 服务器

## 更新日志

### v1.0 (2026-02-02)
- ✅ 初始版本
- ✅ 支持多种代理组类型
- ✅ 完整的分流规则
- ✅ 优化的 DNS 配置
- ✅ 广告拦截规则
- ✅ AI 服务专用规则
- ✅ 流媒体分流规则

## 参考资源

- [Clash 官方文档](https://github.com/Dreamacro/clash/wiki/configuration)
- [Clash Meta 文档](https://wiki.metacubex.one/)
- [规则集合](https://github.com/Loyalsoldier/clash-rules)
