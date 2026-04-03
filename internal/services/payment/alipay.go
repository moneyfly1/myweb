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
		return nil, fmt.Errorf("支付宝应用私钥格式错误：无法识别私钥格式。请确保私钥是完整的PEM格式")
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
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	if paymentConfig.AlipayPublicKey.Valid && paymentConfig.AlipayPublicKey.String != "" {
		publicKey := utils.NormalizePublicKey(paymentConfig.AlipayPublicKey.String)
		if publicKey != "" {
			if err := client.LoadAliPayPublicKey(publicKey); err != nil {
				return nil, fmt.Errorf("加载支付宝公钥失败: %v", err)
			}
		} else {
			return nil, fmt.Errorf("支付宝公钥格式无法识别，请提供完整的PEM格式公钥")
		}
	} else {
		return nil, fmt.Errorf("未配置支付宝公钥，无法验证回调签名")
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

// CreatePayment 创建支付请求
// 注意：对于个人开发者，这里返回的字符串不再是一个可以直接在浏览器打开的网页URL！
// 返回的是一个二维码字符串 (例如: https://qr.alipay.com/bax00...)
// 你的前端拿到这个字符串后，必须使用二维码库将其渲染成图片让用户扫码。
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

	// 个人账号专属：使用当面付 (alipay.trade.precreate)
	qrCodeStr, err := s.createPrecreatePay(order, amount)
	if err != nil {
		utils.LogWarn("当面付(TradePreCreate)调用失败: %v", err)
		return "", fmt.Errorf("创建支付宝扫码支付失败: %v", err)
	}

	return qrCodeStr, nil
}

// createPrecreatePay 调用当面付接口获取二维码字符串
func (s *AlipayService) createPrecreatePay(order *models.Order, amount float64) (string, error) {
	var param = alipay.TradePreCreate{}

	param.OutTradeNo = order.OrderNo
	param.Subject = fmt.Sprintf("订单支付-%s", order.OrderNo)
	param.TotalAmount = fmt.Sprintf("%.2f", amount)

	// 当面付不需要 ReturnURL (因为是扫码，不会跳转前端页面)
	// 只需要异步通知 NotifyURL
	if s.notifyURL != "" {
		param.NotifyURL = s.notifyURL
	}

	utils.LogInfo("支付宝当面付请求参数: OutTradeNo=%s, TotalAmount=%s, Subject=%s",
		param.OutTradeNo, param.TotalAmount, param.Subject)

	// 注意：V3版本的 API 发起网络请求需要传入 Context
	ctx := context.Background()
	rsp, err := s.client.TradePreCreate(ctx, param)
	if err != nil {
		return "", err
	}

	// 检查业务是否成功
	if rsp.IsFailure() {
		// 如果这里依然报 40006 权限不足，说明你在开放平台还没有签约“当面付”，或者应用未上线
		return "", fmt.Errorf("支付宝返回失败: Code=%s, Msg=%s, SubCode=%s, SubMsg=%s",
			rsp.Code, rsp.Msg, rsp.SubCode, rsp.SubMsg)
	}

	if rsp.QRCode == "" {
		return "", fmt.Errorf("支付宝未返回二维码信息")
	}

	utils.LogInfo("支付宝当面付创建成功，返回二维码数据 (订单号: %s)", order.OrderNo)
	return rsp.QRCode, nil
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
		return nil, fmt.Errorf("支付宝返回错误: Code=%s, Msg=%s, SubCode=%s", rsp.Code, rsp.Msg, rsp.SubCode)
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
