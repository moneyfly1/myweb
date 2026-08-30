package payment

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cboard-go/internal/models"
)

// ============================================================
// USDT 加密货币支付（XBoard 的 EPUSDT 类似）
// 生成订单 → 展示钱包地址/二维码 → 用户转账后确认 → 管理员/系统核对。
// 简化实现：展示钱包地址 + 金额，用户转账后通过「确认已支付」回调由管理员核对放行；
// 支持 TRC20 / ERC20 网络（配置中指定）。
// ============================================================

type USDTService struct {
	walletAddress string
	network       string // TRC20 / ERC20
	confirmations int
}

func NewUSDTService(paymentConfig *models.PaymentConfig) (*USDTService, error) {
	address := ""
	if paymentConfig.WalletAddress.Valid {
		address = strings.TrimSpace(paymentConfig.WalletAddress.String)
	}
	if address == "" {
		return nil, fmt.Errorf("USDT 钱包地址未配置")
	}
	svc := &USDTService{
		walletAddress: address,
		network:       "TRC20",
		confirmations: 1,
	}
	if paymentConfig.ConfigJSON.Valid {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(paymentConfig.ConfigJSON.String), &cfg) == nil {
			if net, ok := cfg["network"].(string); ok && net != "" {
				svc.network = net
			}
			if conf, ok := cfg["confirmations"].(float64); ok && conf > 0 {
				svc.confirmations = int(conf)
			}
		}
	}
	return svc, nil
}

// PaymentInfo USDT 支付信息（返回给前端展示）
type PaymentInfo struct {
	WalletAddress string  `json:"wallet_address"`
	Network       string  `json:"network"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	OrderNo       string  `json:"order_no"`
	ExpireAt      string  `json:"expire_at"`
	Memo          string  `json:"memo,omitempty"`
}

// CreatePayment 生成 USDT 支付信息（地址 + 金额 + 过期时间）
func (s *USDTService) CreatePayment(order *models.Order, amount float64) (*PaymentInfo, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("订单金额无效")
	}
	expireAt := time.Now().Add(30 * time.Minute)
	return &PaymentInfo{
		WalletAddress: s.walletAddress,
		Network:       s.network,
		Amount:        amount,
		Currency:      "USDT",
		OrderNo:       order.OrderNo,
		ExpireAt:      expireAt.Format("2006-01-02 15:04:05"),
		Memo:          order.OrderNo,
	}, nil
}

// GetConfirmations 返回配置的确认数
func (s *USDTService) GetConfirmations() int {
	if s.confirmations <= 0 {
		return 1
	}
	return s.confirmations
}
