package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"database/sql"
	"math"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Checkin(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	userID := userIDVal.(uint)

	db := database.GetDB()
	now := utils.GetBeijingTime()
	dayStart, dayEnd := utils.GetDayRange(now)

	var count int64
	db.Model(&models.CheckinRecord{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).Count(&count)
	if count > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "今天已经签到过了", nil)
		return
	}

	amount := math.Floor((rand.Float64()*0.9+0.1)*100) / 100

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "用户不存在", nil)
		return
	}

	balanceBefore := user.Balance
	user.Balance += amount

	tx := db.Begin()
	if err := tx.Model(&user).Update("balance", user.Balance).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "签到失败", nil)
		return
	}

	record := models.CheckinRecord{
		UserID:    userID,
		Amount:    amount,
		CreatedAt: now,
	}
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "签到失败", nil)
		return
	}

	balanceLog := models.BalanceLog{
		UserID:      userID,
		ChangeType:  "checkin",
		Amount:      amount,
		BalanceBefore: balanceBefore,
		BalanceAfter:  user.Balance,
		Description: sql.NullString{String: "每日签到奖励", Valid: true},
		Operator:    sql.NullString{String: "system", Valid: true},
	}
	if err := tx.Create(&balanceLog).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "签到失败", nil)
		return
	}

	tx.Commit()

	utils.SuccessResponse(c, http.StatusOK, "签到成功", gin.H{
		"amount":  amount,
		"balance": user.Balance,
	})
}

func GetCheckinStatus(c *gin.Context) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	userID := userIDVal.(uint)

	db := database.GetDB()
	now := utils.GetBeijingTime()
	dayStart, dayEnd := utils.GetDayRange(now)

	var count int64
	db.Model(&models.CheckinRecord{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).Count(&count)

	var totalCheckins int64
	db.Model(&models.CheckinRecord{}).Where("user_id = ?", userID).Count(&totalCheckins)

	var totalReward float64
	db.Model(&models.CheckinRecord{}).Where("user_id = ?", userID).Select("COALESCE(SUM(amount), 0)").Scan(&totalReward)

	// 连续签到天数
	streak := 0
	checkDate := now
	for {
		ds, de := utils.GetDayRange(checkDate)
		var c int64
		db.Model(&models.CheckinRecord{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, ds, de).Count(&c)
		if c == 0 {
			break
		}
		streak++
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"checked_in":    count > 0,
		"total_checkins": totalCheckins,
		"total_reward":   totalReward,
		"streak":         streak,
	})
}
