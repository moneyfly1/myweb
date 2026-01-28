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

type AlipayService struct {
	client       *alipay.Client
	notifyURL    string
	returnURL    string
	isProduction bool
}

func NewAlipayService(paymentConfig *models.PaymentConfig) (*AlipayService, error) {
	appID := ""
	if paymentConfig.AppID.Valid {
		appID = strings.TrimSpace(paymentConfig.AppID.String)
	}
	if appID == "" {
		return nil, fmt.Errorf("支付宝 AppID 未配置，请在支付配置中设置 AppID")
	}

	privateKey := ""
	if paymentConfig.MerchantPrivateKey.Valid {
		privateKey = strings.TrimSpace(paymentConfig.MerchantPrivateKey.String)
	}
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝应用私钥未配置，请使用支付宝开发平台开发助手生成私钥并配置")
	}

	privateKey = utils.NormalizePrivateKey(privateKey)
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝应用私钥格式错误：无法识别私钥格式。请确保私钥是完整的PEM格式（包含BEGIN和END标记）")
	}

	isProduction := false
	var opts []alipay.OptionFunc

	if paymentConfig.ConfigJSON.Valid {
		var configData map[string]interface{}
		if err := json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &configData); err == nil {
			if prod, ok := configData["is_production"].(bool); ok {
				isProduction = prod
			} else if gatewayURL, ok := configData["gateway_url"].(string); ok && gatewayURL != "" {
				isProduction = !strings.Contains(strings.ToLower(gatewayURL), "alipaydev.com")
			}

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

	client, err := alipay.New(appID, privateKey, isProduction, opts...)
	if err != nil {
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v。请检查：1) AppID是否正确 2) 应用私钥是否为完整的PEM格式（PKCS1或PKCS8）3) 私钥是否与AppID匹配 4) 私钥长度是否为2048位（推荐）", err)
	}

	if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
		publicKey := utils.NormalizePublicKey(paymentConfig.AlipayPublicKey.String)
		if publicKey != "" {
			if err := client.LoadAliPayPublicKey(publicKey); err != nil {
				utils.LogWarn("加载支付宝公钥失败（不影响创建支付，但会影响回调验证）: %v。请检查支付宝公钥格式是否正确", err)
			} else {
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

	if paymentConfig.NotifyURL.Valid && paymentConfig.NotifyURL.String != "" {
		service.notifyURL = strings.TrimSpace(paymentConfig.NotifyURL.String)
	} else {
		utils.LogInfo("支付宝回调地址未配置，将使用支付宝后台配置的地址")
		service.notifyURL = ""
	}

	if paymentConfig.ReturnURL.Valid && paymentConfig.ReturnURL.String != "" {
		service.returnURL = strings.TrimSpace(paymentConfig.ReturnURL.String)
	}

	return service, nil
}

func (s *AlipayService) CreatePayment(order *models.Order, amount float64) (string, error) {
	if order == nil {
		return "", fmt.Errorf("订单信息不能为空")
	}
	if order.OrderNo == "" {
		return "", fmt.Errorf("订单号不能为空")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额必须大于0，当前金额: %.2f", amount)
	}

	var param = alipay.TradePreCreate{}

	if s.notifyURL != "" {
		param.NotifyURL = s.notifyURL
		utils.LogInfo("支付宝回调地址(NotifyURL)已设置: %s", s.notifyURL)
	} else {
		utils.LogWarn("支付宝回调地址未配置，将使用支付宝后台配置的地址（如果后台也未配置，将无法收到回调）")
	}

	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}

	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	param.ProductCode = "" // 明确设置为空，避免使用默认值

	utils.LogInfo("支付宝TradePreCreate请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s",
		param.OutTradeNo, param.TotalAmount, param.Subject, param.NotifyURL)

	ctx := context.Background()
	rsp, err := s.client.TradePreCreate(ctx, param)
	if err != nil {
		utils.LogErrorMsg("支付宝TradePreCreate请求失败: %v (订单号: %s, 金额: %.2f)", err, order.OrderNo, amount)
		pageURL, pageErr := s.createPagePayURL(order, amount)
		if pageErr != nil {
			return "", fmt.Errorf("支付宝预创建失败: %v, 页面支付也失败: %v", err, pageErr)
		}
		utils.LogInfo("使用页面支付作为备选方案 (订单号: %s)", order.OrderNo)
		return pageURL, nil
	}

	if rsp.IsFailure() {
		errorMsg := fmt.Sprintf("支付宝返回错误: Code=%s, Msg=%s", rsp.Code, rsp.Msg)
		if rsp.SubMsg != "" {
			errorMsg += fmt.Sprintf(", SubMsg=%s", rsp.SubMsg)
		}
		utils.LogErrorMsg("支付宝TradePreCreate业务失败: %s (订单号: %s, 金额: %.2f)", errorMsg, order.OrderNo, amount)

		if rsp.Code == "40004" {
			errorMsg += "。提示：请检查 AppID 和应用私钥是否匹配，以及是否在支付宝后台正确配置了应用公钥"
		} else if rsp.Code == "40001" {
			errorMsg += "。提示：请检查签名是否正确，确保私钥格式正确（PKCS1或PKCS8格式的PEM）"
		} else if rsp.Code == "40006" {
			errorMsg += "。提示：ISV权限不足，应用未签约相应产品。请登录支付宝开放平台，在应用管理中签约\"当面付\"产品，并确保应用已上线。详细步骤请查看支付配置页面的说明。"
			return "", fmt.Errorf("%s。解决方案：1) 登录 https://open.alipay.com 2) 进入应用管理 3) 签约\"当面付\"产品 4) 确保应用状态为\"已上线\"", errorMsg)
		}

		if rsp.Code != "40006" {
			pageURL, pageErr := s.createPagePayURL(order, amount)
			if pageErr != nil {
				return "", fmt.Errorf("%s, 页面支付也失败: %v", errorMsg, pageErr)
			}
			utils.LogInfo("使用页面支付作为备选方案 (订单号: %s)", order.OrderNo)
			return pageURL, nil
		}

		return "", fmt.Errorf("%s", errorMsg)
	}

	if rsp.QRCode != "" {
		utils.LogInfo("支付宝TradePreCreate成功，二维码URL: %s (订单号: %s, 金额: %.2f, 环境: %s)",
			rsp.QRCode, order.OrderNo, amount, map[bool]string{true: "生产", false: "沙箱"}[s.isProduction])
		return rsp.QRCode, nil
	}

	utils.LogWarn("支付宝返回的二维码为空，使用页面支付作为备选 (订单号: %s)", order.OrderNo)
	pageURL, pageErr := s.createPagePayURL(order, amount)
	if pageErr != nil {
		return "", fmt.Errorf("支付宝返回的二维码为空，且页面支付失败: %v", pageErr)
	}
	return pageURL, nil
}

func (s *AlipayService) createPagePayURL(order *models.Order, amount float64) (string, error) {
	if order.OrderNo == "" {
		return "", fmt.Errorf("订单号不能为空")
	}
	if amount <= 0 {
		return "", fmt.Errorf("支付金额必须大于0")
	}

	var param = alipay.TradePagePay{}

	if s.notifyURL != "" {
		param.NotifyURL = s.notifyURL
	}
	if s.returnURL != "" {
		param.ReturnURL = s.returnURL
	}

	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.OutTradeNo = order.OrderNo
	param.TotalAmount = fmt.Sprintf("%.2f", amount)
	param.ProductCode = "FAST_INSTANT_TRADE_PAY"

	utils.LogInfo("支付宝TradePagePay请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s, NotifyURL=%s",
		param.OutTradeNo, param.TotalAmount, param.Subject, param.NotifyURL)

	payURL, err := s.client.TradePagePay(param)
	if err != nil {
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

func (s *AlipayService) ParseNotification(req *http.Request) (*AlipayNotification, error) {
	notification, err := s.client.GetTradeNotification(req)
	if err != nil {
		return nil, fmt.Errorf("解析或验证支付宝通知失败: %v", err)
	}

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

func (s *AlipayService) VerifyNotify(params map[string]string) bool {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	if err := s.client.VerifySign(values); err != nil {
		return false
	}

	return true
}

func (s *AlipayService) DecodeNotification(params map[string]string) (*AlipayNotification, error) {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	notification, err := s.client.DecodeNotification(values)
	if err != nil {
		return nil, err
	}

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

type AlipayQueryResult struct {
	TradeNo      string
	OutTradeNo   string
	TradeStatus  string // WAIT_BUYER_PAY, TRADE_SUCCESS, TRADE_FINISHED, TRADE_CLOSED
	TotalAmount  string
	BuyerLogonID string
}

func (r *AlipayQueryResult) IsPaid() bool {
	return r.TradeStatus == "TRADE_SUCCESS" || r.TradeStatus == "TRADE_FINISHED"
}

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
