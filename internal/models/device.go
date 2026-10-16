package models

import (
	"time"
)

type Device struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	UserID            *int64  `gorm:"index" json:"user_id,omitempty"`
	SubscriptionID    uint    `gorm:"index;not null;index:idx_device_sub_active,priority:1" json:"subscription_id"`
	DeviceFingerprint string  `gorm:"type:varchar(255);not null" json:"device_fingerprint"`
	DeviceHash        *string `gorm:"type:varchar(255)" json:"device_hash,omitempty"`
	// DeviceHwFp 客户端上报的**硬件指纹**（X-MF-Device-Fp）：hash(平台 +
	// Windows MachineGuid / macOS IOPlatformUUID / Android ANDROID_ID / iOS
	// identifierForVendor)。与安装实例 ID 的区别：
	//   - 安装 ID（device_hash）：重装 App / 清数据就变；
	//   - 硬件指纹：同一台物理机跨重装不变，用于"认亲"复用同一行，
	//     避免每次重装都多出一行幽灵设备白占名额（用户实测 susanzhang1230：
	//     一台 Windows 因为 2.2.25→2.2.28 升级被算成两台）。
	// 存量行为空，不影响老数据；只在客户端上报时参与匹配。
	DeviceHwFp       *string `gorm:"type:varchar(64);index" json:"device_hw_fp,omitempty"`
	DeviceUA         *string `gorm:"type:varchar(255)" json:"device_ua,omitempty"`
	DeviceName       *string `gorm:"type:varchar(100)" json:"device_name,omitempty"`
	DeviceType       *string `gorm:"type:varchar(50)" json:"device_type,omitempty"`
	IPAddress        *string `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	Location         *string `gorm:"type:varchar(255)" json:"location,omitempty"` // GeoIP 位置信息
	UserAgent        *string `gorm:"type:text" json:"user_agent,omitempty"`
	SoftwareName     *string `gorm:"type:varchar(100)" json:"software_name,omitempty"`
	SoftwareVersion  *string `gorm:"type:varchar(50)" json:"software_version,omitempty"`
	OSName           *string `gorm:"type:varchar(50)" json:"os_name,omitempty"`
	OSVersion        *string `gorm:"type:varchar(50)" json:"os_version,omitempty"`
	DeviceModel      *string `gorm:"type:varchar(100)" json:"device_model,omitempty"`
	DeviceBrand      *string `gorm:"type:varchar(50)" json:"device_brand,omitempty"`
	SubscriptionType *string `gorm:"type:varchar(20);index" json:"subscription_type,omitempty"` // 订阅类型: clash, v2ray, ssr
	IsActive         bool    `gorm:"default:true;index;index:idx_device_sub_active,priority:2" json:"is_active"`
	IsAllowed        bool    `gorm:"default:true" json:"is_allowed"`
	// KickedAt 设备被「踢下线」时间（用户在设备管理删除设备 = 软删除 + 踢下线）：
	// 行保留但 IsActive=false，该设备再次拉订阅时被拒并提示「已被移除」，
	// 无法静默重新注册（防回注册）；重置订阅/管理员物理删除可清。
	KickedAt    *time.Time `gorm:"index" json:"kicked_at,omitempty"`
	FirstSeen   *time.Time `json:"first_seen,omitempty"`
	LastAccess  time.Time  `gorm:"autoCreateTime" json:"last_access"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	AccessCount int        `gorm:"default:0" json:"access_count"`
	Remark      *string    `gorm:"type:varchar(255)" json:"remark,omitempty"` // 用户自定义备注
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

func (Device) TableName() string {
	return "devices"
}
