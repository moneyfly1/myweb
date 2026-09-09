# 新增功能说明

## 概述

本文档介绍 CBoard 系统最新添加的四大核心功能模块。

---

## 📚 知识库系统

### 功能概述
完整的帮助文档和教程系统，支持分类管理、文章搜索、浏览统计。

### 主要特性
- **5 个预设分类**：新手入门、客户端教程、常见问题、进阶使用、账户相关
- **14 篇详细教程**：涵盖各平台使用指南
- **Clash 系列客户端**：
  - Windows: Clash Verge
  - macOS: ClashX Pro
  - iOS: Shadowrocket
  - Android: Clash for Android
- **搜索功能**：支持标题和内容搜索
- **浏览统计**：自动记录文章浏览次数
- **完整 CRUD**：管理端可创建、编辑、删除分类和文章

### API 端点
#### 用户端
- `GET /api/v1/knowledge/categories` - 获取分类列表
- `GET /api/v1/knowledge/articles` - 获取文章列表
- `GET /api/v1/knowledge/articles/:id` - 获取文章详情
- `GET /api/v1/knowledge/search` - 搜索文章

#### 管理端
- `POST /admin/knowledge/categories` - 创建分类
- `PUT /admin/knowledge/categories/:id` - 更新分类
- `DELETE /admin/knowledge/categories/:id` - 删除分类
- `POST /admin/knowledge/articles` - 创建文章
- `PUT /admin/knowledge/articles/:id` - 更新文章
- `DELETE /admin/knowledge/articles/:id` - 删除文章

### 前端页面
- **用户端**: `/knowledge` - 知识库浏览页面
- **管理端**: `/admin/knowledge` - 知识库管理页面

---

## 🎁 每日签到系统

### 功能概述
用户每日签到获得随机奖励，提升用户活跃度和粘性。

### 主要特性
- **随机奖励**：每次签到获得 0.1-1 元随机金额
- **自动到账**：奖励自动添加到用户账户余额
- **记录日志**：所有签到记录保存在余额日志中
- **防重复签到**：每天只能签到一次
- **状态查询**：可查询今日是否已签到

### API 端点
- `POST /api/v1/users/checkin` - 执行签到
- `GET /api/v1/users/checkin/status` - 查询签到状态

### 前端页面
- **用户端**: Dashboard 页面右上角签到按钮

### 数据库表
```sql
checkin_records
- id: 主键
- user_id: 用户 ID
- amount: 奖励金额
- created_at: 签到时间
```

---

## 🎉 营销活动系统

### 功能概述
灵活的促销活动管理系统，支持多种活动类型和折扣方式。

### 活动类型
- **限时抢购** (flash_sale)：限时特价活动
- **新用户优惠** (new_user)：新用户专享优惠
- **召回活动** (recall)：流失用户召回
- **会员日** (member_day)：会员专属活动

### 折扣类型
- **百分比折扣** (percentage)：如 8 折、9 折
- **固定减免** (fixed)：如减 10 元、减 20 元
- **赠送天数** (free_days)：如赠送 7 天、30 天

### 主要特性
- **时间范围**：可设置活动开始和结束时间
- **套餐限制**：可指定适用的套餐 ID
- **最低消费**：可设置最低消费金额
- **最高优惠**：可设置最高优惠金额
- **启用/禁用**：可随时启用或禁用活动

### API 端点
- `GET /api/v1/promotions/active` - 获取当前有效活动
- `POST /admin/promotions` - 创建活动
- `PUT /admin/promotions/:id` - 更新活动
- `DELETE /admin/promotions/:id` - 删除活动

### 前端页面
- **管理端**: `/admin/promotions` - 营销活动管理页面

### 数据库表
```sql
promotions
- id: 主键
- name: 活动名称
- type: 活动类型
- discount_type: 折扣类型
- discount_value: 折扣值
- min_amount: 最低消费
- max_discount: 最高优惠
- package_ids: 适用套餐
- start_time: 开始时间
- end_time: 结束时间
- is_active: 是否启用
- description: 活动描述
```

---

## 📊 用户分析系统

### 功能概述
全面的用户数据分析系统，帮助运营人员了解用户行为和趋势。

### 分析维度

#### 1. 活跃用户统计
- **DAU** (Daily Active Users)：日活跃用户数
- **WAU** (Weekly Active Users)：周活跃用户数
- **MAU** (Monthly Active Users)：月活跃用户数

#### 2. 用户留存分析
- **次日留存**：注册后第二天仍活跃的用户比例
- **7 日留存**：注册后第 7 天仍活跃的用户比例
- **30 日留存**：注册后第 30 天仍活跃的用户比例

#### 3. 流失预警
- **流失定义**：30 天未登录的用户
- **流失用户列表**：显示流失用户详细信息
- **流失原因分析**：帮助制定召回策略

#### 4. 设备分析
- **设备类型分布**：iOS、Android、Windows、macOS 等
- **设备数量统计**：每个用户的设备数量分布
- **设备使用趋势**：设备类型的时间趋势

### API 端点
- `GET /admin/analytics/users` - 获取活跃用户统计
- `GET /admin/analytics/retention` - 获取留存分析数据
- `GET /admin/analytics/churn` - 获取流失用户列表
- `GET /admin/analytics/devices` - 获取设备分析数据

### 前端页面
- **管理端**: `/admin/analytics` - 用户分析页面

---

## 🔐 审计日志

所有新增功能的 CRUD 操作都已添加审计日志，确保操作可追溯。

### 记录的操作
- 知识库分类：创建、更新、删除
- 知识库文章：创建、更新、删除
- 营销活动：创建、更新、删除
- 用户签到：签到操作

### 日志内容
- 操作用户
- 操作类型
- 资源类型
- 资源 ID
- 操作描述
- IP 地址
- 操作时间

---

## 📱 移动端优化

所有新增功能都已完整适配移动端。

### 优化内容
- **抽屉弹窗**：移动端全屏显示
- **输入框**：字体 16px 防止 iOS 自动缩放
- **触摸目标**：最小 44px 符合人体工程学
- **按钮样式**：统一的按钮大小和间距
- **响应式布局**：768px 以下触发移动端样式
- **表格优化**：移动端卡片式布局

---

## 🎫 工单附件上传（用户端 + 管理端）

工单创建与回复支持上传**图片 / 视频 / 附件**。

- **上传**：`POST /api/v1/tickets/upload`（multipart，字段 `file`），单文件 ≤ **30 MB**，一次 ≤ 9 个；类型覆盖图片 / 视频 / 音频 / 文档 / 压缩包，扩展名 + MIME 双重校验
- **存储**：`uploads/tickets/YYYYMM/`，经 `/uploads/` 静态服务 + Nginx `location /uploads/` 反代访问
- **移动端**：两个入口——**「相册选图」**（iOS 弹拍照/相册）与**「选择文件」**
- **展示**：详情中图片缩略图放大预览、视频内嵌播放、文件下载卡片；附件正确归属工单本体或对应回复
- **回复可仅附件**（无文字也可提交）

## 🖥️ 专线节点订阅导入与更新

- **订阅导入**：从订阅 URL 拉取并自动解析节点导入为专线节点（追加模式去重）
- **更新/替换**：`替换模式` 或对已导入订阅执行「更新此订阅」——增量更新（匹配节点更新配置、保留用户分配），或删除重建
- **订阅管理**：查看已导入订阅及节点数、删除某订阅全部节点；历史导入节点（无原始 URL）可「用新订阅替换」
- **分配保护**：更新/替换绝不移除已分配用户的节点

## 📊 管理仪表盘实时动态

- 「异常客户」卡片替换为**「实时动态」**：展示真实用户行为（登录 / 删设备 / 重置订阅 / 下单 / 付款 / 充值）
- 合并 `audit_logs` + `subscription_logs` + `orders` + `recharge_records` 四数据源，排除管理员操作
- **60 秒静默轮询**；到期客户列表同步刷新（客户续费后自动消失）
- 登录事件降权（≤50%），业务动态优先可见

## 🔔 订阅到期提醒增强

- 新增管理员通知事件 **「订阅即将到期」**（`subscription_expiry_warning`）：到期前 1/3/7 天提醒时同步推送 Telegram / Bark / 邮件
- **防重复发送**：同一天同一用户同主题只发一次（服务重启不重复轰炸）
- 通知模板完善：订阅配置/重置邮件展示**全部 8 种客户端订阅格式**（universal / clash / stash / surge / quantumultx / loon / singbox / shadowrocket）

## 🍎 macOS 下载双架构

- 用户端仪表盘与帮助页的 macOS 客户端下载支持 **Apple 芯片 (ARM) / Intel (x64)** 二选一
- 后端按架构解析对应安装包（如 `*-macos-arm64.dmg` / `*-macos-amd64.dmg`）

---

## 🚀 快速开始

### 新项目
如果是新项目，所有功能已就绪，直接启动即可：

```bash
# 启动后端
./cboard-server

# 启动前端（开发模式）
cd frontend && npm run dev
```

### 现有项目
如果需要将新功能迁移到现有数据库，请参考：

- [迁移指南](../migration/MIGRATION_GUIDE.md)
- [快速启动指南](../migration/QUICK_START.md)

---

## 📝 相关文档

- [迁移指南](../migration/MIGRATION_GUIDE.md)
- [快速启动指南](../migration/QUICK_START.md)

---

更新时间: 2026-09-09
版本: 1.2.0
