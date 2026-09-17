package handlers

import (
	"strings"

	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

// 用户名在界面、订单、日志中直接呈现，仅大小写不同的两个账号
// （如 alex / Alex、admin）肉眼无法区分，极易造成冒用与误操作。
// 数据库唯一索引区分大小写（SQLite 默认 BINARY 排序规则），
// 因此唯一性一律在本层按“忽略大小写”判定，作为所有写入口的统一闸门。
// 说明：ValidateUsername 只放行 ASCII 字母/数字/下划线/中文，
// 其中仅 ASCII 字母有大小写，故 SQL LOWER() 的 ASCII 折叠已足够覆盖。
const (
	usernameFormatMessage = "用户名格式不正确，长度为2-20个字符，只能包含字母、数字、下划线和中文"
	usernameTakenMessage  = "用户名已被使用（用户名不区分大小写），请选择其他用户名"
)

// usernameTaken 判断用户名是否已被占用（忽略大小写）。
// excludeUserID > 0 时跳过该用户自身，供改资料/后台编辑使用。
func usernameTaken(db *gorm.DB, username string, excludeUserID uint) (bool, error) {
	query := db.Model(&models.User{}).Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(username)))
	if excludeUserID > 0 {
		query = query.Where("id != ?", excludeUserID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// checkUsernameAvailable 统一校验用户名格式与唯一性（忽略大小写）。
// 返回的 error 为数据库层错误，调用方应回 500 而非 400。
func checkUsernameAvailable(db *gorm.DB, username string, excludeUserID uint) (string, error) {
	if !utils.ValidateUsername(username) {
		return usernameFormatMessage, nil
	}
	taken, err := usernameTaken(db, username, excludeUserID)
	if err != nil {
		return "", err
	}
	if taken {
		return usernameTakenMessage, nil
	}
	return "", nil
}
