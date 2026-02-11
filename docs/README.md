# 文档说明 / Documentation

本目录包含部署、故障排查、功能说明与后台配置说明。

This directory contains deployment, troubleshooting, feature docs and backend configuration guides.

---

## 中文

### 部署与故障排查

| 文档 | 说明 |
|------|------|
| [VPS 部署教程（无宝塔）](./VPS部署教程-无宝塔.md) | 纯 VPS 一键部署步骤 |
| [安装问题排查指南](./故障排查/安装问题排查指南.md) | 常见安装问题与解决方法 |

### 功能说明

| 文档 | 说明 |
|------|------|
| [列表功能索引](./功能/列表功能索引.md) | 所有列表功能索引与导航 |
| [用户列表管理功能说明](./功能/用户列表管理功能说明.md) | 用户管理、搜索、批量操作 |
| [订阅管理功能说明](./功能/订阅管理功能说明.md) | 订阅管理、设备数量限制原理 |
| [订单管理功能说明](./功能/订单管理功能说明.md) | 订单处理、支付管理 |
| [套餐管理功能说明](./功能/套餐管理功能说明.md) | 套餐创建、编辑、删除、定价管理 |
| [邀请码管理功能说明](./功能/邀请码管理功能说明.md) | 邀请码生成、管理、奖励设置 |
| [用户等级管理功能说明](./功能/用户等级管理功能说明.md) | 用户等级创建、折扣设置、自动升级 |
| [订阅重置功能说明](./功能/订阅重置功能说明.md) | 用户重置订阅、管理员重置订阅、订阅延长 |
| [节点管理功能说明](./功能/节点管理功能说明.md) | 节点采集、导入、管理 |
| [节点手动导入说明](./功能/节点手动导入说明.md) | 节点链接导入、手动填写、Clash 配置导入 |
| [专线节点管理说明](./功能/专线节点管理说明.md) | 专线节点创建、分配、取消分配、删除、测速 |
| [节点测速说明](./功能/节点测速说明.md) | 节点测速、批量测速、测速原理 |
| [工单管理功能说明](./功能/工单管理功能说明.md) | 工单处理、回复、状态管理 |
| [设备管理功能说明](./功能/设备管理功能说明.md) | 设备查看、删除、限制原理 |
| [设备删除说明](./功能/设备删除说明.md) | 用户删除设备、管理员删除设备 |
| [登录历史管理功能说明](./功能/登录历史管理功能说明.md) | 登录记录、地区信息 |
| [异常用户管理功能说明](./功能/异常用户管理功能说明.md) | 异常用户识别与处理 |
| [数据分析功能说明](./功能/数据分析功能说明.md) | 数据统计、地区分析 |

### 后台与配置说明

#### 支付配置

| 文档 | 说明 |
|------|------|
| [支付宝配置说明](./配置/支付宝配置说明.md) | 支付宝开放平台与系统后台配置 |
| [易支付配置指南](./配置/易支付配置指南.md) | 易支付配置与使用 |

#### 通知配置

| 文档 | 说明 |
|------|------|
| [邮件服务器配置说明](./配置/邮件服务器配置说明.md) | 如何配置 SMTP 邮件服务器 |
| [Telegram 通知配置说明](./配置/Telegram通知配置说明.md) | Telegram Bot 通知配置 |
| [Bark 通知配置说明](./配置/Bark通知配置说明.md) | Bark iOS 推送通知配置 |
| [客户通知设置说明](./配置/客户通知设置说明.md) | 客户邮件通知开关配置 |

#### 系统设置

| 文档 | 说明 |
|------|------|
| [基本设置说明](./配置/基本设置说明.md) | 网站信息、Logo、域名、GeoIP 等 |
| [注册设置说明](./配置/注册设置说明.md) | 注册流程、密码要求、新用户默认订阅 |
| [安全设置与限制说明](./配置/安全设置与限制说明.md) | 登录失败限制、锁定、IP 白名单、解锁方法 |
| [主题设置说明](./配置/主题设置说明.md) | 默认主题、用户自定义权限、可用主题 |
| [公告管理说明](./配置/公告管理说明.md) | 登录公告弹窗配置 |

#### 节点与备份

| 文档 | 说明 |
|------|------|
| [采集地址配置说明](./配置/采集地址配置说明.md) | 如何配置节点采集地址（订阅 URL） |
| [节点健康检查设置说明](./配置/节点健康检查设置说明.md) | 节点自动检查间隔、延迟阈值等 |
| [备份设置说明](./配置/备份设置说明.md) | 自动备份、Gitee/GitHub 备份配置 |
| [GitHub 配置说明](./配置/GitHub配置说明.md) | 备份到 GitHub 的详细配置 |
| [Gitee 配置说明](./配置/Gitee配置说明.md) | 备份到 Gitee 的详细配置 |

完整使用说明请查看项目根目录 **[README_zh.md](../README_zh.md)**。

---

## English

### Deployment & Troubleshooting

| Document | Description |
|----------|-------------|
| [VPS Deployment (No BT Panel)](./VPS部署教程-无宝塔.md) | One-click VPS deployment |
| [Installation Troubleshooting](./故障排查/安装问题排查指南.md) | Common installation issues and solutions |

### Feature Documentation

| Document | Description |
|----------|-------------|
| [List Functions Index](./功能/列表功能索引.md) | Index of list functions |
| [User List](./功能/用户列表管理功能说明.md) | User management |
| [Subscription List](./功能/订阅管理功能说明.md) | Subscription & device limit |
| [Order List](./功能/订单管理功能说明.md) | Order management |
| [Package Management](./功能/套餐管理功能说明.md) | Package create, edit, delete, pricing |
| [Invite Code Management](./功能/邀请码管理功能说明.md) | Invite code generation, management, rewards |
| [User Level Management](./功能/用户等级管理功能说明.md) | User level create, discount settings, auto upgrade |
| [Subscription Reset](./功能/订阅重置功能说明.md) | User reset subscription, admin reset, extend subscription |
| [Node Management](./功能/节点管理功能说明.md) | Node collection & import |
| [Manual Node Import](./功能/节点手动导入说明.md) | Link import, manual entry, Clash config import |
| [Custom Node Management](./功能/专线节点管理说明.md) | Custom node create, assign, unassign, delete, test |
| [Node Speed Test](./功能/节点测速说明.md) | Node speed test, batch test, test原理 |
| [Ticket Management](./功能/工单管理功能说明.md) | Ticket handling |
| [Device Management](./功能/设备管理功能说明.md) | Device list & limit |
| [Device Deletion](./功能/设备删除说明.md) | User delete device, admin delete device |
| [Login History](./功能/登录历史管理功能说明.md) | Login history |
| [Abnormal Users](./功能/异常用户管理功能说明.md) | Abnormal user handling |
| [Data Analysis](./功能/数据分析功能说明.md) | Statistics & regions |

### Backend & Configuration

#### Payment Configuration

| Document | Description |
|----------|-------------|
| [Alipay configuration](./配置/支付宝配置说明.md) | Alipay backend and panel config |
| [Yipay guide](./配置/易支付配置指南.md) | Yipay setup and usage |

#### Notification Configuration

| Document | Description |
|----------|-------------|
| [SMTP / Email](./配置/邮件服务器配置说明.md) | Configure mail server |
| [Telegram notifications](./配置/Telegram通知配置说明.md) | Telegram Bot notification setup |
| [Bark notifications](./配置/Bark通知配置说明.md) | Bark iOS push notification setup |
| [Customer notifications](./配置/客户通知设置说明.md) | Customer email notification settings |

#### System Settings

| Document | Description |
|----------|-------------|
| [Basic settings](./配置/基本设置说明.md) | Site info, logo, domain, GeoIP |
| [Registration settings](./配置/注册设置说明.md) | Registration flow, password requirements |
| [Security & limits](./配置/安全设置与限制说明.md) | Login fail limit, lock, IP whitelist, unlock |
| [Theme settings](./配置/主题设置说明.md) | Default theme, user customization |
| [Announcement management](./配置/公告管理说明.md) | Login announcement popup |

#### Node & Backup

| Document | Description |
|----------|-------------|
| [Node collection URLs](./配置/采集地址配置说明.md) | Configure node collection addresses |
| [Node health check](./配置/节点健康检查设置说明.md) | Auto check interval, latency threshold |
| [Backup settings](./配置/备份设置说明.md) | Auto backup, Gitee/GitHub backup |
| [GitHub configuration](./配置/GitHub配置说明.md) | Detailed GitHub backup setup |
| [Gitee configuration](./配置/Gitee配置说明.md) | Detailed Gitee backup setup |

For full documentation see **[README.md](../README.md)** in the project root.
