package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/middleware"
	"cboard-go/internal/models"
	discountService "cboard-go/internal/services/discount"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/services/payment"
	"cboard-go/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func resolveUserInfo(user models.User) gin.H {
	if user.ID > 0 {
		return gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		}
	}
	return gin.H{
		"id":       0,
		"username": "已删除",
		"email":    "deleted",
	}
}

func resolvePackageInfo(order models.Order) (uint, string, interface{}) {
	if order.PackageID == 0 {
		name := "设备升级"
		itemType := "device_upgrade"
		if order.ExtraData.Valid && order.ExtraData.String != "" {
			var extraData map[string]interface{}
			if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
				if orderType, ok := extraData["type"].(string); ok && orderType == "custom_package" {
					devices := 0
					months := 0
					if devicesVal, ok := extraData["devices"].(float64); ok {
						devices = int(devicesVal)
					}
					if monthsVal, ok := extraData["months"].(float64); ok {
						months = int(monthsVal)
					}
					name = fmt.Sprintf("自定义套餐 (%d设备/%d月)", devices, months)
					itemType = "custom_package"
					return 0, name, gin.H{"id": 0, "name": name, "type": itemType}
				}
				if orderType, ok := extraData["type"].(string); ok && orderType == "device_upgrade" {
					oldLimit := 0
					newLimit := 0
					if v, ok := extraData["old_device_limit"].(float64); ok {
						oldLimit = int(v)
					}
					if v, ok := extraData["new_device_limit"].(float64); ok {
						newLimit = int(v)
					}
					if oldLimit > 0 && newLimit > 0 {
						name = fmt.Sprintf("设备升级 (%d→%d台)", oldLimit, newLimit)
					} else {
						name = "设备升级订单"
					}
					return 0, name, gin.H{"id": 0, "name": name, "type": itemType, "old_device_limit": oldLimit, "new_device_limit": newLimit}
				}
			}
			name = "设备升级订单"
		}
		return 0, name, gin.H{"id": 0, "name": name, "type": itemType}
	}

	if order.Package.ID > 0 && order.Package.ID == order.PackageID {
		return order.PackageID, order.Package.Name, gin.H{
			"id":           order.Package.ID,
			"name":         order.Package.Name,
			"price":        order.Package.Price,
			"device_limit": order.Package.DeviceLimit,
		}
	}

	// 如果Package未预加载，返回基本 info
	return order.PackageID, "套餐未加载", gin.H{"id": order.PackageID, "name": "套餐未加载"}
}

func formatOrderData(order models.Order) gin.H {
	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

	var balanceUsed float64
	var parsedExtraData map[string]interface{}
	if order.ExtraData.Valid && order.ExtraData.String != "" {
		if err := json.Unmarshal([]byte(order.ExtraData.String), &parsedExtraData); err == nil {
			if balanceUsedVal, ok := parsedExtraData["balance_used"].(float64); ok {
				balanceUsed = balanceUsedVal
			}
		}
	}
	if balanceUsed > 0 {
		amount = utils.RoundFloat(amount+balanceUsed, 2)
	}

	paymentMethod := ""
	if order.PaymentMethodName.Valid {
		paymentMethod = order.PaymentMethodName.String
	}

	paymentTime := ""
	if order.PaymentTime.Valid {
		paymentTime = utils.FormatBeijingTime(order.PaymentTime.Time)
	}

	userInfo := resolveUserInfo(order.User)
	pkgID, pkgName, pkgInfo := resolvePackageInfo(order)

	return gin.H{
		"id":                     order.ID,
		"order_no":               order.OrderNo,
		"user_id":                order.UserID,
		"user":                   userInfo,
		"package_id":             pkgID,
		"package_name":           pkgName,
		"package":                pkgInfo,
		"amount":                 amount,
		"final_amount":           utils.GetNullFloat64Value(order.FinalAmount),
		"discount_amount":        utils.GetNullFloat64Value(order.DiscountAmount),
		"payment_method":         paymentMethod,
		"payment_method_id":      utils.GetNullInt64Value(order.PaymentMethodID),
		"payment_time":           paymentTime,
		"payment_transaction_id": utils.GetNullStringValue(order.PaymentTransactionID),
		"status":                 order.Status,
		"created_at":             utils.FormatBeijingTime(order.CreatedAt),
		"updated_at":             utils.FormatBeijingTime(order.UpdatedAt),
		"expire_time":            utils.GetNullTimeValue(order.ExpireTime),
		"coupon_id":              utils.GetNullInt64Value(order.CouponID),
		"extra_data":             parsedExtraData,
	}
}

type adminMergedOrderRef struct {
	RecordType string
	ID         uint
	CreatedAt  time.Time
}

func buildAdminMergedOrderWhere(keyword string, status string) (string, string, []interface{}, []interface{}) {
	orderWhere := "1 = 1"
	rechargeWhere := "1 = 1"
	orderArgs := make([]interface{}, 0, 5)
	rechargeArgs := make([]interface{}, 0, 5)

	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			orderWhere += " AND (orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?)"
			rechargeWhere += " AND (recharge_records.order_no LIKE ? OR recharge_records.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?)"
			orderArgs = append(orderArgs, "%"+sanitizedKeyword+"%", "%ORD%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
			rechargeArgs = append(rechargeArgs, "%"+sanitizedKeyword+"%", "%RCH%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}

	if status != "" && status != "all" {
		orderWhere += " AND orders.status = ?"
		rechargeWhere += " AND recharge_records.status = ?"
		orderArgs = append(orderArgs, status)
		rechargeArgs = append(rechargeArgs, status)
	}

	return orderWhere, rechargeWhere, orderArgs, rechargeArgs
}

func scanAdminMergedOrderRefs(db *gorm.DB, keyword string, status string, offset int, size int) ([]adminMergedOrderRef, int64, error) {
	orderWhere, rechargeWhere, orderArgs, rechargeArgs := buildAdminMergedOrderWhere(keyword, status)

	var orderCount, rechargeCount int64
	orderCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders LEFT JOIN users ON orders.user_id = users.id WHERE %s", orderWhere)
	if err := db.Raw(orderCountQuery, orderArgs...).Scan(&orderCount).Error; err != nil {
		return nil, 0, err
	}
	rechargeCountQuery := fmt.Sprintf("SELECT COUNT(*) FROM recharge_records LEFT JOIN users ON recharge_records.user_id = users.id WHERE %s", rechargeWhere)
	if err := db.Raw(rechargeCountQuery, rechargeArgs...).Scan(&rechargeCount).Error; err != nil {
		return nil, 0, err
	}

	args := make([]interface{}, 0, len(orderArgs)+len(rechargeArgs)+2)
	args = append(args, orderArgs...)
	args = append(args, rechargeArgs...)
	args = append(args, size, offset)
	listQuery := fmt.Sprintf(`
		SELECT record_type, id, created_at
		FROM (
			SELECT 'order' AS record_type, orders.id AS id, orders.created_at AS created_at
			FROM orders
			LEFT JOIN users ON orders.user_id = users.id
			WHERE %s
			UNION ALL
			SELECT 'recharge' AS record_type, recharge_records.id AS id, recharge_records.created_at AS created_at
			FROM recharge_records
			LEFT JOIN users ON recharge_records.user_id = users.id
			WHERE %s
		) merged_records
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, orderWhere, rechargeWhere)

	var refs []adminMergedOrderRef
	if err := db.Raw(listQuery, args...).Scan(&refs).Error; err != nil {
		return nil, 0, err
	}

	return refs, orderCount + rechargeCount, nil
}

func extractYipayPaymentType(payType string, orderPaymentMethodName string) string {
	if strings.HasPrefix(payType, "yipay_") {
		return strings.TrimPrefix(payType, "yipay_")
	}

	if orderPaymentMethodName != "" {
		if strings.Contains(orderPaymentMethodName, "yipay_wxpay") || strings.Contains(orderPaymentMethodName, "易支付-微信") {
			return "wxpay"
		}
		if strings.Contains(orderPaymentMethodName, "yipay_alipay") || strings.Contains(orderPaymentMethodName, "易支付-支付宝") {
			return "alipay"
		}
		if strings.Contains(orderPaymentMethodName, "yipay_qqpay") || strings.Contains(orderPaymentMethodName, "易支付-QQ") {
			return "qqpay"
		}
		if strings.HasPrefix(orderPaymentMethodName, "yipay_") {
			parts := strings.Split(orderPaymentMethodName, "yipay_")
			if len(parts) > 1 {
				for _, part := range parts {
					if strings.HasPrefix(part, "wxpay") {
						return "wxpay"
					} else if strings.HasPrefix(part, "alipay") {
						return "alipay"
					} else if strings.HasPrefix(part, "qqpay") {
						return "qqpay"
					}
				}
			}
		}
	}
	return "alipay"
}

func extractCodepayPaymentType(payType string, reqMethod string) string {
	if reqMethod != "" && strings.HasPrefix(reqMethod, "codepay_") {
		return strings.TrimPrefix(reqMethod, "codepay_")
	}
	if strings.HasPrefix(payType, "codepay_") {
		return strings.TrimPrefix(payType, "codepay_")
	}
	return "alipay"
}

func generatePaymentURL(db *gorm.DB, order *models.Order, paymentConfig *models.PaymentConfig, reqMethod string, amount float64) (string, error) {
	markPendingTransactionsFailed := func(err error) (string, error) {
		if order != nil && order.ID > 0 {
			if updateErr := db.Model(&models.PaymentTransaction{}).
				Where("order_id = ? AND status = ?", order.ID, "pending").
				Update("status", "failed").Error; updateErr != nil {
				utils.LogError("generatePaymentURL: mark order payment transactions failed", updateErr, map[string]interface{}{
					"order_no": order.OrderNo,
					"order_id": order.ID,
				})
			}
		} else if order != nil && order.OrderNo != "" {
			if updateErr := db.Model(&models.PaymentTransaction{}).
				Where("order_id = ? AND transaction_id = ? AND status = ?", 0, order.OrderNo, "pending").
				Update("status", "failed").Error; updateErr != nil {
				utils.LogError("generatePaymentURL: mark recharge payment transactions failed", updateErr, map[string]interface{}{
					"order_no": order.OrderNo,
				})
			}
		}
		return "", err
	}

	switch paymentConfig.PayType {
	case "alipay":
		service, err := payment.NewAlipayService(paymentConfig)
		if err != nil {
			return markPendingTransactionsFailed(err)
		}
		url, err := service.CreatePayment(order, amount)
		if err != nil {
			return markPendingTransactionsFailed(err)
		}
		return url, nil
	case "wechat":
		service, err := payment.NewWechatService(paymentConfig)
		if err != nil {
			return markPendingTransactionsFailed(err)
		}
		url, err := service.CreatePayment(order, amount)
		if err != nil {
			return markPendingTransactionsFailed(err)
		}
		return url, nil
	default:
		if paymentConfig.PayType == "yipay" || strings.HasPrefix(paymentConfig.PayType, "yipay_") {
			service, err := payment.NewYipayService(paymentConfig)
			if err != nil {
				return markPendingTransactionsFailed(err)
			}
			pType := extractYipayPaymentType(paymentConfig.PayType, "")
			if reqMethod != "" && strings.HasPrefix(reqMethod, "yipay_") {
				pType = strings.TrimPrefix(reqMethod, "yipay_")
			} else if order.PaymentMethodName.Valid {
				pType = extractYipayPaymentType(paymentConfig.PayType, order.PaymentMethodName.String)
			}
			url, err := service.CreatePayment(order, amount, pType)
			if err != nil {
				return markPendingTransactionsFailed(err)
			}
			return url, nil
		}
		if paymentConfig.PayType == "codepay" || strings.HasPrefix(paymentConfig.PayType, "codepay_") {
			service, err := payment.NewCodepayService(paymentConfig)
			if err != nil {
				return markPendingTransactionsFailed(err)
			}
			pType := extractCodepayPaymentType(paymentConfig.PayType, reqMethod)
			url, err := service.CreatePayment(order, amount, pType)
			if err != nil {
				return markPendingTransactionsFailed(err)
			}
			return url, nil
		}
		return markPendingTransactionsFailed(fmt.Errorf("不支持的支付方式: %s", paymentConfig.PayType))
	}
}

func shouldQueryPaymentStatus(createdAt time.Time) bool {
	timeSince := time.Since(createdAt)
	if timeSince >= 3*time.Second && timeSince < 10*time.Second {
		return true
	} else if timeSince >= 10*time.Second && timeSince < 60*time.Second {
		return int(timeSince.Seconds())%5 < 2
	} else if timeSince >= 60*time.Second {
		return int(timeSince.Seconds())%30 < 2
	}
	return false
}

func performAlipayQuery(db *gorm.DB, orderNo string, isRecharge bool) (bool, *payment.AlipayQueryResult, error) {
	configModel, err := paymentConfigForOrderStatusQuery(db, orderNo, "alipay")
	if err != nil {
		return false, nil, err
	}
	if !strings.EqualFold(configModel.PayType, "alipay") {
		return false, nil, nil
	}

	service, err := payment.NewAlipayService(configModel)
	if err != nil {
		return false, nil, err
	}

	result, err := service.QueryOrder(orderNo)
	if err != nil || result == nil || !result.IsPaid() {
		return false, result, err
	}

	// 验证订单号匹配（防止订单号碰撞）
	if result.OutTradeNo != orderNo {
		utils.LogError("performAlipayQuery: order number mismatch", nil, map[string]interface{}{
			"expected_order_no": orderNo,
			"alipay_order_no":   result.OutTradeNo,
			"trade_no":          result.TradeNo,
		})
		return false, result, fmt.Errorf("订单号不匹配")
	}

	params := map[string]string{
		"out_trade_no":   orderNo,
		"trade_no":       result.TradeNo,
		"total_amount":   result.TotalAmount,
		"trade_status":   result.TradeStatus,
		"query_fallback": "true",
	}

	err = finalizeStatusQueriedPayment(db, orderNo, isRecharge, "alipay", configModel.ID, result.TradeNo, params)

	return err == nil, result, err
}

func performPaymentStatusQuery(db *gorm.DB, orderNo string, isRecharge bool) (bool, error) {
	configModel, err := paymentConfigForOrderStatusQuery(db, orderNo, "")
	if err != nil {
		return false, err
	}
	if configModel == nil {
		return false, nil
	}
	paymentType := statusQueryPaymentType(db, orderNo, isRecharge, configModel.PayType)
	configPayType := strings.ToLower(strings.TrimSpace(configModel.PayType))

	switch {
	case configPayType == "alipay":
		success, _, err := performAlipayQuery(db, orderNo, isRecharge)
		return success, err
	case configPayType == "wechat":
		return performWechatPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	case configPayType == "yipay" || strings.HasPrefix(configPayType, "yipay_"):
		return performYipayPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	case configPayType == "codepay" || strings.HasPrefix(configPayType, "codepay_"):
		return performCodepayPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	case paymentType == "alipay":
		success, _, err := performAlipayQuery(db, orderNo, isRecharge)
		return success, err
	case paymentType == "wechat":
		return performWechatPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	case strings.HasPrefix(paymentType, "yipay"):
		return performYipayPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	case strings.HasPrefix(paymentType, "codepay"):
		return performCodepayPaymentQuery(db, orderNo, isRecharge, configModel, paymentType)
	default:
		return false, nil
	}
}

func statusQueryPaymentType(db *gorm.DB, orderNo string, isRecharge bool, fallback string) string {
	if isRecharge {
		var recharge models.RechargeRecord
		if err := db.Where("order_no = ?", orderNo).First(&recharge).Error; err == nil && recharge.PaymentMethod.Valid {
			return normalizeStatusPaymentType(recharge.PaymentMethod.String, fallback)
		}
		return normalizeStatusPaymentType(fallback, fallback)
	}

	var order models.Order
	if err := db.Where("order_no = ?", orderNo).First(&order).Error; err == nil && order.PaymentMethodName.Valid {
		return normalizeStatusPaymentType(order.PaymentMethodName.String, fallback)
	}
	return normalizeStatusPaymentType(fallback, fallback)
}

func normalizeStatusPaymentType(method string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(method))
	fallback = strings.ToLower(strings.TrimSpace(fallback))
	if normalized == "" {
		return fallback
	}
	if strings.HasPrefix(normalized, "yipay_") || normalized == "yipay" ||
		strings.HasPrefix(normalized, "codepay_") || normalized == "codepay" ||
		normalized == "alipay" || normalized == "wechat" {
		return normalized
	}

	if strings.Contains(normalized, "易支付") || strings.Contains(normalized, "yipay") {
		if strings.Contains(normalized, "微信") || strings.Contains(normalized, "wxpay") {
			return "yipay_wxpay"
		}
		if strings.Contains(normalized, "qq") {
			return "yipay_qqpay"
		}
		return "yipay_alipay"
	}
	if strings.Contains(normalized, "码支付") || strings.Contains(normalized, "codepay") {
		if strings.Contains(normalized, "微信") || strings.Contains(normalized, "wxpay") {
			return "codepay_wxpay"
		}
		return "codepay_alipay"
	}
	if strings.Contains(normalized, "微信") {
		return "wechat"
	}
	if strings.Contains(normalized, "支付宝") {
		return "alipay"
	}
	return fallback
}

func performWechatPaymentQuery(db *gorm.DB, orderNo string, isRecharge bool, configModel *models.PaymentConfig, paymentType string) (bool, error) {
	service, err := payment.NewWechatService(configModel)
	if err != nil {
		return false, err
	}
	result, err := service.QueryOrder(orderNo)
	if err != nil || result == nil || !result.IsPaid() {
		return false, err
	}
	if result.OutTradeNo != "" && result.OutTradeNo != orderNo {
		return false, fmt.Errorf("微信支付订单号不匹配: expected=%s, actual=%s", orderNo, result.OutTradeNo)
	}

	params := copyPaymentParams(result.Raw)
	params["out_trade_no"] = orderNo
	params["transaction_id"] = result.TransactionID
	params["total_fee"] = result.TotalFee
	params["return_code"] = result.ReturnCode
	params["result_code"] = result.ResultCode
	params["trade_state"] = result.TradeState
	params["query_fallback"] = "true"
	if paymentType == "" {
		paymentType = "wechat"
	}

	err = finalizeStatusQueriedPayment(db, orderNo, isRecharge, paymentType, configModel.ID, result.TransactionID, params)
	return err == nil, err
}

func performYipayPaymentQuery(db *gorm.DB, orderNo string, isRecharge bool, configModel *models.PaymentConfig, paymentType string) (bool, error) {
	service, err := payment.NewYipayService(configModel)
	if err != nil {
		return false, err
	}
	result, err := service.QueryOrder(orderNo)
	if err != nil || result == nil || !result.IsPaid() {
		return false, err
	}
	if result.OutTradeNo != "" && result.OutTradeNo != orderNo {
		return false, fmt.Errorf("易支付订单号不匹配: expected=%s, actual=%s", orderNo, result.OutTradeNo)
	}
	if paymentType == "" || !strings.HasPrefix(paymentType, "yipay") {
		paymentType = "yipay"
	}

	params := epayQueryParams(orderNo, result)
	err = finalizeStatusQueriedPayment(db, orderNo, isRecharge, paymentType, configModel.ID, result.TradeNo, params)
	return err == nil, err
}

func performCodepayPaymentQuery(db *gorm.DB, orderNo string, isRecharge bool, configModel *models.PaymentConfig, paymentType string) (bool, error) {
	service, err := payment.NewCodepayService(configModel)
	if err != nil {
		return false, err
	}
	result, err := service.QueryOrder(orderNo)
	if err != nil || result == nil || !result.IsPaid() {
		return false, err
	}
	if result.OutTradeNo != "" && result.OutTradeNo != orderNo {
		return false, fmt.Errorf("码支付订单号不匹配: expected=%s, actual=%s", orderNo, result.OutTradeNo)
	}
	if paymentType == "" || !strings.HasPrefix(paymentType, "codepay") {
		paymentType = "codepay"
	}

	params := epayQueryParams(orderNo, result)
	err = finalizeStatusQueriedPayment(db, orderNo, isRecharge, paymentType, configModel.ID, result.TradeNo, params)
	return err == nil, err
}

func copyPaymentParams(raw map[string]string) map[string]string {
	params := make(map[string]string, len(raw)+4)
	for k, v := range raw {
		params[k] = v
	}
	return params
}

func epayQueryParams(orderNo string, result *payment.EpayQueryResult) map[string]string {
	params := copyPaymentParams(result.Raw)
	params["out_trade_no"] = orderNo
	if result.TradeNo != "" {
		params["trade_no"] = result.TradeNo
	}
	if result.Amount != "" {
		params["money"] = result.Amount
	}
	if result.Status != "" {
		params["trade_status"] = result.Status
		params["status"] = result.Status
	}
	params["query_fallback"] = "true"
	return params
}

func finalizeStatusQueriedPayment(db *gorm.DB, orderNo string, isRecharge bool, paymentType string, paymentConfigID uint, externalTransactionID string, params map[string]string) error {
	if isRecharge {
		_, err := processPaidRecharge(db, orderNo, paymentType, paymentConfigID, externalTransactionID, params, "system-query")
		return err
	}

	var order models.Order
	if err := db.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return err
	}
	if callbackAmount, ok := parseCallbackAmount(paymentType, params); ok {
		expectedAmount := order.Amount
		if order.FinalAmount.Valid {
			expectedAmount = order.FinalAmount.Float64
		}
		if !amountMatches(expectedAmount, callbackAmount) {
			return fmt.Errorf("订单金额不匹配: 预期支付%.2f元, 支付渠道返回%.2f元", expectedAmount, callbackAmount)
		}
	} else {
		utils.LogWarn("performPaymentStatusQuery: unable to verify order amount for payment type: %+v", map[string]interface{}{
			"order_no":     orderNo,
			"payment_type": paymentType,
		})
	}

	callbackData := ""
	if data, err := json.Marshal(params); err == nil {
		callbackData = string(data)
	}
	_, err := orderServicePkg.NewOrderService().FinalizePaidOrder(orderNo, orderServicePkg.FinalizePaidOrderOptions{
		PaymentMethodName:     paymentType,
		PaymentMethodID:       paymentConfigID,
		ExternalTransactionID: externalTransactionID,
		CallbackData:          callbackData,
		IPAddress:             "system-query",
	})
	if err == nil && order.Status == "pending" {
		sendPaymentNotifications(db, orderNo)
	}
	return err
}

func paymentConfigForOrderStatusQuery(db *gorm.DB, orderNo string, fallbackPayType string) (*models.PaymentConfig, error) {
	var order models.Order
	if err := db.Where("order_no = ?", orderNo).First(&order).Error; err == nil {
		if order.PaymentMethodID.Valid && order.PaymentMethodID.Int64 > 0 {
			var paymentConfig models.PaymentConfig
			if err := db.First(&paymentConfig, order.PaymentMethodID.Int64).Error; err == nil && paymentConfig.Status == 1 {
				return &paymentConfig, nil
			}
		}
		if order.PaymentMethodName.Valid && order.PaymentMethodName.String != "" {
			if paymentConfig, err := utils.FindEnabledPaymentConfig(db, order.PaymentMethodName.String); err == nil {
				return &paymentConfig, nil
			}
		}
	}

	var recharge models.RechargeRecord
	if err := db.Where("order_no = ?", orderNo).First(&recharge).Error; err == nil {
		if recharge.PaymentMethod.Valid && recharge.PaymentMethod.String != "" {
			if paymentConfig, err := utils.FindEnabledPaymentConfig(db, recharge.PaymentMethod.String); err == nil {
				return &paymentConfig, nil
			}
		}
		var paymentTx models.PaymentTransaction
		if err := db.Where("order_id = ? AND transaction_id = ?", 0, orderNo).
			Order("created_at DESC").First(&paymentTx).Error; err == nil {
			var paymentConfig models.PaymentConfig
			if err := db.First(&paymentConfig, paymentTx.PaymentMethodID).Error; err == nil && paymentConfig.Status == 1 {
				return &paymentConfig, nil
			}
		}
	}

	if strings.TrimSpace(fallbackPayType) == "" {
		return nil, nil
	}

	paymentConfig, err := utils.FindEnabledPaymentConfig(db, fallbackPayType)
	if err != nil {
		return nil, err
	}
	return &paymentConfig, nil
}

type CreateOrderRequest struct {
	PackageID      uint    `json:"package_id" binding:"required"`
	CouponCode     string  `json:"coupon_code"`
	PaymentMethod  string  `json:"payment_method"`
	Amount         float64 `json:"amount"`
	UseBalance     bool    `json:"use_balance"`
	Currency       string  `json:"currency"`
	DurationMonths int     `json:"duration_months"`
}

func CreateOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req orderServicePkg.CreateOrderParams
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	req.UserAgent = c.GetHeader("User-Agent")
	req.ClientIP = utils.GetRealClientIP(c)

	svc := orderServicePkg.NewOrderService()
	order, paymentURL, err := svc.CreateOrder(user.ID, req)
	if err != nil {
		if strings.Contains(err.Error(), "生成支付链接失败") {
			orderNo := ""
			if order != nil {
				orderNo = order.OrderNo
			}
			utils.CreateBusinessLog(c, "order_payment_url_failed", "创建订单生成支付链接失败", "error", map[string]interface{}{
				"user_id": user.ID, "order_no": orderNo, "reason": err.Error(),
			})
		}
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	data := gin.H{
		"order_no":            order.OrderNo,
		"id":                  order.ID,
		"user_id":             order.UserID,
		"package_id":          order.PackageID,
		"amount":              order.Amount,
		"final_amount":        utils.GetNullFloat64Value(order.FinalAmount),
		"discount_amount":     utils.GetNullFloat64Value(order.DiscountAmount),
		"status":              order.Status,
		"payment_method":      utils.GetNullStringValue(order.PaymentMethodName),
		"payment_method_name": utils.GetNullStringValue(order.PaymentMethodName),
		"created_at":          utils.FormatBeijingTime(order.CreatedAt),
	}

	if order.PaymentMethodName.Valid {
		data["payment_method_name"] = order.PaymentMethodName.String
	}
	if paymentURL != "" {
		data["payment_url"] = paymentURL
		data["payment_qr_code"] = paymentURL
	}
	if order.Status == "paid" {
		data["message"] = "订单已支付成功"
	}
	if order.CouponID.Valid {
		data["coupon_id"] = order.CouponID.Int64
	}

	utils.SuccessResponse(c, http.StatusCreated, "订单创建成功", data)
}

func GetOrders(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	page, size, _ := getPagination(c)
	status := c.Query("status")
	paymentMethod := c.Query("payment_method")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	isAdmin, exists := c.Get("is_admin")
	admin := exists && isAdmin.(bool)

	db := database.GetDB()
	var orders []models.Order
	var total int64

	query := db.Model(&models.Order{}).Preload("User").Preload("Package").Preload("Coupon")

	if !admin {
		query = query.Where("user_id = ?", user.ID)
	}

	if status != "" && status != "all" {
		statusMap := map[string]string{
			"pending":   "pending",
			"paid":      "paid",
			"cancelled": "cancelled",
			"expired":   "expired",
			"待支付":       "pending",
			"已支付":       "paid",
			"已取消":       "cancelled",
			"已过期":       "expired",
		}
		if mappedStatus, ok := statusMap[status]; ok {
			query = query.Where("status = ?", mappedStatus)
		} else {
			query = query.Where("status = ?", status)
		}
	}

	if paymentMethod != "" && paymentMethod != "all" {
		query = query.Where("payment_method_name = ?", paymentMethod)
	}
	var startParsed time.Time
	if startDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "start_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		startParsed = parsed
		query = query.Where("created_at >= ?", startParsed)
	}
	if endDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endDate, utils.BeijingTZ)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date格式错误，应为YYYY-MM-DD", err)
			return
		}
		if !startParsed.IsZero() && parsed.Before(startParsed) {
			utils.ErrorResponse(c, http.StatusBadRequest, "end_date不能早于start_date", nil)
			return
		}
		endTime := parsed.Add(24 * time.Hour)
		query = query.Where("created_at < ?", endTime)
	}

	query.Count(&total)

	offset := (page - 1) * size
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&orders).Error; err != nil {
		utils.LogError("GetOrders: query orders", err, nil)
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败，请稍后重试", err)
		return
	}

	pages := (int(total) + size - 1) / size
	formattedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		formattedOrders[i] = formatOrderData(order)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"orders": formattedOrders,
		"total":  total,
		"page":   page,
		"size":   size,
		"pages":  pages,
	})
}

func GetOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	isAdmin, _ := c.Get("is_admin")
	admin := isAdmin.(bool)

	db := database.GetDB()
	var order models.Order
	query := db.Preload("Package").Preload("Coupon").Where("id = ?", id)

	if !admin {
		query = query.Where("user_id = ?", user.ID)
	}

	if err := query.First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", order)
}

func CancelOrder(c *gin.Context) {
	id := c.Param("id")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("id = ? AND user_id = ?", id, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		return
	}

	cancelledOrder, err := orderServicePkg.NewOrderService().CancelPendingOrder(order.OrderNo, user.ID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已取消", cancelledOrder)
}

func CancelOrderByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	order, err := orderServicePkg.NewOrderService().CancelPendingOrder(orderNo, user.ID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "订单已取消", order)
}

func GetAdminOrders(c *gin.Context) {
	db := database.GetDB()
	includeRecharges := c.Query("include_recharges") == "true"
	page, size, _ := getPagination(c)
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	status := c.Query("status")

	if includeRecharges {
		offset := (page - 1) * size
		refs, total, err := scanAdminMergedOrderRefs(db, keyword, status, offset, size)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
			return
		}

		orderIDs := make([]uint, 0, len(refs))
		rechargeIDs := make([]uint, 0, len(refs))
		for _, ref := range refs {
			if ref.RecordType == "order" {
				orderIDs = append(orderIDs, ref.ID)
			} else if ref.RecordType == "recharge" {
				rechargeIDs = append(rechargeIDs, ref.ID)
			}
		}

		orderMap := make(map[uint]gin.H, len(orderIDs))
		if len(orderIDs) > 0 {
			var orders []models.Order
			if err := db.Preload("User").Preload("Package").Where("id IN ?", orderIDs).Find(&orders).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
				return
			}
			for _, order := range orders {
				formatted := formatOrderData(order)
				formatted["record_type"] = "order"
				orderMap[order.ID] = formatted
			}
		}

		rechargeMap := make(map[uint]gin.H, len(rechargeIDs))
		if len(rechargeIDs) > 0 {
			var recharges []models.RechargeRecord
			if err := db.Preload("User").Where("id IN ?", rechargeIDs).Find(&recharges).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
				return
			}
			for _, record := range recharges {
				formatted := gin.H{
					"id":                     record.ID,
					"user_id":                record.UserID,
					"user":                   resolveUserInfo(record.User),
					"order_no":               record.OrderNo,
					"amount":                 record.Amount,
					"status":                 record.Status,
					"payment_method":         utils.GetNullStringValue(record.PaymentMethod),
					"payment_transaction_id": utils.GetNullStringValue(record.PaymentTransactionID),
					"paid_at":                utils.GetNullTimeValue(record.PaidAt),
					"created_at":             utils.FormatBeijingTime(record.CreatedAt),
					"record_type":            "recharge",
				}
				rechargeMap[record.ID] = formatted
			}
		}

		mergedList := make([]gin.H, 0, len(refs))
		for _, ref := range refs {
			if ref.RecordType == "order" {
				if record, ok := orderMap[ref.ID]; ok {
					mergedList = append(mergedList, record)
				}
				continue
			}
			if record, ok := rechargeMap[ref.ID]; ok {
				mergedList = append(mergedList, record)
			}
		}

		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"orders": mergedList,
			"total":  total,
			"page":   page,
			"size":   size,
		})
		return
	}

	var orders []models.Order
	query := db.Model(&models.Order{}).Joins("LEFT JOIN users ON orders.user_id = users.id")

	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where(
				"orders.order_no LIKE ? OR orders.order_no LIKE ? OR users.username LIKE ? OR users.email LIKE ?",
				"%"+sanitizedKeyword+"%", "%ORD%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%",
			)
		}
	}

	if status != "" && status != "all" {
		query = query.Where("orders.status = ?", status)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * size
	if err := query.Preload("User").Preload("Package").Offset(offset).Limit(size).Order("orders.created_at DESC").Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
		return
	}

	orderList := make([]gin.H, len(orders))
	for i, order := range orders {
		orderList[i] = formatOrderData(order)
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"orders": orderList,
		"total":  total,
		"page":   page,
		"size":   size,
	})
}

func UpdateAdminOrder(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误，请检查输入格式", err)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	if req.Status == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单状态不能为空", nil)
		return
	}

	oldStatus := order.Status

	if order.Status != "paid" && req.Status == "paid" {
		svc := orderServicePkg.NewOrderService()
		if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "admin_manual",
			IPAddress:         utils.GetRealClientIP(c),
		}); err != nil {
			utils.LogError("UpdateAdminOrder: finalize paid order", err, map[string]interface{}{"order_id": order.ID})
			utils.ErrorResponse(c, http.StatusInternalServerError, "处理订单失败，请稍后重试", nil)
			return
		}
		if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
			return
		}
	} else if order.Status == "pending" && (req.Status == "cancelled" || req.Status == "expired") {
		updatedOrder, err := orderServicePkg.NewOrderService().MarkPendingOrderStatus(order.OrderNo, 0, req.Status)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		if err := db.Preload("Package").Preload("User").First(&order, updatedOrder.ID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
			return
		}
	} else {
		if order.Status == "paid" && req.Status != "paid" {
			utils.ErrorResponse(c, http.StatusBadRequest, "已支付订单请使用退款流程，不能直接改状态", nil)
			return
		}
		if order.Status != req.Status {
			utils.ErrorResponse(c, http.StatusBadRequest, "当前订单状态不允许直接修改", nil)
			return
		}
		order.Status = req.Status
		if err := db.Save(&order).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "更新订单失败", err)
			return
		}
	}

	utils.CreateAuditLog(c, "update_admin_order", "order", order.ID,
		fmt.Sprintf("管理员更新订单 %s(%s, ¥%.2f) 状态 → %s", order.OrderNo, order.User.Email, order.Amount, order.Status),
		map[string]interface{}{"order_no": order.OrderNo, "old_status": oldStatus, "amount": order.Amount, "user_id": order.UserID},
		map[string]interface{}{"order_no": order.OrderNo, "status": order.Status, "amount": order.Amount, "user_id": order.UserID})
	utils.SuccessResponse(c, http.StatusOK, "订单已更新", order)
}

func RefundAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var order models.Order
	if err := db.Preload("Package").Preload("User").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单失败", err)
		}
		return
	}

	// 只能退款已支付的订单
	if order.Status != "paid" {
		utils.ErrorResponse(c, http.StatusBadRequest, "只能退款已支付的订单", nil)
		return
	}

	// 检查支付方式，只有易支付订单才能退款
	paymentMethodName := ""
	if order.PaymentMethodName.Valid {
		paymentMethodName = order.PaymentMethodName.String
	}

	isYipay := strings.Contains(paymentMethodName, "易支付") || strings.Contains(paymentMethodName, "yipay")
	if !isYipay && order.PaymentMethodID.Valid {
		var paymentConfig models.PaymentConfig
		if err := db.First(&paymentConfig, order.PaymentMethodID.Int64).Error; err == nil {
			if paymentConfig.PayType == "yipay" || strings.HasPrefix(paymentConfig.PayType, "yipay_") {
				isYipay = true
			}
		}
	}

	if !isYipay {
		utils.ErrorResponse(c, http.StatusBadRequest, "只有易支付订单才能退款", nil)
		return
	}

	// 获取支付配置
	var paymentConfig models.PaymentConfig
	if order.PaymentMethodID.Valid {
		if err := db.First(&paymentConfig, order.PaymentMethodID.Int64).Error; err != nil {
			// 如果找不到支付配置，尝试查找易支付配置
			if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "未找到易支付配置", nil)
				return
			}
		}
	} else {
		// 如果没有支付方式ID，尝试查找易支付配置
		if err := db.Where("LOWER(pay_type) = LOWER(?) AND status = ?", "yipay", 1).First(&paymentConfig).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "未找到易支付配置", nil)
			return
		}
	}

	// 创建易支付服务
	yipayService, err := payment.NewYipayService(&paymentConfig)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "初始化易支付服务失败: "+err.Error(), nil)
		return
	}

	// 计算退款金额
	refundAmount := order.Amount
	if order.FinalAmount.Valid {
		refundAmount = order.FinalAmount.Float64
	}

	// 获取易支付交易号
	tradeNo := ""
	if order.PaymentTransactionID.Valid {
		tradeNo = order.PaymentTransactionID.String
	}

	// 调用易支付退款API
	utils.LogInfo("RefundAdminOrder: 开始退款 - order_id=%d, order_no=%s, trade_no=%s, refund_amount=%.2f", order.ID, order.OrderNo, tradeNo, refundAmount)
	if err := yipayService.RefundOrder(order.OrderNo, tradeNo, refundAmount); err != nil {
		utils.LogError("RefundAdminOrder: 易支付退款失败", err, map[string]interface{}{
			"order_id":      order.ID,
			"order_no":      order.OrderNo,
			"trade_no":      tradeNo,
			"refund_amount": refundAmount,
		})
		utils.CreateBusinessLog(c, "refund_failed", "管理员退款失败（易支付接口）", "error", map[string]interface{}{
			"order_id": order.ID, "order_no": order.OrderNo, "trade_no": tradeNo, "refund_amount": refundAmount, "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "退款失败: "+err.Error(), nil)
		return
	}

	// 退款成功，处理订单回退逻辑
	orderService := orderServicePkg.NewOrderService()
	if err := orderService.ProcessRefundOrder(&order); err != nil {
		utils.LogError("RefundAdminOrder: 处理退款订单失败", err, map[string]interface{}{
			"order_id": order.ID,
		})
		utils.CreateBusinessLog(c, "refund_process_failed", "管理员退款后订单回退处理失败", "error", map[string]interface{}{
			"order_id": order.ID, "order_no": order.OrderNo, "reason": err.Error(),
		})
		utils.ErrorResponse(c, http.StatusInternalServerError, "处理退款订单失败: "+err.Error(), nil)
		return
	}

	utils.LogInfo("RefundAdminOrder: 订单退款成功 - order_id=%d, order_no=%s, refund_amount=%.2f", order.ID, order.OrderNo, refundAmount)
	utils.CreateAuditLog(c, "refund_admin_order", "order", order.ID,
		fmt.Sprintf("管理员退款订单 %s(%s, ¥%.2f → 退款 ¥%.2f)", order.OrderNo, order.User.Email, order.Amount, refundAmount),
		map[string]interface{}{
			"order_no": order.OrderNo, "user_id": order.UserID, "user_email": order.User.Email,
			"amount": order.Amount, "status": "paid", "payment_method": order.PaymentMethodName.String,
		},
		map[string]interface{}{
			"order_no": order.OrderNo, "refund_amount": refundAmount, "status": "refunded",
		})
	utils.SuccessResponse(c, http.StatusOK, "订单退款成功", order)
}

func DeleteAdminOrder(c *gin.Context) {
	id := c.Param("id")
	db := database.GetDB()
	var order models.Order
	if err := db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "查询订单失败", err)
		return
	}

	if _, err := orderServicePkg.NewOrderService().DeleteOrders([]uint{order.ID}); err != nil {
		utils.LogError("DeleteOrder: delete order failed", err, map[string]interface{}{"order_id": order.ID})
		utils.ErrorResponse(c, http.StatusInternalServerError, "删除订单失败", err)
		return
	}
	utils.CreateAuditLog(c, "delete_admin_order", "order", order.ID,
		fmt.Sprintf("管理员删除订单 %s(%s, ¥%.2f)", order.OrderNo, order.User.Email, order.Amount),
		map[string]interface{}{"order_no": order.OrderNo, "user_id": order.UserID, "amount": order.Amount, "status": order.Status}, nil)
	utils.SuccessResponse(c, http.StatusOK, "订单已删除", nil)
}

func GetOrderStatistics(c *gin.Context) {
	db := database.GetDB()
	dayStart, dayEnd := utils.GetDayRange(utils.GetBeijingTime())
	paymentSummary := utils.CalculatePaymentSummary(db, dayStart, dayEnd)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total_orders":   paymentSummary.Total,
		"pending_orders": paymentSummary.Pending,
		"paid_orders":    paymentSummary.Paid,
		"cancelled":      paymentSummary.Cancelled,
		"total_revenue":  paymentSummary.PaidRevenue,
	})
}

func BulkMarkOrdersPaid(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	svc := orderServicePkg.NewOrderService()
	successCount := 0
	failCount := 0

	// 批量查出所有待处理订单
	var orders []models.Order
	db.Where("id IN ? AND status = ?", req.OrderIDs, "pending").Find(&orders)
	orderMap := make(map[uint]*models.Order)
	for i := range orders {
		orderMap[orders[i].ID] = &orders[i]
	}

	for _, orderID := range req.OrderIDs {
		order, ok := orderMap[orderID]
		if !ok {
			failCount++
			continue
		}
		if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "admin_manual",
			IPAddress:         utils.GetRealClientIP(c),
		}); err != nil {
			utils.LogError("BulkMarkOrdersPaid: process order failed", err, map[string]interface{}{"order_id": orderID})
			failCount++
		} else {
			successCount++
			go sendPaymentNotifications(db, order.OrderNo)
		}
	}
	utils.CreateAuditLog(c, "bulk_mark_orders_paid", "order", 0,
		fmt.Sprintf("管理员批量标记订单已支付 成功%d 失败%d", successCount, failCount),
		map[string]interface{}{"order_ids": req.OrderIDs, "total": len(req.OrderIDs)}, nil)
	utils.SuccessResponse(c, http.StatusOK, fmt.Sprintf("处理完成: 成功 %d, 失败 %d", successCount, failCount), nil)
}

func BulkCancelOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	cancelledCount, err := orderServicePkg.NewOrderService().CancelPendingOrders(req.OrderIDs)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量取消失败", err)
		return
	}
	utils.CreateAuditLog(c, "bulk_cancel_orders", "order", 0,
		fmt.Sprintf("管理员批量取消 %d 个订单", cancelledCount),
		map[string]interface{}{"order_ids": req.OrderIDs, "total": len(req.OrderIDs), "cancelled": cancelledCount}, nil)
	utils.SuccessResponse(c, http.StatusOK, "批量取消成功", gin.H{"cancelled": cancelledCount})
}

func BatchDeleteOrders(c *gin.Context) {
	var req struct {
		OrderIDs []uint `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}
	deletedCount, err := orderServicePkg.NewOrderService().DeleteOrders(req.OrderIDs)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "批量删除失败", err)
		return
	}
	utils.CreateAuditLog(c, "batch_delete_orders", "order", 0,
		fmt.Sprintf("管理员批量删除 %d 个订单", deletedCount),
		map[string]interface{}{"order_ids": req.OrderIDs, "total": len(req.OrderIDs), "deleted": deletedCount}, nil)
	utils.SuccessResponse(c, http.StatusOK, "批量删除成功", gin.H{"deleted": deletedCount})
}

func ExportOrders(c *gin.Context) {
	db := database.GetDB()
	query := db.Model(&models.Order{})
	keyword := c.Query("keyword")
	if keyword == "" {
		keyword = c.Query("search")
	}
	if keyword != "" {
		sanitizedKeyword := utils.SanitizeSearchKeyword(keyword)
		if sanitizedKeyword != "" {
			query = query.Where("order_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ? OR email LIKE ?)",
				"%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%", "%"+sanitizedKeyword+"%")
		}
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []models.Order
	if err := query.Order("created_at DESC").Find(&orders).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "获取订单列表失败", err)
		return
	}

	var csvContent strings.Builder
	csvContent.WriteString("\xEF\xBB\xBF")
	csvContent.WriteString("订单号,用户ID,用户名,邮箱,套餐ID,套餐名称,订单金额,支付方式,订单状态,创建时间,支付时间,更新时间\n")

	for _, order := range orders {
		formatted := formatOrderData(order)
		userMap := formatted["user"].(gin.H)
		statusText := order.Status
		switch order.Status {
		case "pending":
			statusText = "待支付"
		case "paid":
			statusText = "已支付"
		case "cancelled":
			statusText = "已取消"
		}

		csvContent.WriteString(fmt.Sprintf("%s,%d,%s,%s,%d,%s,%.2f,%s,%s,%s,%s,%s\n",
			order.OrderNo,
			order.UserID,
			userMap["username"],
			userMap["email"],
			formatted["package_id"],
			formatted["package_name"],
			formatted["amount"],
			formatted["payment_method"],
			statusText,
			utils.FormatBeijingTime(order.CreatedAt),
			formatted["payment_time"],
			utils.FormatBeijingTime(order.UpdatedAt),
		))
	}

	filename := fmt.Sprintf("orders_export_%s.csv", utils.GetBeijingTime().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csvContent.String()))
}

func GetOrderStats(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	db := database.GetDB()
	summary := utils.CalculateUserPaymentSummary(db, user.ID)

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"total":       summary.Total,
		"pending":     summary.Pending,
		"paid":        summary.Paid,
		"cancelled":   summary.Cancelled,
		"totalAmount": summary.PaidAmount,
		"paidAmount":  summary.PaidAmount,
	})
}

func GetOrderStatusByNo(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}
	db := database.GetDB()

	if strings.HasPrefix(orderNo, "RCH") {
		var recharge models.RechargeRecord
		if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&recharge).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "充值记录不存在", err)
			return
		}
		if recharge.Status == "pending" && shouldQueryPaymentStatus(recharge.CreatedAt) {
			if success, err := performPaymentStatusQuery(db, orderNo, true); success {
				db.Where("order_no = ?", orderNo).First(&recharge)
			} else if err != nil {
				utils.LogWarn("GetOrderStatusByNo: active recharge payment query failed: %+v", map[string]interface{}{
					"order_no": orderNo,
					"error":    err.Error(),
				})
			}
		}
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"order_no": recharge.OrderNo,
			"status":   recharge.Status,
			"amount":   recharge.Amount,
			"type":     "recharge",
		})
		return
	}

	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", err)
		return
	}

	if order.Status == "pending" && shouldQueryPaymentStatus(order.CreatedAt) {
		if success, err := performPaymentStatusQuery(db, orderNo, false); success {
			db.Where("order_no = ?", orderNo).First(&order)
		} else if err != nil {
			utils.LogWarn("GetOrderStatusByNo: active order payment query failed: %+v", map[string]interface{}{
				"order_no": orderNo,
				"error":    err.Error(),
			})
		}
	}
	if order.Status == "pending" && order.ExpireTime.Valid && utils.GetBeijingTime().After(order.ExpireTime.Time) {
		if expiredOrder, err := orderServicePkg.NewOrderService().MarkPendingOrderStatus(orderNo, user.ID, "expired"); err == nil && expiredOrder != nil {
			order = *expiredOrder
		}
	} else if order.Status == "paid" && !order.FulfilledAt.Valid {
		if _, err := orderServicePkg.NewOrderService().FinalizePaidOrder(orderNo, orderServicePkg.FinalizePaidOrderOptions{
			IPAddress: utils.GetRealClientIP(c),
		}); err == nil {
			db.Where("order_no = ?", orderNo).First(&order)
		}
	}

	orderType := "order"
	if order.PackageID == 0 {
		if order.ExtraData.Valid && order.ExtraData.String != "" {
			var extraData map[string]interface{}
			if err := json.Unmarshal([]byte(order.ExtraData.String), &extraData); err == nil {
				if extraType, ok := extraData["type"].(string); ok && extraType != "" {
					orderType = extraType
				}
			}
		}
	}
	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"order_no":     order.OrderNo,
		"status":       order.Status,
		"amount":       order.Amount,
		"final_amount": utils.GetNullFloat64Value(order.FinalAmount),
		"type":         orderType,
	})
}

// calcDeviceUpgradeAmount 按剩余时间与可选续期计算设备升级费用（年度价格法）
// 规则：年度价 200 对应 baseDevices 台；仅加设备时按剩余天数比例；若同时加时长则 = 原设备续期费 + 新增设备在(剩余+续期)内的费用
func calcDeviceUpgradeAmount(expireTime time.Time, currentDeviceLimit, additionalDevices, additionalDays int, yearlyPrice float64, baseDevices int) float64 {
	if yearlyPrice <= 0 || baseDevices <= 0 {
		return 0
	}
	now := utils.GetBeijingTime()
	remainingDays := expireTime.Sub(now).Hours() / 24
	if remainingDays < 0 {
		remainingDays = 0
	}

	var cost float64
	if additionalDays > 0 {
		// 原设备续期费用 = 年度价 × (续期天数/365) × (当前设备数/基准设备数)
		cost += yearlyPrice * (float64(additionalDays) / 365) * (float64(currentDeviceLimit) / float64(baseDevices))
		// 新增设备在（剩余+续期）内的费用 = 年度价 × ((剩余+续期)天数/365) × (新增设备数/基准设备数)
		cost += yearlyPrice * ((remainingDays + float64(additionalDays)) / 365) * (float64(additionalDevices) / float64(baseDevices))
	} else {
		// 仅增加设备：按剩余时间比例计费
		if remainingDays <= 0 {
			return 0 // 已过期且未续期，无法仅加设备
		}
		cost = yearlyPrice * (remainingDays / 365) * (float64(additionalDevices) / float64(baseDevices))
	}
	return cost
}

func UpgradeDevices(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		AdditionalDevices int    `json:"additional_devices" binding:"required,min=1"`
		AdditionalDays    int    `json:"additional_days"`
		PaymentMethod     string `json:"payment_method"`
		UseBalance        bool   `json:"use_balance"`
		PreviewOnly       bool   `json:"preview_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var subscription models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订阅不存在", err)
		return
	}

	yearlyPrice := config.AppConfig.DeviceUpgradePricePerYear
	baseDevices := config.AppConfig.DeviceUpgradeBaseDevices
	if baseDevices <= 0 {
		baseDevices = 5
	}

	var totalAmount float64
	if yearlyPrice > 0 && baseDevices > 0 {
		totalAmount = calcDeviceUpgradeAmount(
			subscription.ExpireTime,
			subscription.DeviceLimit,
			req.AdditionalDevices,
			req.AdditionalDays,
			yearlyPrice,
			baseDevices,
		)
		// 已过期且未选续期时不允许仅加设备
		if totalAmount == 0 && req.AdditionalDays <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期，请同时选择增加时长后再升级设备数量", nil)
			return
		}
	}
	// 未配置年度价时回退到按月的旧逻辑
	if totalAmount == 0 {
		now := utils.GetBeijingTime()
		if subscription.ExpireTime.Before(now) && req.AdditionalDays <= 0 {
			utils.ErrorResponse(c, http.StatusBadRequest, "订阅已过期，请同时选择增加时长后再升级设备数量", nil)
			return
		}
		devicePricePerMonth := config.AppConfig.DeviceUpgradePricePerMonth
		if devicePricePerMonth <= 0 {
			devicePricePerMonth = 10.0
		}
		totalAmount = (float64(req.AdditionalDevices) * devicePricePerMonth) + (float64(req.AdditionalDays) * (devicePricePerMonth / 30.0))
	}
	totalAmount = utils.RoundFloat(totalAmount, 2)

	var userLevel models.UserLevel
	levelDiscount := 1.0
	levelDiscountAmount := 0.0
	payableAmount := totalAmount
	if user.UserLevelID.Valid {
		if err := db.First(&userLevel, user.UserLevelID.Int64).Error; err == nil && userLevel.DiscountRate > 0 && userLevel.DiscountRate < 1.0 {
			levelDiscount = userLevel.DiscountRate
			levelDiscountAmount = utils.RoundFloat(totalAmount*(1-userLevel.DiscountRate), 2)
			payableAmount = utils.RoundFloat(totalAmount-levelDiscountAmount, 2)
			if payableAmount < 0 {
				payableAmount = 0
			}
		}
	}

	// 仅预览：返回费用不创建订单
	if req.PreviewOnly {
		utils.SuccessResponse(c, http.StatusOK, "", gin.H{
			"upgrade_cost":          totalAmount,
			"level_discount":        levelDiscountAmount,
			"level_discount_rate":   levelDiscount,
			"amount":                totalAmount,
			"final_amount":          payableAmount,
			"actual_payment_amount": payableAmount,
		})
		return
	}

	balanceUsed := 0.0
	finalAmount := payableAmount
	if req.UseBalance && user.Balance > 0 {
		availableBalance := math.Round(user.Balance*100) / 100
		if availableBalance > finalAmount {
			availableBalance = finalAmount
		}
		if availableBalance > 0 {
			balanceUsed = availableBalance
			finalAmount -= balanceUsed
		}
	}

	orderNo, err := utils.GenerateDeviceUpgradeOrderNo(db)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成订单号失败", err)
		return
	}
	balanceDeductedStr := "false"
	if balanceUsed > 0 {
		balanceDeductedStr = "true"
	}
	newDeviceLimit := subscription.DeviceLimit + req.AdditionalDevices
	oldExpireTime := subscription.ExpireTime.Format(TimeLayout)
	newExpireTime := subscription.ExpireTime.AddDate(0, 0, req.AdditionalDays).Format(TimeLayout)
	extraData := fmt.Sprintf(`{"type":"device_upgrade","additional_devices":%d,"additional_days":%d,"old_device_limit":%d,"new_device_limit":%d,"old_expire_time":"%s","new_expire_time":"%s","balance_used":%.2f,"balance_deducted":%s,"level_discount":%.2f,"level_discount_rate":%.4f,"payable_amount":%.2f}`, req.AdditionalDevices, req.AdditionalDays, subscription.DeviceLimit, newDeviceLimit, oldExpireTime, newExpireTime, balanceUsed, balanceDeductedStr, levelDiscountAmount, levelDiscount, payableAmount)

	order := models.Order{
		OrderNo:           orderNo,
		UserID:            user.ID,
		PackageID:         0,
		Amount:            totalAmount,
		FinalAmount:       database.NullFloat64(finalAmount),
		DiscountAmount:    database.NullFloat64(levelDiscountAmount),
		Status:            "pending",
		ExtraData:         database.NullString(extraData),
		PaymentMethodName: database.NullString(req.PaymentMethod),
	}

	if balanceUsed > 0 {
		if finalAmount > 0.01 {
			order.PaymentMethodName = database.NullString(fmt.Sprintf("余额支付(%.2f元)+%s", balanceUsed, req.PaymentMethod))
		} else {
			order.PaymentMethodName = database.NullString("余额支付")
		}
	}

	var paymentConfig models.PaymentConfig
	var paymentURL string

	// 余额支付但余额不足时直接提示，避免创建无法支付的死订单
	if req.PaymentMethod == "balance" && finalAmount > 0.01 {
		utils.ErrorResponse(c, http.StatusBadRequest, "余额不足，请选择其他支付方式或充值后再试", nil)
		return
	}

	// 使用事务保证余额扣除与订单创建的一致性
	txErr := db.Transaction(func(tx *gorm.DB) error {
		// 查找支付配置（需外部支付时）
		if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
			var err error
			paymentConfig, err = utils.FindEnabledPaymentConfig(tx, req.PaymentMethod)
			if err != nil {
				return fmt.Errorf("未找到启用的支付配置: %v", err)
			}
			order.PaymentMethodID = database.NullInt64(utils.MustSafeUintToInt64(paymentConfig.ID))
		}

		// 创建订单
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %v", err)
		}

		// 扣除余额
		if balanceUsed > 0 {
			oldBalance := user.Balance
			result := tx.Model(&models.User{}).Where("id = ? AND balance >= ?", user.ID, balanceUsed).
				Update("balance", gorm.Expr("balance - ?", balanceUsed))
			if result.Error != nil {
				return fmt.Errorf("扣除余额失败: %v", result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("余额不足")
			}
			orderID := uint(order.ID)
			userID := user.ID
			if err := utils.CreateBalanceLogWithDB(
				tx, user.ID, "consume", -balanceUsed,
				oldBalance, oldBalance-balanceUsed,
				&orderID, nil,
				fmt.Sprintf("设备升级订单余额抵扣，订单号: %s", orderNo),
				"user", &userID, utils.GetRealClientIP(c),
			); err != nil {
				return fmt.Errorf("记录余额日志失败: %v", err)
			}
		}

		// 创建支付交易记录（外部支付）
		if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
			transaction := models.PaymentTransaction{
				OrderID:         order.ID,
				UserID:          user.ID,
				PaymentMethodID: paymentConfig.ID,
				Amount:          int(math.Round(finalAmount * 100)),
				Currency:        "CNY",
				Status:          "pending",
			}
			if err := tx.Create(&transaction).Error; err != nil {
				return fmt.Errorf("创建支付记录失败: %v", err)
			}
		}

		return nil
	})

	if txErr != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, txErr.Error(), txErr)
		return
	}

	// 生成支付链接（在事务外，因为涉及外部API调用）
	if finalAmount > 0.01 && req.PaymentMethod != "" && req.PaymentMethod != "balance" {
		var err error
		paymentURL, err = generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod, finalAmount)
		if err != nil {
			_, _ = orderServicePkg.NewOrderService().MarkPendingOrderStatus(order.OrderNo, user.ID, "failed")
			utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付链接失败", err)
			return
		}
	}

	if finalAmount <= 0.01 {
		processedSubscription, err := orderServicePkg.NewOrderService().FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "余额支付",
			IPAddress:         utils.GetRealClientIP(c),
		})
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		db.Where("user_id = ?", user.ID).First(&subscription)
		if processedSubscription != nil {
			subscription = *processedSubscription
		}
		utils.SuccessResponse(c, http.StatusOK, "设备数量升级成功", gin.H{
			"order_no":           order.OrderNo,
			"status":             "paid",
			"subscription":       subscription,
			"additional_devices": req.AdditionalDevices,
			"additional_days":    req.AdditionalDays,
		})
		return
	}

	var extraDataMap map[string]interface{}
	_ = json.Unmarshal([]byte(order.ExtraData.String), &extraDataMap) // Ignore error, use empty if invalid
	if extraDataMap == nil {
		extraDataMap = make(map[string]interface{})
	}
	extraDataMap["type"] = "device_upgrade"
	extraDataMap["additional_devices"] = req.AdditionalDevices
	extraDataMap["additional_days"] = req.AdditionalDays
	if balanceUsed > 0 {
		extraDataMap["balance_used"] = balanceUsed
	}
	if levelDiscountAmount > 0 {
		extraDataMap["level_discount"] = levelDiscountAmount
		extraDataMap["level_discount_rate"] = levelDiscount
	}
	extraDataMap["payable_amount"] = payableAmount
	extraDataBytes, _ := json.Marshal(extraDataMap)
	order.ExtraData = database.NullString(string(extraDataBytes))
	if err := db.Save(&order).Error; err != nil {
		log.Printf("failed to update order extra data: %v", err)
	}

	actualTotalAmount := utils.RoundFloat(balanceUsed+finalAmount, 2)
	responseData := gin.H{
		"order_no":              order.OrderNo,
		"id":                    order.ID,
		"status":                order.Status,
		"amount":                totalAmount,
		"final_amount":          finalAmount,
		"actual_payment_amount": finalAmount,
		"actual_total_amount":   actualTotalAmount,
		"level_discount":        levelDiscountAmount,
		"level_discount_rate":   levelDiscount,
		"payable_amount":        payableAmount,
		"balance_used":          balanceUsed,
		"additional_devices":    req.AdditionalDevices,
		"additional_days":       req.AdditionalDays,
	}
	if paymentURL != "" {
		responseData["payment_url"] = paymentURL
		responseData["payment_qr_code"] = paymentURL
	}
	utils.SuccessResponse(c, http.StatusOK, "", responseData)
}

func PayOrder(c *gin.Context) {
	orderNo := c.Param("orderNo")
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		PaymentMethodID uint   `json:"payment_method_id"`
		PaymentMethod   string `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	db := database.GetDB()
	var order models.Order
	if err := db.Where("order_no = ? AND user_id = ?", orderNo, user.ID).First(&order).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "订单不存在", nil)
		return
	}
	if order.Status != "pending" {
		utils.ErrorResponse(c, http.StatusBadRequest, "订单状态不允许支付", nil)
		return
	}

	amount := order.Amount
	if order.FinalAmount.Valid {
		amount = order.FinalAmount.Float64
	}

	// 余额支付
	if req.PaymentMethod == "balance" {
		var freshUser models.User
		if err := db.First(&freshUser, user.ID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "用户不存在", err)
			return
		}
		if freshUser.Balance+0.001 < amount {
			utils.ErrorResponse(c, http.StatusBadRequest, "余额不足，无法完成支付", nil)
			return
		}
		if _, err := orderServicePkg.NewOrderService().FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "余额支付",
			IPAddress:         utils.GetRealClientIP(c),
			BalanceAmount:     amount,
		}); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		utils.SuccessResponse(c, http.StatusOK, "支付成功", gin.H{
			"status":   "paid",
			"order_no": order.OrderNo,
			"amount":   amount,
		})
		return
	}

	var paymentConfig models.PaymentConfig
	if req.PaymentMethodID > 0 {
		if err := db.First(&paymentConfig, req.PaymentMethodID).Error; err != nil || paymentConfig.Status != 1 {
			utils.ErrorResponse(c, http.StatusBadRequest, "支付方式无效", nil)
			return
		}
	} else if req.PaymentMethod != "" {
		var err error
		paymentConfig, err = utils.FindEnabledPaymentConfig(db, req.PaymentMethod)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "支付方式无效", nil)
			return
		}
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "请选择支付方式", nil)
		return
	}

	transaction := models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: paymentConfig.ID,
		Amount:          int(math.Round(amount * 100)),
		Currency:        "CNY",
		Status:          "pending",
	}
	if err := db.Create(&transaction).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付交易失败", err)
		return
	}
	if req.PaymentMethod != "" {
		order.PaymentMethodName = database.NullString(req.PaymentMethod)
	}
	order.PaymentMethodID = database.NullInt64(utils.MustSafeUintToInt64(paymentConfig.ID))
	if err := db.Model(&order).Updates(map[string]interface{}{
		"payment_method_id":   order.PaymentMethodID,
		"payment_method_name": order.PaymentMethodName,
	}).Error; err != nil {
		utils.LogError("PayOrder: update order payment method id failed", err, map[string]interface{}{"order_no": order.OrderNo})
	}

	paymentURL, err := generatePaymentURL(db, &order, &paymentConfig, req.PaymentMethod, amount)
	if err != nil {
		if failedOrder, markErr := orderServicePkg.NewOrderService().MarkPendingOrderStatus(order.OrderNo, user.ID, "failed"); markErr != nil {
			utils.LogError("PayOrder: mark order failed after payment link error", markErr, map[string]interface{}{
				"order_no": order.OrderNo,
			})
		} else if failedOrder != nil {
			order = *failedOrder
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "创建支付失败", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "支付订单已创建", gin.H{
		"payment_url":           paymentURL,
		"payment_qr_code":       paymentURL,
		"order_no":              order.OrderNo,
		"amount":                amount,
		"final_amount":          amount,
		"actual_payment_amount": amount,
		"payment_method":        req.PaymentMethod,
		"transaction_id":        transaction.ID,
	})
}

// CreateCustomOrder 创建自定义套餐订单
func CreateCustomOrder(c *gin.Context) {
	user, ok := middleware.GetCurrentUser(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "未登录", nil)
		return
	}

	var req struct {
		Devices    int    `json:"devices" binding:"required,min=1"`
		Months     int    `json:"months" binding:"required,min=1"`
		CouponCode string `json:"coupon_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "请求参数错误", err)
		return
	}

	// 检查自定义套餐是否启用
	db := database.GetDB()
	var enabledConfig models.SystemConfig
	if err := db.Where("`key` = ? AND category = ?", "custom_package_enabled", "custom_package").First(&enabledConfig).Error; err != nil || enabledConfig.Value != "true" {
		utils.ErrorResponse(c, http.StatusBadRequest, "自定义套餐功能未启用", nil)
		return
	}

	// 获取配置
	var configs []models.SystemConfig
	db.Where("category = ?", "custom_package").Find(&configs)
	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}

	pricePerDeviceYear := utils.ParseFloat(configMap["custom_package_price_per_device_year"], 40.0)
	minDevices := utils.ParseInt(configMap["custom_package_min_devices"], 5)
	maxDevices := utils.ParseInt(configMap["custom_package_max_devices"], 100)
	minMonths := utils.ParseInt(configMap["custom_package_min_months"], 6)

	// 验证参数
	if req.Devices < minDevices || req.Devices > maxDevices {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("设备数量需在 %d ~ %d 之间", minDevices, maxDevices), nil)
		return
	}
	if req.Months < minMonths {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("最少购买 %d 个月", minMonths), nil)
		return
	}
	if req.Months > 120 {
		utils.ErrorResponse(c, http.StatusBadRequest, "最多购买 120 个月", nil)
		return
	}

	// 计算价格
	basePrice := pricePerDeviceYear * float64(req.Devices) * (float64(req.Months) / 12.0)
	basePrice = utils.RoundFloat(basePrice, 2)

	// 解析时长折扣
	var discountTiers []struct {
		Months   int     `json:"months"`
		Discount float64 `json:"discount"`
	}
	if discountsJSON := configMap["custom_package_duration_discounts"]; discountsJSON != "" {
		_ = json.Unmarshal([]byte(discountsJSON), &discountTiers) // Ignore error, use empty if invalid
	}

	// 找到最佳折扣
	var discountPercent float64
	for _, tier := range discountTiers {
		if req.Months >= tier.Months && tier.Discount > discountPercent {
			discountPercent = tier.Discount
		}
	}

	finalPrice := basePrice * (1 - discountPercent/100)
	finalPrice = utils.RoundFloat(finalPrice, 2)

	// 应用优惠券
	var couponDiscount float64
	var coupon *models.Coupon
	couponFreeDays := 0
	if strings.TrimSpace(req.CouponCode) != "" {
		quote, err := discountService.QuoteCouponForPreparedAmount(db, req.CouponCode, user.ID, 0, finalPrice)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		couponDiscount = quote.CouponDiscountAmount
		couponFreeDays = quote.FreeDays
		coupon = quote.Coupon
		finalPrice = quote.FinalAmount
	}
	if finalPrice <= 0.01 {
		finalPrice = 0
	}

	// 构建额外数据
	extraData := map[string]interface{}{
		"type":             "custom_package",
		"devices":          req.Devices,
		"months":           req.Months,
		"discount_percent": discountPercent,
	}
	if couponFreeDays > 0 {
		extraData["coupon_free_days"] = couponFreeDays
	}
	extraDataJSON, _ := json.Marshal(extraData)
	extraStr := string(extraDataJSON)

	// 创建订单号
	orderNo, err := utils.GenerateOrderNo(db)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "生成订单号失败", err)
		return
	}
	expireTime := time.Now().Add(30 * time.Minute)
	totalDiscount := basePrice - finalPrice

	order := models.Order{
		OrderNo:        orderNo,
		UserID:         user.ID,
		PackageID:      0,
		Amount:         basePrice,
		Status:         "pending",
		ExpireTime:     database.NullTime(expireTime),
		DiscountAmount: database.NullFloat64(totalDiscount),
		FinalAmount:    database.NullFloat64(finalPrice),
		ExtraData:      database.NullString(extraStr),
	}
	if finalPrice == 0 {
		order.PaymentMethodName = database.NullString("优惠抵扣")
	}

	if coupon != nil {
		order.CouponID = database.NullInt64(utils.MustSafeUintToInt64(coupon.ID))
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %v", err)
		}
		if coupon != nil {
			if err := discountService.ReserveCouponUsageTx(tx, coupon.ID, user.ID, order.ID, couponDiscount); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// 创建审计日志
	pkgName := fmt.Sprintf("自定义套餐 (%d设备/%d月)", req.Devices, req.Months)
	utils.CreateAuditLogSimple(c, "create_custom_order", "order", order.ID, fmt.Sprintf("用户创建自定义套餐订单: %s, 金额: %.2f", pkgName, finalPrice))

	if finalPrice == 0 {
		if _, err := orderServicePkg.NewOrderService().FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
			PaymentMethodName: "优惠抵扣",
			IPAddress:         utils.GetRealClientIP(c),
		}); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "处理自定义套餐订单失败", err)
			return
		}
		if err := db.First(&order, order.ID).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "刷新订单状态失败", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusCreated, "订单创建成功", gin.H{
		"order_no":         order.OrderNo,
		"id":               order.ID,
		"user_id":          order.UserID,
		"package_id":       0,
		"package_name":     pkgName,
		"amount":           basePrice,
		"final_amount":     finalPrice,
		"discount_amount":  totalDiscount,
		"coupon_id":        utils.GetNullInt64Value(order.CouponID),
		"coupon_free_days": couponFreeDays,
		"status":           order.Status,
		"created_at":       utils.FormatBeijingTime(order.CreatedAt),
	})
}
