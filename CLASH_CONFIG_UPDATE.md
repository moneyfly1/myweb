# Clash 配置模板更新说明

## 📝 更新概述

本次更新优化了 Clash 订阅配置模板，为客户提供更完善的分流规则和更好的使用体验。

## ✨ 主要改进

### 1. 优化的配置模板 (`uploads/config/temp.yaml`)
- ✅ 16 个精心设计的代理组
- ✅ 229 条智能分流规则
- ✅ 优化的 DNS 配置（fake-ip 模式）
- ✅ 完善的广告拦截规则
- ✅ AI 服务专用分流规则

### 2. 后端代码增强 (`internal/services/config_update/config_update.go`)
- ✅ 支持 `load-balance`（负载均衡）代理组类型
- ✅ 优化代理组节点注入逻辑
- ✅ 保持配置模板的完整性

### 3. 新增工具
- ✅ 配置验证脚本 (`scripts/verify_clash_config.py`)
- ✅ 完整的配置文档 (`docs/clash_config_template.md`)

## 📦 文件变更

### 修改的文件
1. **uploads/config/temp.yaml** - Clash 配置模板
   - 完全重写，提供专业的分流配置
   - 支持多种使用场景

2. **internal/services/config_update/config_update.go** - 配置生成服务
   - 第 1158 行：添加 `load-balance` 类型支持
   - 第 1174 行：更新节点注入逻辑

### 新增的文件
1. **docs/clash_config_template.md** - 配置模板文档
   - 详细说明每个代理组的用途
   - 规则说明和自定义指南
   - 故障排除指南

2. **scripts/verify_clash_config.py** - 配置验证工具
   - 验证 YAML 语法
   - 检查配置完整性
   - 统计规则信息

3. **CLASH_CONFIG_UPDATE.md** - 本文件，更新说明

## 🎯 代理组说明

| 代理组 | 类型 | 用途 |
|--------|------|------|
| 🚀 节点选择 | select | 主选择组，包含所有选项 |
| ♻️ 自动选择 | url-test | 自动选择最快节点 |
| 🔰 故障转移 | fallback | 节点故障自动切换 |
| 🔮 负载均衡 | load-balance | 流量负载均衡 |
| 🎬 流媒体 | select | 通用流媒体服务 |
| 🌍 国际媒体 | select | YouTube, Netflix 等 |
| 📺 港台番剧 | select | Bilibili, 巴哈姆特 |
| 📲 Telegram | select | Telegram 专用 |
| Ⓜ️ 微软服务 | select | Microsoft 全系列 |
| 🍎 苹果服务 | select | Apple 全系列 |
| 📢 谷歌服务 | select | Google 全系列 |
| 🤖 OpenAI | select | AI 服务专用 |
| 🎮 游戏平台 | select | Steam, Epic 等 |
| 🎯 全球直连 | select | 直连流量 |
| 🛑 广告拦截 | select | 广告过滤 |
| 🐟 漏网之鱼 | select | 未匹配规则 |

## 📊 规则统计

- **总规则数**: 229 条
- **域名规则**: 210 条
- **IP 规则**: 17 条
- **GeoIP 规则**: 1 条
- **最终规则**: 1 条

### 规则分类
- 🛑 广告拦截: 70 条
- 🎯 全球直连: 33 条
- 🌍 国际媒体: 30 条
- Ⓜ️ 微软服务: 15 条
- 🚀 节点选择: 14 条
- 🎮 游戏平台: 12 条
- 🍎 苹果服务: 9 条
- 🤖 OpenAI: 8 条
- 📢 谷歌服务: 8 条
- 📲 Telegram: 7 条
- 其他: 23 条

## 🚀 如何使用

### 客户端使用
客户只需要：
1. 获取订阅链接
2. 在 Clash 客户端中添加订阅
3. 更新订阅即可获得完整配置

### 管理员自定义
1. 编辑 `uploads/config/temp.yaml`
2. 运行验证脚本确认配置正确：
   ```bash
   python3 scripts/verify_clash_config.py
   ```
3. 保存后，新的订阅请求将使用更新后的配置

## 🔍 验证配置

运行验证脚本：
```bash
cd /Users/apple/Downloads/goweb
python3 scripts/verify_clash_config.py
```

输出示例：
```
✅ YAML 配置解析成功!

📋 基本配置:
  - 端口: 7890
  - SOCKS 端口: 7891
  - 模式: Rule
  - 日志级别: info

👥 代理组配置: 16 个
📜 分流规则: 229 条
🔍 配置完整性检查:
  ✅ 所有检查项通过!
```

## 💡 技术细节

### 后端处理流程
1. 读取 `uploads/config/temp.yaml` 模板
2. 解析 YAML 配置
3. 获取用户订阅的实际节点列表
4. 替换 `proxies` 字段为实际节点
5. 遍历 `proxy-groups`，注入节点名称：
   - **url-test/fallback/load-balance**: 只添加实际节点
   - **select**: 保留特殊选项（DIRECT/REJECT）和组名，添加实际节点
6. 保留所有 `rules`
7. 生成最终的 YAML 配置返回给客户端

### 支持的代理组类型
- ✅ `select` - 手动选择
- ✅ `url-test` - 自动测速
- ✅ `fallback` - 故障转移
- ✅ `load-balance` - 负载均衡（本次新增）

### 支持的节点类型
- VMess
- VLess
- Trojan
- Shadowsocks (SS)
- ShadowsocksR (SSR)
- Hysteria
- Hysteria2
- TUIC

## 📚 相关文档

- [Clash 配置模板详细文档](docs/clash_config_template.md)
- [Clash 官方文档](https://github.com/Dreamacro/clash/wiki/configuration)
- [Clash Meta 文档](https://wiki.metacubex.one/)

## ⚠️ 注意事项

1. **不要手动修改 `proxies` 字段**：该字段会被后端自动替换
2. **MATCH 规则必须在最后**：确保作为兜底规则
3. **测试配置**：修改后建议使用验证脚本检查
4. **备份原配置**：修改前建议备份原始配置

## 🎉 更新效果

客户订阅后将获得：
- ✅ 开箱即用的专业配置
- ✅ 智能分流，提升访问速度
- ✅ 广告拦截，优化浏览体验
- ✅ AI 服务优化，更好的 ChatGPT/Claude 体验
- ✅ 流媒体分流，解锁更多内容
- ✅ 游戏加速优化
- ✅ 隐私保护增强

## 🔧 故障排除

### 配置不生效？
1. 检查文件路径是否正确：`uploads/config/temp.yaml`
2. 运行验证脚本检查语法
3. 查看后端日志了解详细错误

### 节点显示不正常？
- 后端会自动填充节点，无需手动添加
- 确保订阅有效且包含节点

### 规则不匹配？
- 检查规则顺序，优先级由上到下
- 确保有 MATCH 兜底规则

## 📞 技术支持

如有问题，请查看：
1. 配置文档：`docs/clash_config_template.md`
2. 系统日志：`logs/app.log`
3. 验证工具：`scripts/verify_clash_config.py`

---

**更新日期**: 2026-02-02
**版本**: v1.0
