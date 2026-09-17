package models

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type UserActivity struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index;index:idx_user_activities_user_created_at,priority:1;not null" json:"user_id"`
	ActivityType     string         `gorm:"type:varchar(50);not null" json:"activity_type"`
	Description      sql.NullString `gorm:"type:text" json:"description,omitempty"`
	IPAddress        sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent        sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location         sql.NullString `gorm:"type:text" json:"location,omitempty"` // GeoIP JSON，可能超过 100 字符（MySQL 5.7 varchar 会 1406）
	ActivityMetadata sql.NullString `gorm:"type:json" json:"activity_metadata,omitempty"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index;index:idx_user_activities_user_created_at,priority:2" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (UserActivity) TableName() string {
	return "user_activities"
}

type LoginHistory struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"index;not null" json:"user_id"`
	LoginTime         time.Time      `gorm:"autoCreateTime" json:"login_time"`
	LogoutTime        sql.NullTime   `json:"logout_time,omitempty"`
	IPAddress         sql.NullString `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent         sql.NullString `gorm:"type:text" json:"user_agent,omitempty"`
	Location          sql.NullString `gorm:"type:text" json:"location,omitempty"` // GeoIP JSON，实测最长 144 字符，varchar(100) 会 1406
	DeviceFingerprint sql.NullString `gorm:"type:varchar(255)" json:"device_fingerprint,omitempty"`
	LoginStatus       string         `gorm:"type:varchar(20);default:success" json:"login_status"`
	FailureReason     sql.NullString `gorm:"type:text" json:"failure_reason,omitempty"`
	SessionDuration   sql.NullInt64  `json:"session_duration,omitempty"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (LoginHistory) TableName() string {
	return "login_history"
}

// GetLocationInfo 解析 location 字段，返回国家与城市。
//
// 历史缺陷：此前先判断 strings.Contains(locationStr, ",")，而 JSON 形式
// {"country":"中国","city":"杭州"} 本身含逗号，于是被当作 "国家,城市" 拆分，
// 返回 country = `{"country":"中国"` 这样的垃圾值 —— 前端登录历史的国家筛选框
// 与个人中心因此显示乱码。现在改为「先按 JSON 解析，失败再按 国家,城市 拆分」，
// 并识别 "本地"/"内网" 这类占位值。
func (h *LoginHistory) GetLocationInfo() (country, city string) {
	if !h.Location.Valid || h.Location.String == "" {
		return "", ""
	}
	locationStr := strings.TrimSpace(h.Location.String)

	// 1) 占位值（内网/本地无归属地信息，不作为国家返回）
	if locationStr == "本地" || locationStr == "内网" {
		return "", ""
	}

	// 2) JSON 形式（落库的标准形态）
	if strings.HasPrefix(locationStr, "{") {
		var locationData map[string]interface{}
		if err := json.Unmarshal([]byte(locationStr), &locationData); err == nil {
			if c, ok := locationData["country"].(string); ok {
				country = strings.TrimSpace(c)
			}
			if c, ok := locationData["city"].(string); ok {
				city = strings.TrimSpace(c)
			}
			return country, city
		}
	}

	// 3) 纯文本形式 "国家, 城市"
	if strings.Contains(locationStr, ",") {
		parts := strings.SplitN(locationStr, ",", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}

	// 4) 只有一个值，视为国家
	return locationStr, ""
}
