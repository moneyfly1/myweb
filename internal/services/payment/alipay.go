package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"cboard-go/internal/models"

	"github.com/smartwalle/alipay/v3"
)

// AlipayService 支付宝支付服务
type AlipayService struct {
	client       *alipay.Client
	notifyURL    string
	returnURL    string
	isProduction bool
}

// NewAlipayService 创建支付宝服务
// 根据支付宝官方文档：https://opendocs.alipay.com/common/02kkv7
func NewAlipayService(paymentConfig *models.PaymentConfig) (*AlipayService, error) {
	// 1. 获取并验证 AppID
	appID := ""
	if paymentConfig.AppID.Valid {
		appID = strings.TrimSpace(paymentConfig.AppID.String)
	}
	if appID == "" {
		return nil, fmt.Errorf("支付宝 AppID 未配置，请在支付配置中设置 AppID")
	}

	// 2. 获取并验证应用私钥
	// 注意：应用私钥是通过"支付宝开发平台开发助手"创建的私钥文件
	// 参考：https://opendocs.alipay.com/common/02kkv7
	privateKey := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		privateKey = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝应用私钥未配置，请使用支付宝开发平台开发助手生成私钥并配置")
	}

	// 3. 自动修复私钥格式（支持多种输入格式）
	// SDK支持PKCS1和PKCS8格式的RSA2私钥，但私钥必须是完整的PEM格式
	privateKey = normalizePrivateKey(privateKey)
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝应用私钥格式错误：无法识别私钥格式。请确保私钥是完整的PEM格式（包含BEGIN和END标记）")
	}

	// 4. 判断是否为生产环境并配置网关地址
	// 根据支付宝文档：isProduction=false 为沙箱环境，isProduction=true 为生产环境
	// 参考：https://github.com/smartwalle/alipay
	isProduction := false
	var opts []alipay.OptionFunc

	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			// 判断生产/沙箱环境
			if prod, ok := configData["is_production"].(bool); ok {
				isProduction = prod
			} else if gatewayURL, ok := configData["gateway_url"].(string); ok && gatewayURL != "" {
				// 根据网关地址判断：如果包含 alipaydev.com 则是沙箱环境
				isProduction = !strings.Contains(strings.ToLower(gatewayURL), "alipaydev.com")
			}

			// 配置沙箱网关地址（如果需要使用老地址）
			// SDK 默认使用新地址：https://openapi-sandbox.dl.alipaydev.com/gateway.do
			// 老地址：https://openapi.alipaydev.com/gateway.do
			if !isProduction {
				if useOldGateway, ok := configData["use_old_sandbox_gateway"].(bool); ok && useOldGateway {
					opts = append(opts, alipay.WithPastSandboxGateway())
					fmt.Printf("使用支付宝沙箱老网关地址\n")
				} else {
					opts = append(opts, alipay.WithNewSandboxGateway())
					fmt.Printf("使用支付宝沙箱新网关地址（默认）\n")
				}
			}
		}
	}

	// 5. 创建支付宝客户端
	// 参考：https://github.com/smartwalle/alipay
	// SDK 会自动尝试 PKCS1 和 PKCS8 两种格式的私钥
	client, err := alipay.New(appID, privateKey, isProduction, opts...)
	if err != nil {
		// 提供更详细的错误信息，帮助用户排查问题
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v。请检查：1) AppID是否正确 2) 应用私钥是否为完整的PEM格式（PKCS1或PKCS8）3) 私钥是否与AppID匹配 4) 私钥长度是否为2048位（推荐）", err)
	}

	fmt.Printf("支付宝客户端初始化成功: AppID=%s, 环境=%s\n", appID, map[bool]string{true: "生产环境", false: "沙箱环境"}[isProduction])

	// 6. 加载支付宝公钥（普通公钥模式）
	// 注意：此处使用的是支付宝公钥，不是应用公钥
	// 参考：https://opendocs.alipay.com/common/057aqe
	// 公钥主要用于验证回调签名，创建支付时不是必须的
	// 但如果配置了公钥，应该确保格式正确
	if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
		publicKey := normalizePublicKey(paymentConfig.AlipayPublicKey.String)
		if publicKey != "" {
			if err := client.LoadAliPayPublicKey(publicKey); err != nil {
				// 公钥加载失败不影响创建支付，但会影响回调验证
				// 这里只记录警告，不返回错误
				fmt.Printf("警告：加载支付宝公钥失败（不影响创建支付，但会影响回调验证）: %v。请检查支付宝公钥格式是否正确\n", err)
			} else {
				fmt.Printf("支付宝公钥加载成功\n")
			}
		} else {
			fmt.Printf("警告：支付宝公钥格式无法识别，回调验证可能失败\n")
		}
	} else {
		fmt.Printf("提示：未配置支付宝公钥，回调验证可能失败。建议配置支付宝公钥以确保回调安全\n")
	}

	service := &AlipayService{
		client:       client,
		isProduction: isProduction,
	}

	// 设置回调地址
	if paymentConfig.NotifyURL.Valid && paymentConfig.NotifyURL.String != "" {
		service.notifyURL = strings.TrimSpace(paymentConfig.NotifyURL.String)
		fmt.Printf("支付宝回调地址已配置: %s\n", service.notifyURL)
	} else {
		// 如果未配置回调地址，返回错误
		fmt.Printf("错误：支付宝回调地址未配置\n")
		service.notifyURL = ""
	}
	if paymentConfig.ReturnURL.Valid && paymentConfig.ReturnURL.String != "" {
		service.returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
		fmt.Printf("支付宝返回地址已配置: %s\n", service.returnURL)
	}

	return service, nil
}

// normalizePrivateKey 规范化私钥格式
// 支持多种输入格式，自动转换为标准的PEM格式
func normalizePrivateKey(privateKey string) string {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return ""
	}

	// 如果已经包含 BEGIN 标记，说明已经是PEM格式，直接返回
	if strings.Contains(privateKey, "BEGIN") {
		// 确保有正确的换行符
		privateKey = strings.ReplaceAll(privateKey, "\r\n", "\n")
		privateKey = strings.ReplaceAll(privateKey, "\r", "\n")
		return privateKey
	}

	// 如果没有 BEGIN 标记，尝试自动添加
	// 移除所有空白字符以便识别
	cleanKey := strings.ReplaceAll(privateKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	// 检查是否是PKCS1格式（RSA PRIVATE KEY）
	// PKCS1格式通常以 MII 或 MIIC 开头
	if strings.HasPrefix(cleanKey, "MII") || strings.HasPrefix(cleanKey, "MIIC") {
		// 可能是PKCS1格式，添加PKCS1标记
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN RSA PRIVATE KEY-----") {
			privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END RSA PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END RSA PRIVATE KEY-----"
		}
		// 确保每64个字符换行（PEM格式标准）
		privateKey = formatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	// 检查是否是PKCS8格式（PRIVATE KEY）
	// PKCS8格式通常以 MIIE 或 MIIEv 开头
	if strings.HasPrefix(cleanKey, "MIIE") || strings.HasPrefix(cleanKey, "MIIEv") {
		// 可能是PKCS8格式，添加PKCS8标记
		privateKey = cleanKey
		if !strings.HasPrefix(privateKey, "-----BEGIN PRIVATE KEY-----") {
			privateKey = "-----BEGIN PRIVATE KEY-----\n" + privateKey
		}
		if !strings.HasSuffix(strings.TrimSpace(privateKey), "-----END PRIVATE KEY-----") {
			privateKey = privateKey + "\n-----END PRIVATE KEY-----"
		}
		// 确保每64个字符换行（PEM格式标准）
		privateKey = formatPEMKey(privateKey, "PRIVATE KEY")
		return privateKey
	}

	// 如果无法识别格式，尝试作为PKCS1格式处理（最常见）
	// 因为大多数支付宝私钥都是PKCS1格式
	if len(cleanKey) > 100 {
		privateKey = cleanKey
		privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey + "\n-----END RSA PRIVATE KEY-----"
		privateKey = formatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	// 如果无法识别格式，尝试作为PKCS1格式处理（最常见）
	// 因为大多数支付宝私钥都是PKCS1格式
	if len(cleanKey) > 100 {
		privateKey = cleanKey
		privateKey = "-----BEGIN RSA PRIVATE KEY-----\n" + privateKey + "\n-----END RSA PRIVATE KEY-----"
		privateKey = formatPEMKey(privateKey, "RSA PRIVATE KEY")
		return privateKey
	}

	// 如果无法识别格式，返回空字符串
	return ""
}

// normalizePublicKey 规范化公钥格式
// 支持多种输入格式，自动转换为标准的PEM格式
func normalizePublicKey(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return ""
	}

	// 如果已经包含 BEGIN 标记，说明已经是PEM格式
	if strings.Contains(publicKey, "BEGIN") {
		// 确保有正确的换行符
		publicKey = strings.ReplaceAll(publicKey, "\r\n", "\n")
		publicKey = strings.ReplaceAll(publicKey, "\r", "\n")
		return publicKey
	}

	// 如果没有 BEGIN 标记，尝试自动添加
	// 移除所有空白字符以便识别
	cleanKey := strings.ReplaceAll(publicKey, "\n", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\r", "")
	cleanKey = strings.ReplaceAll(cleanKey, " ", "")
	cleanKey = strings.ReplaceAll(cleanKey, "\t", "")

	// 公钥通常以 MIGf 或 MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A 开头
	if strings.HasPrefix(cleanKey, "MIGf") || strings.HasPrefix(cleanKey, "MIIBIjAN") || strings.HasPrefix(cleanKey, "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A") {
		// 添加 PUBLIC KEY 标记
		publicKey = cleanKey
		if !strings.HasPrefix(publicKey, "-----BEGIN PUBLIC KEY-----") {
			publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey
		}
		if !strings.HasSuffix(strings.TrimSpace(publicKey), "-----END PUBLIC KEY-----") {
			publicKey = publicKey + "\n-----END PUBLIC KEY-----"
		}
		// 格式化公钥
		return formatPEMPublicKey(publicKey)
	}

	// 如果无法识别格式，尝试作为标准公钥处理
	if len(cleanKey) > 50 {
		publicKey = cleanKey
		publicKey = "-----BEGIN PUBLIC KEY-----\n" + publicKey + "\n-----END PUBLIC KEY-----"
		return formatPEMPublicKey(publicKey)
	}

	// 如果无法识别格式，返回空字符串
	return ""
}

// formatPEMPublicKey 格式化PEM公钥，确保每64个字符换行
func formatPEMPublicKey(key string) string {
	beginMarker := "-----BEGIN PUBLIC KEY-----"
	endMarker := "-----END PUBLIC KEY-----"

	// 移除已有的标记
	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	// 移除所有空白字符（包括换行符和空格）
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	// 每64个字符换行
	var formatted strings.Builder
	formatted.WriteString(beginMarker)
	formatted.WriteString("\n")
	for i := 0; i < len(key); i += 64 {
		end := i + 64
		if end > len(key) {
			end = len(key)
		}
		formatted.WriteString(key[i:end])
		if end < len(key) {
			formatted.WriteString("\n")
		}
	}
	formatted.WriteString("\n")
	formatted.WriteString(endMarker)

	return formatted.String()
}

// formatPEMKey 格式化PEM密钥，确保每64个字符换行
func formatPEMKey(key, keyType string) string {
	// 提取BEGIN和END标记之间的内容
	beginMarker := fmt.Sprintf("-----BEGIN %s-----", keyType)
	endMarker := fmt.Sprintf("-----END %s-----", keyType)

	// 移除已有的标记
	key = strings.TrimPrefix(key, beginMarker)
	key = strings.TrimSuffix(key, endMarker)
	key = strings.TrimSpace(key)

	// 移除所有空白字符（包括换行符和空格）
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, " ", "")
	key = strings.ReplaceAll(key, "\t", "")

	// 每64个字符换行
	var formatted strings.Builder
	formatted.WriteString(beginMarker)
	formatted.WriteString("\n")
	for i := 0; i < len(key); i += 64 {
		end := i + 64
		if end > len(key) {
			end = len(key)
		}
		formatted.WriteString(key[i:end])
		if end < len(key) {
			formatted.WriteString("\n")
		}
	}
	formatted.WriteString("\n")
	formatted.WriteString(endMarker)

	return formatted.String()
}

// CreatePayment 创建支付（返回二维码URL）
// 根据支付宝官方文档：https://opendocs.alipay.com/common/02kkv7
func (s *AlipayService) CreatePayment(order *models.Order, amount float64) (string, error) {
	// 1. 验证必要参数
	if order == nil {
		return "", fmt.Errorf("订单信息不能为空")
	}
	if order.OrderNo == "" {
		return "", fmt.Errorf("订单号不能为空")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额必须大于0，当前金额: %.2f", amount)
	}
	if s.notifyURL == "" {
		return "", fmt.Errorf("异步回调地址未配置，请在支付配置中设置 NotifyURL")
	}

	// 2. 使用 TradePreCreate 接口生成二维码（推荐用于PC端扫码支付）
	// 参考：https://opendocs.alipay.com/apis/api_1/alipay.trade.precreate
	// 重要：TradePreCreate（当面付-生成二维码）接口不需要 ProductCode 字段
	// FAST_INSTANT_TRADE_PAY 是电脑网站支付的产品码，不适用于当面付接口
	// 如果设置了错误的产品码，会导致权限不足（40006）错误
	param := alipay.TradePreCreate{}
	param.NotifyURL = s.notifyURL
	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}
	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	// 重要：TradePreCreate（当面付-生成二维码）接口不需要 ProductCode 字段
	// 如果设置了 ProductCode，应该设置为空字符串，不要使用 FAST_INSTANT_TRADE_PAY
	// FAST_INSTANT_TRADE_PAY 是电脑网站支付的产品码，会导致权限不足错误
	param.ProductCode = "" // 明确设置为空，避免使用默认值

	fmt.Printf("支付宝TradePreCreate请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s\n",
		param.OutTradeNo, param.TotalAmount, param.Subject, param.NotifyURL)

	// 3. 调用预创建接口
	ctx := context.Background()
	rsp, err := s.client.TradePreCreate(ctx, param)
	if err != nil {
		// 网络错误或请求错误，记录详细错误并尝试使用页面支付作为备选
		fmt.Printf("支付宝TradePreCreate请求失败: %v (订单号: %s, 金额: %.2f)\n", err, order.OrderNo, amount)
		pageURL, pageErr := s.createPagePayURL(order, amount)
		if pageErr != nil {
			// 如果页面支付也失败，返回详细错误信息
			return "", fmt.Errorf("支付宝预创建失败: %v, 页面支付也失败: %v", err, pageErr)
		}
		fmt.Printf("使用页面支付作为备选方案 (订单号: %s)\n", order.OrderNo)
		return pageURL, nil
	}

	// 4. 检查响应
	if rsp.IsFailure() {
		// 支付宝返回业务错误，记录详细错误信息
		errorMsg := fmt.Sprintf("支付宝返回错误: Code=%s, Msg=%s", rsp.Code, rsp.Msg)
		if rsp.SubMsg != "" {
			errorMsg += fmt.Sprintf(", SubMsg=%s", rsp.SubMsg)
		}
		fmt.Printf("支付宝TradePreCreate业务失败: %s (订单号: %s, 金额: %.2f)\n", errorMsg, order.OrderNo, amount)

		// 常见错误码提示和处理
		if rsp.Code == "40004" {
			errorMsg += "。提示：请检查 AppID 和应用私钥是否匹配，以及是否在支付宝后台正确配置了应用公钥"
		} else if rsp.Code == "40001" {
			errorMsg += "。提示：请检查签名是否正确，确保私钥格式正确（PKCS1或PKCS8格式的PEM）"
		} else if rsp.Code == "40006" {
			// 40006 表示接口调用权限不足，通常是应用未签约相应产品
			errorMsg += "。提示：ISV权限不足，应用未签约相应产品。请登录支付宝开放平台，在应用管理中签约\"当面付\"产品，并确保应用已上线。详细步骤请查看支付配置页面的说明。"
			// 权限错误不应该降级到页面支付，因为页面支付也需要相同的权限
			return "", fmt.Errorf("%s。解决方案：1) 登录 https://open.alipay.com 2) 进入应用管理 3) 签约\"当面付\"产品 4) 确保应用状态为\"已上线\"", errorMsg)
		}

		// 对于其他错误，尝试使用页面支付作为备选
		// 但如果是权限相关错误（40006），不应该降级
		if rsp.Code != "40006" {
			pageURL, pageErr := s.createPagePayURL(order, amount)
			if pageErr != nil {
				// 如果页面支付也失败，返回详细错误信息
				return "", fmt.Errorf("%s, 页面支付也失败: %v", errorMsg, pageErr)
			}
			fmt.Printf("使用页面支付作为备选方案 (订单号: %s)\n", order.OrderNo)
			return pageURL, nil
		}

		// 如果是40006错误，直接返回错误，不尝试页面支付
		return "", fmt.Errorf("%s", errorMsg)
	}

	// 5. 返回二维码URL
	// 注意：响应中的字段名是 qr_code（小写+下划线），SDK 会自动映射到 QRCode
	if rsp.QRCode != "" {
		fmt.Printf("支付宝TradePreCreate成功，二维码URL: %s (订单号: %s, 金额: %.2f, 环境: %s)\n",
			rsp.QRCode, order.OrderNo, amount, map[bool]string{true: "生产", false: "沙箱"}[s.isProduction])
		return rsp.QRCode, nil
	}

	// 6. 如果二维码为空，使用页面支付作为备选
	fmt.Printf("支付宝返回的二维码为空，使用页面支付作为备选 (订单号: %s)\n", order.OrderNo)
	pageURL, pageErr := s.createPagePayURL(order, amount)
	if pageErr != nil {
		return "", fmt.Errorf("支付宝返回的二维码为空，且页面支付失败: %v", pageErr)
	}
	return pageURL, nil
}

// createPagePayURL 创建支付页面URL（备选方案）
func (s *AlipayService) createPagePayURL(order *models.Order, amount float64) (string, error) {
	// 验证必要参数
	if order.OrderNo == "" {
		return "", fmt.Errorf("订单号不能为空")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额必须大于0")
	}
	if s.notifyURL == "" {
		return "", fmt.Errorf("异步回调地址未配置")
	}

	param := alipay.TradePagePay{}
	param.NotifyURL = s.notifyURL
	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}
	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	param.ProductCode = "FAST_INSTANT_TRADE_PAY"

	fmt.Printf("支付宝TradePagePay请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s\n",
		param.OutTradeNo, param.TotalAmount, param.Subject, param.NotifyURL)

	payURL, err := s.client.TradePagePay(param)
	if err != nil {
		// 检查是否是权限错误
		if strings.Contains(err.Error(), "40006") || strings.Contains(err.Error(), "insufficient") || strings.Contains(err.Error(), "权限") {
			return "", fmt.Errorf("生成支付页面URL失败: %v。提示：ISV权限不足，请登录支付宝开放平台签约\"当面付\"产品并确保应用已上线", err)
		}
		return "", fmt.Errorf("生成支付页面URL失败: %v", err)
	}

	if payURL == nil {
		return "", fmt.Errorf("支付页面URL为空")
	}

	fmt.Printf("支付宝TradePagePay成功，支付页面URL已生成 (订单号: %s, 金额: %.2f)\n", order.OrderNo, amount)
	return payURL.String(), nil
}

// VerifyNotify 验证回调
func (s *AlipayService) VerifyNotify(params map[string]string) bool {
	// 将 map 转换为 url.Values
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	// 使用 SDK 的验证方法
	if err := s.client.VerifySign(values); err != nil {
		return false
	}

	return true
}

// DecodeNotification 解析异步通知
func (s *AlipayService) DecodeNotification(params map[string]string) (*AlipayNotification, error) {
	// 将 map 转换为 url.Values
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	// 使用 SDK 的解析方法
	notification, err := s.client.DecodeNotification(values)
	if err != nil {
		return nil, err
	}

	// 转换为内部结构
	return &AlipayNotification{
		NotifyID:      notification.NotifyId,
		TradeNo:       notification.TradeNo,
		OutTradeNo:    notification.OutTradeNo,
		TradeStatus:   string(notification.TradeStatus),
		TotalAmount:   notification.TotalAmount,
		ReceiptAmount: notification.ReceiptAmount,
		BuyerID:       notification.BuyerId,
		BuyerLogonID:  notification.BuyerLogonId,
		SellerID:      notification.SellerId,
		SellerEmail:   notification.SellerEmail,
		GmtPayment:    notification.GmtPayment,
	}, nil
}

// QueryOrder 查询订单支付状态
// 用于主动查询支付状态，当回调未及时到达时使用
func (s *AlipayService) QueryOrder(orderNo string) (*AlipayQueryResult, error) {
	if orderNo == "" {
		return nil, fmt.Errorf("订单号不能为空")
	}

	param := alipay.TradeQuery{}
	param.OutTradeNo = orderNo

	ctx := context.Background()
	rsp, err := s.client.TradeQuery(ctx, param)
	if err != nil {
		return nil, fmt.Errorf("查询订单失败: %v", err)
	}

	if rsp.IsFailure() {
		return nil, fmt.Errorf("支付宝返回错误: Code=%s, Msg=%s", rsp.Code, rsp.Msg)
	}

	result := &AlipayQueryResult{
		TradeNo:      rsp.TradeNo,
		OutTradeNo:   rsp.OutTradeNo,
		TradeStatus:  string(rsp.TradeStatus),
		TotalAmount:  rsp.TotalAmount,
		BuyerLogonID: rsp.BuyerLogonId,
	}

	return result, nil
}

// AlipayQueryResult 支付宝查询结果
type AlipayQueryResult struct {
	TradeNo      string
	OutTradeNo   string
	TradeStatus  string // WAIT_BUYER_PAY, TRADE_SUCCESS, TRADE_FINISHED, TRADE_CLOSED
	TotalAmount  string
	BuyerLogonID string
}

// IsPaid 判断是否已支付
func (r *AlipayQueryResult) IsPaid() bool {
	return r.TradeStatus == "TRADE_SUCCESS" || r.TradeStatus == "TRADE_FINISHED"
}

// AlipayNotification 支付宝通知结构
type AlipayNotification struct {
	NotifyID      string
	TradeNo       string
	OutTradeNo    string
	TradeStatus   string
	TotalAmount   string
	ReceiptAmount string
	BuyerID       string
	BuyerLogonID  string
	SellerID      string
	SellerEmail   string
	GmtPayment    string
}
