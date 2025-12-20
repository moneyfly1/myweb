# 机场节点搭建方式分析与重构方案

## 一、一般机场的节点搭建方式

### 1. 主流代理软件
- **Xray-core**: 支持 VMess、VLESS、Trojan、Shadowsocks、Reality 等
- **V2Ray-core**: 支持 VMess、VLESS 等传统协议
- **Sing-box**: 支持新协议如 Hysteria、Hysteria2、TUIC、Naive 等
- **Shadowsocks**: 轻量级，支持多种加密方式

### 2. 常见安装脚本
- **Xray**: `bash -c "$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)" @ install`
- **V2Ray**: `bash <(curl -L https://raw.githubusercontent.com/v2fly/fhs-install-v2ray/master/install-release.sh)`
- **Sing-box**: 各种第三方脚本（如当前使用的 yonggekkk/sing-box-yg）
- **Shadowsocks**: `wget -N --no-check-certificate https://raw.githubusercontent.com/teddysun/shadowsocks_install/master/shadowsocks.sh && bash shadowsocks.sh`

### 3. 协议选择
- **VMess**: Xray/V2Ray 原生协议，功能强大
- **VLESS**: 更轻量，性能更好
- **Trojan**: 伪装成 HTTPS，抗检测
- **Shadowsocks**: 简单高效
- **Reality**: 基于 VLESS，无需域名
- **Hysteria/Hysteria2**: 基于 QUIC，速度快
- **TUIC**: 基于 QUIC，低延迟

## 二、当前代码分析

### 当前实现
- 使用固定的 sing-box 脚本：`yonggekkk/sing-box-yg/main/sb.sh`
- 只支持 sing-box，协议选择有限
- 通过交互式菜单操作（echo -e '9\n1\n'）
- 无法选择协议类型和搭建方式

### 问题
1. ❌ 只支持一种搭建方式（sing-box）
2. ❌ 无法选择协议类型
3. ❌ 无法自定义端口、密码等参数
4. ❌ 依赖第三方脚本的菜单结构

## 三、重构方案

### 方案概述
设计一个灵活的节点搭建系统，支持：
1. **多种搭建方式**：Xray、V2Ray、Sing-box、Shadowsocks
2. **多种协议**：VMess、VLESS、Trojan、SS、Reality、Hysteria 等
3. **参数配置**：端口、密码、UUID、加密方式等
4. **自动解析**：安装后自动提取节点配置并添加到专线列表

### 技术可行性
✅ **完全可行**

理由：
1. 所有主流代理软件都支持命令行安装和配置
2. 可以通过 SSH 执行安装脚本
3. 配置文件可以自动生成和解析
4. 节点链接格式标准化，易于解析

### 实现步骤

#### 1. 扩展数据模型
```go
type CustomNode struct {
    // ... 现有字段
    InstallMethod string  // 安装方式: xray, v2ray, sing-box, shadowsocks
    Protocol      string  // 协议类型: vmess, vless, trojan, ss, reality, hysteria等
    NodePort      int     // 节点端口（代理端口，不是SSH端口）
    NodePassword  string  // 节点密码/UUID
    // ... 其他协议相关参数
}
```

#### 2. 创建安装器接口
```go
type NodeInstaller interface {
    Install(server, port, username, password string, config *InstallConfig) error
    GetConfig() (string, error)
    Uninstall() error
}
```

#### 3. 实现不同安装器
- `XrayInstaller`: Xray 安装和配置
- `V2RayInstaller`: V2Ray 安装和配置
- `SingBoxInstaller`: Sing-box 安装和配置
- `ShadowsocksInstaller`: Shadowsocks 安装和配置

#### 4. 前端界面
- 添加"安装方式"选择（Xray/V2Ray/Sing-box/SS）
- 添加"协议类型"选择（根据安装方式动态显示）
- 添加"节点端口"输入
- 添加"节点密码/UUID"输入
- 其他协议相关参数

## 四、推荐方案

### 方案A：使用主流安装脚本（推荐）
**优点**：
- 稳定可靠，社区维护
- 支持多种协议
- 配置自动生成

**实现**：
- Xray: 使用官方安装脚本 + 配置文件生成
- V2Ray: 使用官方安装脚本 + 配置文件生成
- Sing-box: 继续使用当前脚本或切换到其他脚本
- Shadowsocks: 使用 teddysun 脚本

### 方案B：自研安装器（更灵活）
**优点**：
- 完全可控
- 可以自定义所有参数
- 不依赖第三方脚本

**缺点**：
- 开发工作量大
- 需要维护多个协议的配置生成逻辑

## 五、建议

**推荐使用方案A**，原因：
1. 利用成熟的社区脚本，稳定性好
2. 开发工作量适中
3. 支持协议全面
4. 易于维护和更新

### 具体实现建议
1. **前端**：添加安装方式和协议选择界面
2. **后端**：创建安装器工厂，根据选择调用对应安装器
3. **配置生成**：根据协议类型自动生成配置文件
4. **节点提取**：安装完成后自动解析配置，提取节点信息
5. **添加到列表**：将生成的节点自动添加到专线节点列表

## 六、总结

✅ **完全可行**，建议采用方案A（使用主流安装脚本）进行重构。

重构后的系统将：
- 支持多种搭建方式（Xray、V2Ray、Sing-box、SS）
- 支持多种协议（VMess、VLESS、Trojan、Reality、Hysteria 等）
- 用户只需提供服务器信息，系统自动搭建
- 自动解析并添加到专线列表

