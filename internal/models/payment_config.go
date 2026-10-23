package models

import (
	"database/sql"
	"time"
)

type PaymentConfig struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	PayType              string         `gorm:"type:varchar(50);not null;index;index:idx_payment_config_lookup,priority:1" json:"pay_type"`
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
	Status               int            `gorm:"default:1;index;index:idx_payment_config_lookup,priority:2" json:"status"`
	ReturnURL            sql.NullString `gorm:"type:text" json:"return_url,omitempty"`
	NotifyURL            sql.NullString `gorm:"type:text" json:"notify_url,omitempty"`
	SortOrder            int            `gorm:"default:0;index:idx_payment_config_lookup,priority:3" json:"sort_order"`
	ConfigJSON           sql.NullString `gorm:"type:json" json:"config_json,omitempty"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (PaymentConfig) TableName() string {
	return "payment_configs"
}
