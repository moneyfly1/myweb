# 专线节点搭建完整流程指南

## 一、搭建前的准备工作

### 1. 在服务器上安装 XrayR

**重要**：在创建专线节点之前，您需要先在服务器上安装并配置 XrayR。

#### 步骤 1：SSH 连接到服务器

使用您添加的服务器信息（IP、端口、用户名、密码）连接到服务器。

#### 步骤 2：安装 XrayR

```bash
# 下载并运行安装脚本
bash <(curl -Ls https://raw.githubusercontent.com/XrayR-project/XrayR-release/master/install.sh)
```

#### 步骤 3：配置 XrayR

编辑配置文件（通常在 `/etc/XrayR/config.yml`）：

```yaml
Log:
  Level: info
  AccessPath: /var/log/XrayR/access.log
  ErrorPath: /var/log/XrayR/error.log

ApiConfig:
  ApiHost: "0.0.0.0:10086"  # API 监听地址和端口（重要！）
  ApiKey: "your-strong-api-key-here"  # API 密钥（重要！）
  NodeID: 1
  NodeType: V2ray
  Timeout: 30
```

**关键配置说明**：
- `ApiHost`: XrayR API 的监听地址和端口，例如 `0.0.0.0:10086`
- `ApiKey`: API 密钥，用于身份验证，建议使用至少 32 位的随机字符串

#### 步骤 4：启动 XrayR

```bash
# 启动服务
systemctl start XrayR

# 设置开机自启
systemctl enable XrayR

# 查看状态
systemctl status XrayR
```

### 2. 配置系统设置中的 XrayR API

#### 步骤 1：进入系统设置

1. 登录管理后台
2. 点击左侧菜单：**系统管理 > 系统设置**
3. 切换到 **专线节点设置** 标签页

#### 步骤 2：填写 XrayR API 地址

**格式**：`http://服务器IP:端口`

**示例**：
- 如果服务器 IP 是 `192.168.1.100`，XrayR 配置的端口是 `10086`
- 那么 API 地址就是：`http://192.168.1.100:10086`

**注意事项**：
- 如果服务器有防火墙，需要开放 API 端口（例如 10086）
- 如果使用域名，可以是：`http://api.yourdomain.com:10086`
- 建议使用 HTTPS（需要配置 SSL 证书）

#### 步骤 3：填写 XrayR API 密钥

**重要**：这里的密钥必须与 XrayR 配置文件中的 `ApiKey` 完全一致！

**示例**：
- 如果 XrayR 配置文件中 `ApiKey: "my-secret-key-12345"`
- 那么系统设置中也要填写：`my-secret-key-12345`

#### 步骤 4：测试连接（可选）

保存设置后，可以尝试创建一个测试节点来验证配置是否正确。

### 3. 配置 Cloudflare API（可选，用于域名和证书）

如果您需要使用域名和自动申请 SSL 证书，需要配置 Cloudflare API：

#### 步骤 1：获取 Cloudflare API Token

1. 登录 Cloudflare 控制台
2. 进入 **My Profile > API Tokens**
3. 点击 **Create Token**
4. 选择 **Edit zone DNS** 模板
5. 选择要管理的域名
6. 创建 Token 并复制

#### 步骤 2：填写 Cloudflare 配置

在系统设置的 **专线节点设置** 中：
- **Cloudflare API Token**: 粘贴刚才创建的 Token
- **Cloudflare 邮箱**: 您的 Cloudflare 注册邮箱（如果使用 API Key 而不是 Token）

#### 步骤 3：配置证书申请邮箱

- **证书申请邮箱**: 用于 Let's Encrypt 证书申请的邮箱地址

## 二、创建专线节点的完整流程

### 步骤 1：添加服务器

1. 进入 **节点管理 > 服务器管理**
2. 点击 **添加服务器**
3. 填写服务器信息：
   - **服务器名称**: 例如 "美国 VPS 1"
   - **服务器地址**: 服务器的 IP 地址或域名
   - **SSH端口**: 默认 22
   - **用户名**: SSH 登录用户名
   - **密码**: SSH 登录密码
4. 点击 **保存**
5. 点击 **测试连接** 验证服务器连接是否正常

### 步骤 2：配置 XrayR API（如果还没配置）

按照上面的说明，在系统设置中配置 XrayR API 地址和密钥。

### 步骤 3：创建专线节点

1. 进入 **节点管理 > 专线节点管理**
2. 点击 **创建专线节点**
3. 填写节点信息：

   **基本信息**：
   - **服务器**: 选择刚才添加的服务器
   - **节点名称**: 例如 "专线-美国-01"
   - **协议类型**: 选择 VMess、VLESS、Trojan 或 Shadowsocks
   - **域名**: （可选）如果使用域名，填写域名；留空则使用服务器 IP
   - **端口**: 点击 **随机生成** 或手动输入
   - **UUID**: （VMess/VLESS）点击 **随机生成** 或手动输入
   - **密码**: （Trojan/SS）点击 **随机生成** 或手动输入

   **高级设置**：
   - **传输协议**: TCP、WebSocket 或 gRPC
   - **安全设置**: TLS、Reality 或 None
   - **SNI**: （TLS/Reality）服务器名称指示
   - **流量限制**: 0 表示无限制，或输入字节数（例如 1073741824 表示 1GB）
   - **到期时间**: （可选）节点到期时间
   - **遵循用户到期**: 开启后，节点到期时间将跟随用户订阅到期时间

4. 点击 **保存**

### 步骤 4：等待自动搭建

创建节点后，系统会在后台自动执行以下操作：

1. **域名配置**（如果提供了域名）：
   - 在 Cloudflare 创建 DNS A 记录
   - 等待 DNS 传播

2. **证书申请**（如果提供了域名）：
   - 使用 acme.sh 申请 Let's Encrypt 证书
   - 上传证书到服务器

3. **节点创建**：
   - 通过 XrayR API 在服务器上创建节点配置
   - 启动 Xray-core 服务

4. **状态更新**：
   - 节点状态会从 `pending` 变为 `active`
   - 如果出错，状态会变为 `error`

### 步骤 5：分配节点给用户

1. 进入 **用户管理 > 用户列表**
2. 找到要分配节点的用户，点击 **详情**
3. 切换到 **专线节点分配** 标签页
4. 在 **分配新专线节点** 下拉框中选择节点
5. 点击 **分配**

## 三、常见问题

### Q1: XrayR API 地址应该填什么？

**A**: 填写 XrayR 服务运行的地址和端口，格式：`http://IP:端口`

**示例**：
- 服务器 IP: `192.168.1.100`
- XrayR 端口: `10086`
- API 地址: `http://192.168.1.100:10086`

### Q2: XrayR API 密钥在哪里设置？

**A**: 
1. 在服务器上的 XrayR 配置文件（`/etc/XrayR/config.yml`）中设置 `ApiKey`
2. 在管理后台的 **系统设置 > 专线节点设置** 中填写相同的密钥

### Q3: 如何测试 XrayR API 是否正常？

**A**: 可以使用 curl 命令测试：

```bash
curl -X GET "http://服务器IP:端口/api/v1/node" \
  -H "Authorization: Bearer your-api-key" \
  -H "X-API-Key: your-api-key"
```

如果返回节点列表，说明配置正确。

### Q4: 创建节点后一直显示 "pending" 状态？

**A**: 可能的原因：
1. XrayR API 地址或密钥配置错误
2. 服务器防火墙未开放 API 端口
3. XrayR 服务未启动
4. 查看服务器日志：`journalctl -u XrayR -f`

### Q5: 如何查看节点搭建日志？

**A**: 
1. 在专线节点列表中，查看节点的状态
2. 如果状态为 `error`，可以查看系统日志
3. 也可以 SSH 到服务器查看 XrayR 日志：`tail -f /var/log/XrayR/error.log`

### Q6: 节点创建成功，但用户无法连接？

**A**: 检查：
1. 节点是否已分配给用户
2. 节点的流量限制和到期时间
3. 服务器的防火墙是否开放了节点端口
4. Xray-core 服务是否正常运行

## 四、完整流程总结

```
1. 在服务器上安装 XrayR
   ↓
2. 配置 XrayR（设置 ApiHost 和 ApiKey）
   ↓
3. 启动 XrayR 服务
   ↓
4. 在管理后台添加服务器信息
   ↓
5. 在系统设置中配置 XrayR API 地址和密钥
   ↓
6. （可选）配置 Cloudflare API 和证书邮箱
   ↓
7. 创建专线节点（系统自动搭建）
   ↓
8. 分配节点给用户
   ↓
9. 用户在订阅中看到专线节点
```

## 五、快速开始检查清单

- [ ] 服务器已安装 XrayR
- [ ] XrayR 配置文件已设置 ApiHost 和 ApiKey
- [ ] XrayR 服务已启动并运行正常
- [ ] 服务器防火墙已开放 XrayR API 端口
- [ ] 管理后台已添加服务器信息
- [ ] 系统设置中已配置 XrayR API 地址和密钥
- [ ] （可选）已配置 Cloudflare API
- [ ] （可选）已配置证书申请邮箱
- [ ] 已创建测试节点并验证成功

完成以上步骤后，您就可以开始创建专线节点了！


