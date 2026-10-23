// Package adminutil 提供「按用户名/邮箱定位管理员账号」的唯一实现。
//
// 为什么单独成包：这段逻辑此前在三个地方各抄了一遍 —— cmd/server 的
// ensureDefaultAdmin、scripts/admin_tool 的 updatePassword 与 createOrUpdateAdmin，
// 连"不要用 username = ? OR email = ? 单查询"这条防误改注释都抄了三份。
// 抄多份的直接风险是它们会漂移：任意一处改了匹配规则，另外两处还在按旧规则改账号。
package adminutil

import (
	"errors"
	"strings"

	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// 默认管理员身份（各调用方可用环境变量覆盖）。
const (
	DefaultUsername = "admin"
	DefaultEmail    = "admin@example.com"
)

// ErrNotFound 两个身份都没匹配到账号。
var ErrNotFound = errors.New("未找到管理员账号")

// FindByIdentities 先按用户名精确匹配，未命中再按邮箱匹配。
//
// 必须分两次查询、不能用 `username = ? OR email = ?` 的单查询：
// 用户名命中 A、邮箱命中 B 时，OR + First 只会取主键靠前的那一行，
// 于是"重置管理员密码"可能改到另一个账号上。
func FindByIdentities(db *gorm.DB, username, email string) (models.User, error) {
	var user models.User
	err := db.Where("username = ?", username).First(&user).Error
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return user, err
	}
	err = db.Where("email = ?", email).First(&user).Error
	if err == nil {
		return user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user, ErrNotFound
	}
	return user, err
}

// HasSimilarUsername 判断是否已存在「仅大小写不同」的同名用户。
//
// 登录区分大小写，若库里已有 "Admin" 而这里又建 "admin"，会出现两个肉眼无法
// 区分的账号（后台看起来一模一样，改密码只改其中一个）。
func HasSimilarUsername(db *gorm.DB, username string) (models.User, bool) {
	var same models.User
	if err := db.Where("LOWER(username) = ?", strings.ToLower(username)).First(&same).Error; err != nil {
		return models.User{}, false
	}
	return same, true
}
