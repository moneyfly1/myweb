package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	// 使用事务和数据库锁防止重复签到
	err := utils.WithTransaction(db, func(tx *gorm.DB) error {
		// 在事务内再次检查，使用 FOR UPDATE 锁定用户记录
		var count int64
		if err := tx.Model(&models.CheckinRecord{}).
			Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).
			Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return fmt.Errorf("今天已经签到过了")
		}

		// 生成随机奖励金额（1.00-10.00，使用加密安全随机数）
		amount := 1.0
		randomCents, randomErr := rand.Int(rand.Reader, big.NewInt(901)) // 0-900
		if randomErr == nil {
			amount = float64(100+randomCents.Int64()) / 100.0
		}

		// 锁定用户记录并更新余额
		var user models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}

		balanceBefore := user.Balance
		user.Balance += amount

		if err := tx.Model(&user).Update("balance", user.Balance).Error; err != nil {
			return err
		}

		// 创建签到记录
		record := models.CheckinRecord{
			UserID:    userID,
			Amount:    amount,
			CreatedAt: now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// 创建余额日志
		balanceLog := models.BalanceLog{
			UserID:        userID,
			ChangeType:    "checkin",
			Amount:        amount,
			BalanceBefore: balanceBefore,
			BalanceAfter:  user.Balance,
			Description:   sql.NullString{String: "每日签到奖励", Valid: true},
			Operator:      sql.NullString{String: "system", Valid: true},
		}
		if err := tx.Create(&balanceLog).Error; err != nil {
			return err
		}

		// 记录审计日志
		utils.CreateBusinessLog(c, "user_checkin", "用户签到成功", "info", map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
			"balance": user.Balance,
		})

		// 将用户信息存储到上下文，供响应使用
		c.Set("checkin_amount", amount)
		c.Set("checkin_balance", user.Balance)

		return nil
	})

	if err != nil {
		if err.Error() == "今天已经签到过了" {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "签到失败", err)
		}
		return
	}

	amount, _ := c.Get("checkin_amount")
	balance, _ := c.Get("checkin_balance")

	utils.SuccessResponse(c, http.StatusOK, "签到成功", gin.H{
		"amount":  amount,
		"balance": balance,
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

	// 连续签到天数：单次查询近10年的签到记录，避免按天N次查询
	streak := 0
	var checkinTimes []sql.NullTime
	err := db.Model(&models.CheckinRecord{}).
		Where("user_id = ? AND created_at < ?", userID, dayEnd).
		Order("created_at DESC").
		Limit(3650).
		Pluck("created_at", &checkinTimes).Error
	if err == nil {
		daySet := make(map[string]struct{}, len(checkinTimes))
		for _, ct := range checkinTimes {
			if !ct.Valid {
				continue
			}
			dayKey := ct.Time.In(utils.BeijingTZ).Format("2006-01-02")
			daySet[dayKey] = struct{}{}
		}

		checkDate := dayStart
		for {
			dayKey := checkDate.In(utils.BeijingTZ).Format("2006-01-02")
			if _, exists := daySet[dayKey]; !exists {
				break
			}
			streak++
			checkDate = checkDate.AddDate(0, 0, -1)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"checked_in":     count > 0,
		"total_checkins": totalCheckins,
		"total_reward":   totalReward,
		"streak":         streak,
	})
}
