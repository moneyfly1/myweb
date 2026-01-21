package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

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
	privateKey = utils.NormalizePrivateKey(privateKey)
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
					utils.LogInfo("使用支付宝沙箱老网关地址")
				} else {
					opts = append(opts, alipay.WithNewSandboxGateway())
					utils.LogInfo("使用支付宝沙箱新网关地址（默认）")
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

	// 将初始化日志改为 DEBUG 级别，避免频繁轮询时产生大量日志
	// utils.LogInfo("支付宝客户端初始化成功: AppID=%s, 环境=%s", appID, map[bool]string{true: "生产环境", false: "沙箱环境"}[isProduction])

	// 6. 加载支付宝公钥（普通公钥模式）
	// 注意：此处使用的是支付宝公钥，不是应用公钥
	// 参考：https://opendocs.alipay.com/common/057aqe
	// 公钥主要用于验证回调签名，创建支付时不是必须的
	// 但如果配置了公钥，应该确保格式正确
	if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
		publicKey := utils.NormalizePublicKey(paymentConfig.AlipayPublicKey.String)
		if publicKey != "" {
			if err := client.LoadAliPayPublicKey(publicKey); err != nil {
				// 公钥加载失败不影响创建支付，但会影响回调验证
				// 这里只记录警告，不返回错误
				utils.LogWarn("加载支付宝公钥失败（不影响创建支付，但会影响回调验证）: %v。请检查支付宝公钥格式是否正确", err)
			} else {
				// 将公钥加载日志改为 DEBUG 级别，避免频繁轮询时产生大量日志
				// utils.LogInfo("支付宝公钥加载成功")
			}
		} else {
			utils.LogWarn("支付宝公钥格式无法识别，回调验证可能失败")
		}
	} else {
		utils.LogWarn("未配置支付宝公钥，回调验证可能失败。建议配置支付宝公钥以确保回调安全")
	}

	service := &AlipayService{
		client:       client,
		isProduction: isProduction,
	}

	// 设置回调地址（可选）
	// 如果未配置回调地址，则不设置 s.notifyURL，创建支付时将不传递 notify_url 参数
	// 此时支付宝将使用应用在支付宝后台配置的回调地址
	if paymentConfig.NotifyURL.Valid && paymentConfig.NotifyURL.String != "" {
		service.notifyURL = strings.TrimSpace(paymentConfig.NotifyURL.String)
	} else {
		// 允许为空，使用支付宝后台配置
		utils.LogInfo("支付宝回调地址未配置，将使用支付宝后台配置的地址")
		service.notifyURL = ""
	}

	if paymentConfig.ReturnURL.Valid && paymentConfig.ReturnURL.String != "" {
		service.returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
	}

	return service, nil
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

	// 2. 创建支付参数 (关键修复: 必须先初始化 param 变量!)
	var param = alipay.TradePreCreate{}

	// 3. 设置回调地址
	// 如果配置了 notifyURL 则使用，否则不传（走支付宝后台配置）
	if s.notifyURL != "" {
		param.NotifyURL = s.notifyURL
		utils.LogInfo("支付宝回调地址(NotifyURL)已设置: %s", s.notifyURL)
	} else {
		utils.LogWarn("支付宝回调地址未配置，将使用支付宝后台配置的地址（如果后台也未配置，将无法收到回调）")
	}

	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}

	// 4. 设置订单信息
	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	// 重要：TradePreCreate（当面付-生成二维码）接口不需要 ProductCode 字段
	// 如果设置了 ProductCode，应该设置为空字符串，不要使用 FAST_INSTANT_TRADE_PAY
	// FAST_INSTANT_TRADE_PAY 是电脑网站支付的产品码，会导致权限不足错误
	param.ProductCode = "" // 明确设置为空，避免使用默认值

	// 5. 记录请求参数（便于排查问题）
	utils.LogInfo("支付宝TradePreCreate请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s",
		param.OutTradeNo, param.TotalAmount, param.Subject, param.NotifyURL)

	// 3. 调用预创建接口
	ctx := context.Background()
	rsp, err := s.client.TradePreCreate(ctx, param)
	if err != nil {
		// 网络错误或请求错误，记录详细错误并尝试使用页面支付作为备选
		utils.LogErrorMsg("支付宝TradePreCreate请求失败: %v (订单号: %s, 金额: %.2f)", err, order.OrderNo, amount)
		pageURL, pageErr := s.createPagePayURL(order, amount)
		if pageErr != nil {
			// 如果页面支付也失败，返回详细错误信息
			return "", fmt.Errorf("支付宝预创建失败: %v, 页面支付也失败: %v", err, pageErr)
		}
		utils.LogInfo("使用页面支付作为备选方案 (订单号: %s)", order.OrderNo)
		return pageURL, nil
	}

	// 4. 检查响应
	if rsp.IsFailure() {
		// 支付宝返回业务错误，记录详细错误信息
		errorMsg := fmt.Sprintf("支付宝返回错误: Code=%s, Msg=%s", rsp.Code, rsp.Msg)
		if rsp.SubMsg != "" {
			errorMsg += fmt.Sprintf(", SubMsg=%s", rsp.SubMsg)
		}
		utils.LogErrorMsg("支付宝TradePreCreate业务失败: %s (订单号: %s, 金额: %.2f)", errorMsg, order.OrderNo, amount)

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
			utils.LogInfo("使用页面支付作为备选方案 (订单号: %s)", order.OrderNo)
			return pageURL, nil
		}

		// 如果是40006错误，直接返回错误，不尝试页面支付
		return "", fmt.Errorf("%s", errorMsg)
	}

	// 5. 返回二维码URL
	// 注意：响应中的字段名是 qr_code（小写+下划线），SDK 会自动映射到 QRCode
	if rsp.QRCode != "" {
		utils.LogInfo("支付宝TradePreCreate成功，二维码URL: %s (订单号: %s, 金额: %.2f, 环境: %s)",
			rsp.QRCode, order.OrderNo, amount, map[bool]string{true: "生产", false: "沙箱"}[s.isProduction])
		return rsp.QRCode, nil
	}

	// 6. 如果二维码为空，使用页面支付作为备选
	utils.LogWarn("支付宝返回的二维码为空，使用页面支付作为备选 (订单号: %s)", order.OrderNo)
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

	// 创建支付参数 (关键修复: 必须先初始化 param 变量!)
	var param = alipay.TradePagePay{}

	// 设置回调地址
	if s.notifyURL != "" {
		param.NotifyURL = s.notifyURL
	}
	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}

	// 设置订单信息
	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	param.ProductCode = "FAST_INSTANT_TRADE_PAY"

	utils.LogInfo("支付宝TradePagePay请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s",
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

	utils.LogInfo("支付宝TradePagePay成功，支付页面URL已生成 (订单号: %s, 金额: %.2f)", order.OrderNo, amount)
	return payURL.String(), nil
}

// ParseNotification 解析并验证异步通知
// 直接使用 SDK 的 GetTradeNotification 方法，它会自动处理签名验证和参数解析
func (s *AlipayService) ParseNotification(req *http.Request) (*AlipayNotification, error) {
	// 使用 SDK 解析请求
	notification, err := s.client.GetTradeNotification(req)
	if err != nil {
		return nil, fmt.Errorf("解析或验证支付宝通知失败: %v", err)
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

// VerifyNotify 验证回调 (已废弃，建议使用 ParseNotification)
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

// DecodeNotification 解析异步通知 (已废弃，建议使用 ParseNotification)
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
