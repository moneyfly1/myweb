# XrayR API 快速配置指南

## 一、XrayR API 地址和密钥的作用

**XrayR API** 是管理后台与服务器上的 XrayR 服务通信的桥梁。当您在后台创建专线节点时，系统会通过这个 API 在服务器上自动创建节点配置。

## 二、如何获取 XrayR API 地址和密钥

### 方法一：在服务器上配置（推荐）

#### 步骤 1：SSH 连接到您的服务器

使用您添加的服务器信息（IP、端口、用户名、密码）连接。

#### 步骤 2：安装 XrayR（如果还没安装）

```bash
bash <(curl -Ls https://raw.githubusercontent.com/XrayR-project/XrayR-release/master/install.sh)
```

#### 步骤 3：编辑 XrayR 配置文件

```bash
nano /etc/XrayR/config.yml
```

找到 `ApiConfig` 部分，修改为：

```yaml
ApiConfig:
  ApiHost: "0.0.0.0:10086"  # 修改为您想要的端口，例如 10086
  ApiKey: "your-strong-api-key-here"  # 修改为您自己的密钥（至少32位）
  NodeID: 1
  NodeType: V2ray
  Timeout: 30
```

**重要**：
- `ApiHost`: 格式为 `0.0.0.0:端口`，端口可以是任意未使用的端口（例如 10086）
- `ApiKey`: 建议使用至少 32 位的随机字符串，例如：`a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`

#### 步骤 4：启动 XrayR

```bash
systemctl start XrayR
systemctl enable XrayR
systemctl status XrayR
```

#### 步骤 5：开放防火墙端口（如果服务器有防火墙）

```bash
# Ubuntu/Debian
ufw allow 10086/tcp

# CentOS/RHEL
firewall-cmd --permanent --add-port=10086/tcp
firewall-cmd --reload
```

### 方法二：在管理后台配置

#### 步骤 1：进入系统设置

1. 登录管理后台
2. 点击左侧菜单：**系统管理 > 系统设置**
3. 切换到 **专线节点设置** 标签页

#### 步骤 2：填写 XrayR API 地址

**格式**：`http://服务器IP:端口`

**示例**：
- 服务器 IP: `192.168.1.100`
- XrayR 端口: `10086`（在步骤 3 中设置的端口）
- API 地址: `http://192.168.1.100:10086`

**注意事项**：
- 如果服务器有防火墙，需要开放这个端口
- 如果使用域名，可以是：`http://api.yourdomain.com:10086`

#### 步骤 3：填写 XrayR API 密钥

**重要**：这里的密钥必须与服务器上 XrayR 配置文件中的 `ApiKey` **完全一致**！

**示例**：
- 如果 XrayR 配置文件中 `ApiKey: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"`
- 那么系统设置中也要填写：`a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6`

#### 步骤 4：保存设置

点击 **保存专线节点设置** 按钮。

## 三、测试配置是否正确

### 方法 1：使用 curl 命令（在服务器上）

```bash
curl -X GET "http://localhost:10086/api/v1/node" \
  -H "Authorization: Bearer your-api-key" \
  -H "X-API-Key: your-api-key"
```

如果返回节点列表（可能是空的），说明配置正确。

### 方法 2：在管理后台创建测试节点

1. 进入 **节点管理 > 专线节点管理**
2. 点击 **创建专线节点**
3. 填写基本信息并保存
4. 查看节点状态：
   - 如果状态变为 `active`，说明配置正确
   - 如果状态为 `error`，查看错误信息

## 四、常见问题

### Q1: 如何知道 XrayR 是否正常运行？

**A**: 在服务器上运行：
```bash
systemctl status XrayR
```

如果显示 `active (running)`，说明运行正常。

### Q2: 如何查看 XrayR 日志？

**A**: 
```bash
# 查看错误日志
tail -f /var/log/XrayR/error.log

# 查看访问日志
tail -f /var/log/XrayR/access.log
```

### Q3: API 地址应该填内网 IP 还是公网 IP？

**A**: 
- 如果管理后台和服务器在同一内网，可以填内网 IP
- 如果管理后台在公网，必须填公网 IP
- 建议使用公网 IP，确保稳定性

### Q4: 如何修改 XrayR 的 API 端口？

**A**: 
1. 编辑配置文件：`nano /etc/XrayR/config.yml`
2. 修改 `ApiHost` 中的端口号
3. 重启服务：`systemctl restart XrayR`
4. 更新管理后台中的 API 地址

### Q5: 如何生成安全的 API 密钥？

**A**: 可以使用以下命令生成：

```bash
# 生成 32 位随机字符串
openssl rand -hex 16

# 或使用 Python
python3 -c "import secrets; print(secrets.token_hex(16))"
```

## 五、完整配置示例

### 服务器端（/etc/XrayR/config.yml）

```yaml
ApiConfig:
  ApiHost: "0.0.0.0:10086"
  ApiKey: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6"
  NodeID: 1
  NodeType: V2ray
  Timeout: 30
```

### 管理后台（系统设置 > 专线节点设置）

```
XrayR API地址: http://192.168.1.100:10086
XrayR API密钥: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6
```

## 六、搭建专线节点前的检查清单

在创建专线节点之前，请确保：

- [ ] 服务器已安装 XrayR
- [ ] XrayR 配置文件已设置 `ApiHost` 和 `ApiKey`
- [ ] XrayR 服务已启动（`systemctl status XrayR` 显示 active）
- [ ] 服务器防火墙已开放 XrayR API 端口
- [ ] 管理后台已添加服务器信息（节点管理 > 服务器管理）
- [ ] 系统设置中已配置 XrayR API 地址（格式：`http://IP:端口`）
- [ ] 系统设置中已配置 XrayR API 密钥（与服务器配置一致）

完成以上步骤后，您就可以创建专线节点了！

