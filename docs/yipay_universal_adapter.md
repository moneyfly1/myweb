# 易支付通用适配器系统

## 概述

本系统设计了一个通用的易支付对接框架，能够自动识别和适配不同的易支付平台，无需为每个新平台修改代码。

## 支持的平台

系统已内置支持以下易支付平台（通过网关地址自动识别）：

- **fhymw.com** - FH易支付
- **yi-zhifu.cn** - 易支付官方
- **ezfp.cn** - EZ易支付
- **myzfw.com** - MY易支付
- **8-pay.cn** - 8-Pay
- **epay.hehanwang.com** - 易支付
- **wx8g.com** - 易支付

## 架构设计

### 1. 适配器接口 (`YipayPlatformAdapter`)

定义了统一的适配器接口，所有平台适配器必须实现：

```go
type YipayPlatformAdapter interface {
    GetPlatformName() string
    GetAPIURL(gatewayURL string) string
    GetSubmitURL(gatewayURL string) string
    NormalizeResponse(resp map[string]interface{}) *YipayResponse
    GetResponseFields() YipayResponseFields
    SupportsSignatureType(signType string) bool
}
```

### 2. 平台检测器 (`YipayPlatformDetector`)

自动检测平台类型：

1. **优先使用配置的平台名称**：如果 `config_json` 中指定了 `platform_name`，直接使用
2. **通过网关地址检测**：分析 `gateway_url` 或 `api_url` 中的域名，匹配已知平台
3. **默认使用标准适配器**：如果无法识别，使用标准适配器（兼容大多数平台）

### 3. 标准适配器 (`StandardYipayAdapter`)

实现了标准的易支付API规范：

- API路径：`/mapi.php`
- 提交路径：`/submit.php`
- 响应字段：`code`, `msg`, `trade_no`, `payurl`, `qrcode`, `urlscheme`
- 签名类型：支持 `MD5`、`RSA`、`MD5+RSA`

## 使用方法

### 后端配置

在 `payment_configs` 表的 `config_json` 字段中配置：

```json
{
  "gateway_url": "https://fhymw.com",
  "api_url": "https://fhymw.com/mapi.php",
  "sign_type": "MD5",
  "platform_public_key": "...",
  "merchant_private_key": "..."
}
```

**注意**：
- `gateway_url`：易支付网关地址（系统会自动拼接 `/mapi.php`）
- `api_url`：完整的API地址（如果已配置，优先使用）
- `platform_name`：可选，手动指定平台类型（如 `"standard"`）

### 前端配置

在管理员后台配置易支付时：

1. **选择支付类型**：`yipay`、`yipay_alipay`、`yipay_wxpay` 等
2. **填写网关地址**：例如 `https://fhymw.com` 或 `https://pay.yi-zhifu.cn`
3. **系统自动识别**：系统会根据网关地址自动识别平台类型
4. **配置签名类型**：选择 `MD5`、`RSA` 或 `MD5+RSA`

## 扩展新平台

如果需要支持新的易支付平台，有两种方式：

### 方式1：使用标准适配器（推荐）

如果新平台的API规范与标准易支付一致，无需任何修改，系统会自动使用标准适配器。

### 方式2：创建自定义适配器

如果新平台有特殊要求，可以创建自定义适配器：

```go
type CustomYipayAdapter struct{}

func (a *CustomYipayAdapter) GetPlatformName() string {
    return "custom"
}

func (a *CustomYipayAdapter) GetAPIURL(gatewayURL string) string {
    // 自定义API路径
    return gatewayURL + "/custom_api.php"
}

func (a *CustomYipayAdapter) NormalizeResponse(resp map[string]interface{}) *YipayResponse {
    // 自定义响应解析逻辑
    // ...
}

// 注册适配器
detector.RegisterAdapter("custom", &CustomYipayAdapter{})
```

然后在 `config_json` 中指定：

```json
{
  "platform_name": "custom",
  "gateway_url": "https://custom-platform.com"
}
```

## 技术细节

### 响应字段映射

不同平台的响应字段可能略有不同，适配器负责统一映射：

- `code` / `status` → `Code`
- `msg` / `message` → `Msg`
- `trade_no` / `o_id` → `TradeNo`
- `payurl` / `pay_url` → `PayURL`
- `qrcode` / `qr_code` → `QRCode`
- `urlscheme` / `url_scheme` → `URLScheme`

### 签名算法

系统支持三种签名类型：

1. **MD5**：`sign = md5(参数串 + KEY)`（小写）
2. **RSA**：使用商户私钥对参数串进行RSA签名
3. **MD5+RSA**：同时使用MD5和RSA签名

### 设备类型检测

系统会根据 `User-Agent` 自动检测设备类型：

- `wechat`：微信内浏览器 + 微信支付
- `mobile`：手机浏览器
- `alipay`：支付宝客户端
- `qq`：QQ内浏览器
- `pc`：电脑浏览器（默认）

## 优势

1. **零代码修改**：接入新平台无需修改代码，只需配置网关地址
2. **自动识别**：系统自动识别平台类型，选择正确的适配器
3. **易于扩展**：通过适配器模式，可以轻松添加新平台支持
4. **向后兼容**：完全兼容现有的易支付配置
5. **统一接口**：所有平台使用相同的调用接口，简化业务逻辑

## 日志

系统会记录详细的平台识别和API调用日志：

```
易支付初始化: platform=standard, api_url=https://fhymw.com/mapi.php, pid=1100, sign_type=MD5
易支付设备类型检测: order_no=ORD20260131001, device=wechat, paymentType=wxpay
易支付返回结果 [平台=standard]: code=1, msg=成功, payurl=https://...
```

## 注意事项

1. **网关地址格式**：建议使用完整URL（包含 `https://`），系统会自动处理
2. **API路径**：如果平台使用非标准路径，可以在 `config_json` 中直接配置 `api_url`
3. **签名类型**：确保选择的签名类型与平台支持的类型一致
4. **回调地址**：系统会自动生成回调地址，也可以手动配置

## 测试

系统已通过以下平台的测试：

- ✅ fhymw.com (MD5+RSA)
- ✅ 标准易支付平台 (MD5)

其他平台可以通过配置网关地址自动适配。
