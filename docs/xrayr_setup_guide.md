# XrayR API 配置指南

## 一、XrayR 是什么？

**XrayR** 是一个基于 Xray-core 的后端框架，用于管理代理节点。它提供了：
- 统一的节点管理接口
- 多面板对接支持
- 用户流量统计
- 节点状态监控

## 二、XrayR API 的作用

在我们的专线节点系统中，XrayR API 用于：

1. **自动创建节点**：当您在后台创建专线节点时，系统会通过 XrayR API 在服务器上自动创建节点配置
2. **节点管理**：更新节点配置、删除节点等操作
3. **流量统计**：获取节点的流量使用情况
4. **状态监控**：检查节点运行状态

## 三、XrayR 与服务器的关系

### 架构关系

```
┌─────────────────┐
│   管理后台       │
│  (您的系统)      │
└────────┬────────┘
         │ HTTP API 调用
         │ (XrayR API)
         ▼
┌─────────────────┐
│   XrayR 服务     │
│  (运行在服务器上) │
└────────┬────────┘
         │ 管理
         ▼
┌─────────────────┐
│  Xray-core      │
│  (实际代理服务)   │
└─────────────────┘
```

### 关系说明

1. **服务器（VPS）**：物理或虚拟服务器，运行 XrayR 服务
2. **XrayR**：安装在服务器上的管理程序，提供 API 接口
3. **管理后台**：您的系统，通过 API 调用 XrayR 来管理节点

### 工作流程

1. 您在后台添加服务器信息（IP、SSH端口、用户名、密码）
2. 您在后台创建专线节点时：
   - 系统通过 XrayR API 调用服务器上的 XrayR 服务
   - XrayR 在服务器上创建节点配置
   - XrayR 启动 Xray-core 服务
   - 节点开始工作

## 四、如何获取 XrayR API 地址和密钥

### 步骤 1：安装 XrayR

在您的服务器上安装 XrayR：

```bash
# 下载安装脚本
bash <(curl -Ls https://raw.githubusercontent.com/XrayR-project/XrayR/master/install.sh)

# 或者使用一键安装脚本
bash <(curl -Ls https://raw.githubusercontent.com/XrayR-project/XrayR-release/master/install.sh)
```

### 步骤 2：配置 XrayR

编辑 XrayR 配置文件（通常在 `/etc/XrayR/config.yml`）：

```yaml
# API 配置
ApiConfig:
  ApiHost: "0.0.0.0:10086"  # API 监听地址和端口
  ApiKey: "your-api-key-here"  # API 密钥（自定义设置）
  NodeID: 1
  NodeType: V2ray
  Timeout: 30
  EnableVless: false
  EnableXTLS: false
  SpeedLimit: 0
  DeviceLimit: 0
  RuleListPath: /etc/XrayR/rulelist
```

### 步骤 3：获取 API 地址

**API 地址格式**：`http://服务器IP:端口`

例如：
- 如果 XrayR 运行在服务器 `192.168.1.100` 上，端口是 `10086`
- 那么 API 地址就是：`http://192.168.1.100:10086`

**注意事项**：
- 如果服务器有防火墙，需要开放 API 端口
- 如果使用域名，可以是：`http://api.yourdomain.com:10086`
- 建议使用 HTTPS（需要配置 SSL 证书）

### 步骤 4：设置 API 密钥

在 XrayR 配置文件中设置 `ApiKey`，然后在管理后台填写相同的密钥。

**安全建议**：
- 使用强密码（至少 32 位随机字符串）
- 不要将密钥泄露给他人
- 定期更换密钥

### 步骤 5：启动 XrayR

```bash
# 启动服务
systemctl start XrayR

# 设置开机自启
systemctl enable XrayR

# 查看状态
systemctl status XrayR

# 查看日志
journalctl -u XrayR -f
```

### 步骤 6：测试 API

可以使用 curl 测试 API 是否正常工作：

```bash
curl -X GET "http://服务器IP:端口/api/v1/node" \
  -H "Authorization: Bearer your-api-key-here" \
  -H "X-API-Key: your-api-key-here"
```

## 五、在管理后台配置

1. 进入 **系统设置 > 专线节点设置**
2. 填写 **XrayR API地址**：`http://服务器IP:端口`
3. 填写 **XrayR API密钥**：与 XrayR 配置文件中的 `ApiKey` 一致
4. 点击 **保存专线节点设置**

## 六、常见问题

### Q1: XrayR API 地址应该填什么？

**A**: 填写 XrayR 服务运行的地址和端口，格式：`http://IP:端口` 或 `https://域名:端口`

### Q2: 一台服务器可以运行多个 XrayR 吗？

**A**: 可以，但需要配置不同的端口。每个 XrayR 实例需要独立的 API 地址和密钥。

### Q3: XrayR 必须和节点在同一台服务器吗？

**A**: 不一定。XrayR 可以管理远程的 Xray-core 节点，但通常建议在同一台服务器上。

### Q4: 如何查看 XrayR 的 API 端口？

**A**: 查看 XrayR 配置文件中的 `ApiHost` 字段，或查看运行日志。

### Q5: API 密钥在哪里设置？

**A**: 在 XrayR 配置文件（`/etc/XrayR/config.yml`）中的 `ApiKey` 字段设置。

### Q6: 如何确保 API 安全？

**A**: 
- 使用强密码作为 API 密钥
- 配置防火墙，只允许管理后台 IP 访问 API 端口
- 使用 HTTPS（需要配置 SSL 证书）
- 定期更换 API 密钥

## 七、完整配置示例

### XrayR 配置文件示例

```yaml
Log:
  Level: info
  AccessPath: /var/log/XrayR/access.log
  ErrorPath: /var/log/XrayR/error.log

ApiConfig:
  ApiHost: "0.0.0.0:10086"
  ApiKey: "your-strong-api-key-here-32-chars-min"
  NodeID: 1
  NodeType: V2ray
  Timeout: 30

NodeConfig:
  NodeType: V2ray
  EnableVless: false
  EnableXTLS: false
  SpeedLimit: 0
  DeviceLimit: 0
```

### 管理后台配置示例

```
XrayR API地址: http://192.168.1.100:10086
XrayR API密钥: your-strong-api-key-here-32-chars-min
```

## 八、验证配置

配置完成后，在管理后台创建专线节点时：
1. 系统会调用 XrayR API 创建节点
2. 如果配置正确，节点会成功创建
3. 如果配置错误，会显示错误信息

建议先创建一个测试节点，确认配置正确后再批量创建。

