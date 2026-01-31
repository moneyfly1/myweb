package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type PaymentConfig struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	PayType              string         `gorm:"type:varchar(50);not null" json:"pay_type"`
	AppID                sql.NullString `gorm:"type:text" json:"app_id,omitempty"`
	MerchantPrivateKey   sql.NullString `gorm:"type:text" json:"merchant_private_key,omitempty"`
	AlipayPublicKey      sql.NullString `gorm:"type:text" json:"alipay_public_key,omitempty"`
	WechatAppID          sql.NullString `gorm:"type:text" json:"wechat_app_id,omitempty"`
	WechatMchID          sql.NullString `gorm:"type:text" json:"wechat_mch_id,omitempty"`
	WechatAPIKey         sql.NullString `gorm:"type:text" json:"wechat_api_key,omitempty"`
	PaypalClientID       sql.NullString `gorm:"type:text" json:"paypal_client_id,omitempty"`
	PaypalSecret         sql.NullString `gorm:"type:text" json:"paypal_secret,omitempty"`
	StripePublishableKey sql.NullString `gorm:"type:text" json:"stripe_publishable_key,omitempty"`
	StripeSecretKey      sql.NullString `gorm:"type:text" json:"stripe_secret_key,omitempty"`
	BankName             sql.NullString `gorm:"type:text" json:"bank_name,omitempty"`
	AccountName          sql.NullString `gorm:"type:text" json:"account_name,omitempty"`
	AccountNumber        sql.NullString `gorm:"type:text" json:"account_number,omitempty"`
	WalletAddress        sql.NullString `gorm:"type:text" json:"wallet_address,omitempty"`
	Status               int            `gorm:"default:1" json:"status"`
	ReturnURL            sql.NullString `gorm:"type:text" json:"return_url,omitempty"`
	NotifyURL            sql.NullString `gorm:"type:text" json:"notify_url,omitempty"`
	SortOrder            int            `gorm:"default:0" json:"sort_order"`
	ConfigJSON           sql.NullString `gorm:"type:json" json:"config_json,omitempty"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}

func (p *PaymentConfig) GetConfig() map[string]interface{} {
	config := map[string]interface{}{
		"pay_type":   p.PayType,
		"status":     p.Status,
		"sort_order": p.SortOrder,
	}

	if p.ReturnURL.Valid {
		config["return_url"] = p.ReturnURL.String
	}
	if p.NotifyURL.Valid {
		config["notify_url"] = p.NotifyURL.String
	}

	switch p.PayType {
	case "alipay":
		if p.AppID.Valid {
			config["app_id"] = p.AppID.String
		}
		if p.MerchantPrivateKey.Valid {
			config["merchant_private_key"] = p.MerchantPrivateKey.String
		}
		if p.AlipayPublicKey.Valid {
			config["alipay_public_key"] = p.AlipayPublicKey.String
		}
	case "wechat":
		if p.WechatAppID.Valid {
			config["app_id"] = p.WechatAppID.String
		}
		if p.WechatMchID.Valid {
			config["mch_id"] = p.WechatMchID.String
		}
		if p.WechatAPIKey.Valid {
			config["api_key"] = p.WechatAPIKey.String
		}
	case "crypto":
		if p.WalletAddress.Valid {
			config["wallet_address"] = p.WalletAddress.String
		}
	case "yipay", "yipay_alipay", "yipay_wxpay", "yipay_qqpay":
		if p.AppID.Valid {
			config["pid"] = p.AppID.String
			config["app_id"] = p.AppID.String
		}
		if p.MerchantPrivateKey.Valid {
			config["key"] = p.MerchantPrivateKey.String
			config["merchant_private_key"] = p.MerchantPrivateKey.String
		}
	}

	if p.ConfigJSON.Valid {
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(p.ConfigJSON.String), &jsonData); err == nil {
			for k, v := range jsonData {
				config[k] = v
			}
		}
	}

	return config
}
