package handlers

import (
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	orderServicePkg "cboard-go/internal/services/order"
	"cboard-go/internal/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPaymentFlowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sqlite sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserLevel{},
		&models.InviteCode{},
		&models.InviteRelation{},
		&models.Subscription{},
		&models.Device{},
		&models.SubscriptionReset{},
		&models.Order{},
		&models.Package{},
		&models.PaymentTransaction{},
		&models.PaymentConfig{},
		&models.PaymentCallback{},
		&models.Coupon{},
		&models.CouponUsage{},
		&models.RegistrationLog{},
		&models.SubscriptionLog{},
		&models.BalanceLog{},
		&models.CommissionLog{},
		&models.SystemConfig{},
		&models.Promotion{},
		&models.PromotionParticipation{},
		&models.RechargeRecord{},
	); err != nil {
		t.Fatalf("migrate sqlite test db: %v", err)
	}
	database.DB = db
	return db
}

func performPaymentFlowRequest(method string, routePath string, body string, user models.User, handler gin.HandlerFunc, requestPath ...string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, routePath, func(c *gin.Context) {
		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("is_admin", user.IsAdmin)
		handler(c)
	})
	path := routePath
	if len(requestPath) > 0 && requestPath[0] != "" {
		path = requestPath[0]
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestCancelPendingOrderReleasesCouponUsage(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "coupon_cancel", Email: "coupon_cancel@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Coupon Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "CANCEL10",
		Name:           "Cancel Coupon",
		Type:           string(models.CouponTypeFixed),
		DiscountValue:  10,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		TotalQuantity:  database.NullInt64(1),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with coupon: %v", err)
	}
	var reserved models.Coupon
	db.First(&reserved, coupon.ID)
	if reserved.UsedQuantity != 1 {
		t.Fatalf("expected coupon usage reserved once, got %d", reserved.UsedQuantity)
	}

	if _, err := svc.CancelPendingOrder(order.OrderNo, user.ID); err != nil {
		t.Fatalf("cancel pending order: %v", err)
	}
	var released models.Coupon
	db.First(&released, coupon.ID)
	if released.UsedQuantity != 0 {
		t.Fatalf("expected coupon usage released, got %d", released.UsedQuantity)
	}
	var usageCount int64
	db.Model(&models.CouponUsage{}).Where("order_id = ?", order.ID).Count(&usageCount)
	if usageCount != 0 {
		t.Fatalf("expected coupon usage record deleted, got %d", usageCount)
	}
}

func TestFreeDaysCouponExtendsSubscription(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "coupon_free_days", Email: "coupon_free_days@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Free Days Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "FREEDAYS7",
		Name:           "Free Days Coupon",
		Type:           string(models.CouponTypeFreeDays),
		DiscountValue:  7,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with free-days coupon: %v", err)
	}
	beforePay := utils.GetBeijingTime()
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{}); err != nil {
		t.Fatalf("finalize order: %v", err)
	}

	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		t.Fatal(err)
	}
	days := int(sub.ExpireTime.Sub(beforePay).Hours() / 24)
	if days < 36 || days > 37 {
		t.Fatalf("expected about 37 days after free-days coupon, got %d expire=%v", days, sub.ExpireTime)
	}
}

func TestDeletePendingOrderReleasesPromotionReservation(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "promo_delete", Email: "promo_delete@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Promo Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	promo := models.Promotion{
		Name:          "Promo Discount",
		Type:          "member_day",
		DiscountType:  "fixed",
		DiscountValue: 10,
		StartTime:     time.Now().Add(-time.Hour),
		EndTime:       time.Now().Add(time.Hour),
		IsActive:      true,
	}
	if err := db.Create(&promo).Error; err != nil {
		t.Fatal(err)
	}
	participation := models.PromotionParticipation{
		PromotionID: promo.ID,
		UserID:      user.ID,
		RewardType:  "discount",
		RewardValue: 10,
		Status:      "pending",
		ExpireAt:    database.NullTime(time.Now().Add(time.Hour)),
	}
	if err := db.Create(&participation).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{PackageID: pkg.ID})
	if err != nil {
		t.Fatalf("create order with promotion: %v", err)
	}
	var reserved models.PromotionParticipation
	db.First(&reserved, participation.ID)
	if !reserved.OrderID.Valid || reserved.OrderID.Int64 != int64(order.ID) {
		t.Fatalf("expected promotion reserved by order, got %+v", reserved.OrderID)
	}

	if _, err := svc.DeleteOrders([]uint{order.ID}); err != nil {
		t.Fatalf("delete pending order: %v", err)
	}
	var released models.PromotionParticipation
	db.First(&released, participation.ID)
	if released.OrderID.Valid {
		t.Fatalf("expected promotion reservation released, got %+v", released.OrderID)
	}
	if released.Status != "pending" {
		t.Fatalf("expected promotion status pending, got %s", released.Status)
	}
}

func TestFailedPendingOrderReleasesDiscountReservationsAndTransaction(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "payment_failed", Email: "payment_failed@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Failed Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "FAIL10",
		Name:           "Failed Coupon",
		Type:           string(models.CouponTypeFixed),
		DiscountValue:  10,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		TotalQuantity:  database.NullInt64(1),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with coupon: %v", err)
	}
	if err := db.Create(&models.PaymentTransaction{
		OrderID:         order.ID,
		UserID:          user.ID,
		PaymentMethodID: 1,
		Amount:          9000,
		Status:          "pending",
	}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := svc.MarkPendingOrderStatus(order.OrderNo, user.ID, "failed"); err != nil {
		t.Fatalf("mark order failed: %v", err)
	}
	var released models.Coupon
	db.First(&released, coupon.ID)
	if released.UsedQuantity != 0 {
		t.Fatalf("expected coupon usage released, got %d", released.UsedQuantity)
	}
	var usageCount int64
	db.Model(&models.CouponUsage{}).Where("order_id = ?", order.ID).Count(&usageCount)
	if usageCount != 0 {
		t.Fatalf("expected coupon usage record deleted, got %d", usageCount)
	}
	var paymentTx models.PaymentTransaction
	if err := db.Where("order_id = ?", order.ID).First(&paymentTx).Error; err != nil {
		t.Fatal(err)
	}
	if paymentTx.Status != "failed" {
		t.Fatalf("expected payment transaction failed, got %s", paymentTx.Status)
	}
}

func TestPayOrderPaymentLinkFailureMarksOrderFailedAndReleasesCoupon(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_link_fail", Email: "pay_link_fail@example.com", Password: "x", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Pay Link Fail Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "PAYFAIL10",
		Name:           "Pay Fail Coupon",
		Type:           string(models.CouponTypeFixed),
		DiscountValue:  10,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		TotalQuantity:  database.NullInt64(1),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}
	paymentConfig := models.PaymentConfig{
		PayType: "yipay",
		AppID:   sql.NullString{String: "pid", Valid: true},
		Status:  1,
	}
	if err := db.Create(&paymentConfig).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with coupon: %v", err)
	}
	recorder := performPaymentFlowRequest(
		http.MethodPost,
		"/orders/:orderNo/pay",
		fmt.Sprintf(`{"payment_method_id":%d,"payment_method":"yipay"}`, paymentConfig.ID),
		user,
		PayOrder,
		"/orders/"+order.OrderNo+"/pay",
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected payment creation failure, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var freshOrder models.Order
	if err := db.First(&freshOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if freshOrder.Status != "failed" {
		t.Fatalf("expected order failed after payment link failure, got %s", freshOrder.Status)
	}
	var released models.Coupon
	db.First(&released, coupon.ID)
	if released.UsedQuantity != 0 {
		t.Fatalf("expected coupon usage released, got %d", released.UsedQuantity)
	}
	var usageCount int64
	db.Model(&models.CouponUsage{}).Where("order_id = ?", order.ID).Count(&usageCount)
	if usageCount != 0 {
		t.Fatalf("expected coupon usage record deleted, got %d", usageCount)
	}
	var pendingTxCount int64
	db.Model(&models.PaymentTransaction{}).Where("order_id = ? AND status = ?", order.ID, "pending").Count(&pendingTxCount)
	if pendingTxCount != 0 {
		t.Fatalf("expected no pending payment transactions, got %d", pendingTxCount)
	}
}

func TestCreatePaymentFailureMarksOrderFailedAndReleasesCoupon(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "legacy_payment_fail", Email: "legacy_payment_fail@example.com", Password: "x", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Legacy Payment Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "LEGACYFAIL10",
		Name:           "Legacy Fail Coupon",
		Type:           string(models.CouponTypeFixed),
		DiscountValue:  10,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		TotalQuantity:  database.NullInt64(1),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}
	paymentConfig := models.PaymentConfig{
		PayType: "codepay",
		AppID:   sql.NullString{String: "pid", Valid: true},
		Status:  1,
	}
	if err := db.Create(&paymentConfig).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with coupon: %v", err)
	}
	recorder := performPaymentFlowRequest(
		http.MethodPost,
		"/payment",
		fmt.Sprintf(`{"order_id":%d,"payment_method_id":%d}`, order.ID, paymentConfig.ID),
		user,
		CreatePayment,
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected payment creation failure, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	var freshOrder models.Order
	if err := db.First(&freshOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if freshOrder.Status != "failed" {
		t.Fatalf("expected order failed after legacy payment link failure, got %s", freshOrder.Status)
	}
	var released models.Coupon
	db.First(&released, coupon.ID)
	if released.UsedQuantity != 0 {
		t.Fatalf("expected coupon usage released, got %d", released.UsedQuantity)
	}
	var pendingTxCount int64
	db.Model(&models.PaymentTransaction{}).Where("order_id = ? AND status = ?", order.ID, "pending").Count(&pendingTxCount)
	if pendingTxCount != 0 {
		t.Fatalf("expected no pending payment transactions, got %d", pendingTxCount)
	}
}

func TestRefundOrderReleasesCouponUsageAndRollsBackFreeDays(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "coupon_refund", Email: "coupon_refund@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Refund Basic", Price: 100, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	coupon := models.Coupon{
		Code:           "REFUND7",
		Name:           "Refund Free Days",
		Type:           string(models.CouponTypeFreeDays),
		DiscountValue:  7,
		ValidFrom:      time.Now().Add(-time.Hour),
		ValidUntil:     time.Now().Add(time.Hour),
		MaxUsesPerUser: 1,
		Status:         "active",
	}
	if err := db.Create(&coupon).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	order, _, err := svc.CreateOrder(user.ID, orderServicePkg.CreateOrderParams{
		PackageID:  pkg.ID,
		CouponCode: coupon.Code,
	})
	if err != nil {
		t.Fatalf("create order with coupon: %v", err)
	}
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{}); err != nil {
		t.Fatalf("finalize order: %v", err)
	}
	var paidOrder models.Order
	if err := db.First(&paidOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ProcessRefundOrder(&paidOrder); err != nil {
		t.Fatalf("refund order: %v", err)
	}

	var released models.Coupon
	db.First(&released, coupon.ID)
	if released.UsedQuantity != 0 {
		t.Fatalf("expected coupon usage released after refund, got %d", released.UsedQuantity)
	}
	var usageCount int64
	db.Model(&models.CouponUsage{}).Where("order_id = ?", order.ID).Count(&usageCount)
	if usageCount != 0 {
		t.Fatalf("expected coupon usage record deleted after refund, got %d", usageCount)
	}
	var sub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&sub).Error; err != nil {
		t.Fatal(err)
	}
	if sub.Status != "expired" {
		t.Fatalf("expected subscription expired after refund rollback, got %s", sub.Status)
	}
}

func TestFinalizePaidOrderIsIdempotentForPackage(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_pkg", Email: "pay_pkg@example.com", Password: "x", Balance: 100}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Basic", Price: 30, DurationDays: 30, DeviceLimit: 3, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	order := models.Order{
		OrderNo:   "ORDTESTPKG001",
		UserID:    user.ID,
		PackageID: pkg.ID,
		Amount:    30,
		Status:    "pending",
		ExtraData: database.NullString(`{"balance_used":10,"duration_months":1}`),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{BalanceAmount: 10}); err != nil {
		t.Fatalf("first finalize: %v", err)
	}
	var firstSub models.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&firstSub).Error; err != nil {
		t.Fatal(err)
	}
	var firstUser models.User
	db.First(&firstUser, user.ID)

	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{BalanceAmount: 10}); err != nil {
		t.Fatalf("second finalize: %v", err)
	}
	var secondSub models.Subscription
	db.Where("user_id = ?", user.ID).First(&secondSub)
	var secondUser models.User
	db.First(&secondUser, user.ID)

	if secondUser.Balance != firstUser.Balance {
		t.Fatalf("balance changed on duplicate finalize: %.2f -> %.2f", firstUser.Balance, secondUser.Balance)
	}
	if !secondSub.ExpireTime.Equal(firstSub.ExpireTime) {
		t.Fatalf("subscription expiry changed on duplicate finalize: %v -> %v", firstSub.ExpireTime, secondSub.ExpireTime)
	}
	if secondSub.DeviceLimit != firstSub.DeviceLimit {
		t.Fatalf("device limit changed on duplicate finalize: %d -> %d", firstSub.DeviceLimit, secondSub.DeviceLimit)
	}
}

func TestFinalizePaidOrderIsIdempotentForDeviceUpgrade(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_upg", Email: "pay_upg@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	sub := models.Subscription{
		UserID:          user.ID,
		SubscriptionURL: "sub-upg",
		DeviceLimit:     3,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      time.Now().AddDate(0, 0, 30),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	order := models.Order{
		OrderNo:   "UPGTEST001",
		UserID:    user.ID,
		PackageID: 0,
		Amount:    20,
		Status:    "pending",
		ExtraData: database.NullString(`{"type":"device_upgrade","additional_devices":2,"additional_days":0}`),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{}); err != nil {
		t.Fatalf("first finalize: %v", err)
	}
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{}); err != nil {
		t.Fatalf("second finalize: %v", err)
	}

	var fresh models.Subscription
	db.First(&fresh, sub.ID)
	if fresh.DeviceLimit != 5 {
		t.Fatalf("expected device limit 5 after duplicate callbacks, got %d", fresh.DeviceLimit)
	}
}

func TestCustomPackageReplacesExistingSubscriptionDuration(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "custom_replace", Email: "custom_replace@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := utils.GetBeijingTime()
	oldExpire := now.AddDate(0, 0, 365)
	oldPackageID := int64(88)
	sub := models.Subscription{
		UserID:          user.ID,
		PackageID:       &oldPackageID,
		SubscriptionURL: "sub-custom-replace",
		DeviceLimit:     5,
		IsActive:        true,
		Status:          "active",
		ExpireTime:      oldExpire,
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	order := models.Order{
		OrderNo:     "ORDCUSTOMREPLACE001",
		UserID:      user.ID,
		PackageID:   0,
		Amount:      760,
		FinalAmount: database.NullFloat64(760),
		Status:      "pending",
		ExtraData:   database.NullString(`{"type":"custom_package","devices":19,"months":12}`),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	beforePay := utils.GetBeijingTime()
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{}); err != nil {
		t.Fatalf("finalize custom package: %v", err)
	}

	var freshSub models.Subscription
	if err := db.First(&freshSub, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	days := int(freshSub.ExpireTime.Sub(beforePay).Hours() / 24)
	if days < 359 || days > 360 {
		t.Fatalf("expected custom package to open about 360 days, got %d expire=%v old_expire=%v", days, freshSub.ExpireTime, oldExpire)
	}
	if freshSub.ExpireTime.After(oldExpire.AddDate(0, 0, 30)) {
		t.Fatalf("custom package appears to have stacked on existing time: expire=%v old_expire=%v", freshSub.ExpireTime, oldExpire)
	}
	if freshSub.DeviceLimit != 19 {
		t.Fatalf("expected device limit replaced with 19, got %d", freshSub.DeviceLimit)
	}
	if freshSub.PackageID != nil {
		t.Fatalf("expected custom package subscription package id nil, got %+v", freshSub.PackageID)
	}

	var paidOrder models.Order
	if err := db.First(&paidOrder, order.ID).Error; err != nil {
		t.Fatal(err)
	}
	var extra map[string]interface{}
	if err := json.Unmarshal([]byte(paidOrder.ExtraData.String), &extra); err != nil {
		t.Fatalf("parse extra data: %v", err)
	}
	if extra["activation_mode"] != "replace" {
		t.Fatalf("expected activation mode replace, got %+v", extra["activation_mode"])
	}

	if err := svc.ProcessRefundOrder(&paidOrder); err != nil {
		t.Fatalf("refund custom package: %v", err)
	}
	var restoredSub models.Subscription
	if err := db.First(&restoredSub, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !restoredSub.ExpireTime.Equal(oldExpire) {
		t.Fatalf("expected old expiry restored after refund: got %v want %v", restoredSub.ExpireTime, oldExpire)
	}
	if restoredSub.DeviceLimit != 5 {
		t.Fatalf("expected old device limit restored after refund, got %d", restoredSub.DeviceLimit)
	}
	if restoredSub.PackageID == nil || *restoredSub.PackageID != oldPackageID {
		t.Fatalf("expected old package id restored after refund, got %+v", restoredSub.PackageID)
	}
}

func TestPublicSettingsExposeEnabledCustomPackageWithoutNormalPackages(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	configs := []models.SystemConfig{
		{Key: "custom_package_enabled", Value: "true", Type: "boolean", Category: "custom_package", DisplayName: "启用自定义套餐"},
		{Key: "custom_package_price_per_device_year", Value: "40", Type: "string", Category: "custom_package", DisplayName: "每设备每年价格"},
		{Key: "custom_package_min_devices", Value: "5", Type: "string", Category: "custom_package", DisplayName: "最小设备数"},
		{Key: "custom_package_max_devices", Value: "100", Type: "string", Category: "custom_package", DisplayName: "最大设备数"},
		{Key: "custom_package_min_months", Value: "6", Type: "string", Category: "custom_package", DisplayName: "最小购买月数"},
	}
	if err := db.Create(&configs).Error; err != nil {
		t.Fatal(err)
	}

	recorder := performPaymentFlowRequest(http.MethodGet, "/settings/public-settings", "", models.User{}, GetPublicSettings)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected public settings status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("parse public settings response: %v body=%s", err, recorder.Body.String())
	}
	if !payload.Success {
		t.Fatalf("expected success response, got body=%s", recorder.Body.String())
	}
	if payload.Data["custom_package_enabled"] != true {
		t.Fatalf("expected custom package enabled=true, got %#v body=%s", payload.Data["custom_package_enabled"], recorder.Body.String())
	}
	if payload.Data["custom_package_min_devices"] != float64(5) {
		t.Fatalf("expected min devices 5, got %#v body=%s", payload.Data["custom_package_min_devices"], recorder.Body.String())
	}
}

func TestProcessPaidRechargeIsIdempotent(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_recharge", Email: "pay_recharge@example.com", Password: "x", Balance: 5}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	recharge := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       "RCHTEST001",
		Amount:        25,
		Status:        "pending",
		PaymentMethod: database.NullString("alipay"),
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	params := map[string]string{
		"out_trade_no": "RCHTEST001",
		"trade_no":     "TRADE001",
		"total_amount": "25.00",
	}

	if _, err := processPaidRecharge(db, recharge.OrderNo, "alipay", 0, "TRADE001", params, ""); err != nil {
		t.Fatalf("first recharge: %v", err)
	}
	if _, err := processPaidRecharge(db, recharge.OrderNo, "alipay", 0, "TRADE001", params, ""); err != nil {
		t.Fatalf("second recharge: %v", err)
	}

	var freshUser models.User
	db.First(&freshUser, user.ID)
	if freshUser.Balance != 30 {
		t.Fatalf("expected balance 30 after duplicate recharge callback, got %.2f", freshUser.Balance)
	}
}

func TestProcessPaidRechargeUpdatesTransactionByOrderNo(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_recharge_same_amount", Email: "pay_recharge_same_amount@example.com", Password: "x", Balance: 0}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	first := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       "RCHTESTSAME001",
		Amount:        25,
		Status:        "pending",
		PaymentMethod: database.NullString("alipay"),
	}
	second := models.RechargeRecord{
		UserID:        user.ID,
		OrderNo:       "RCHTESTSAME002",
		Amount:        25,
		Status:        "pending",
		PaymentMethod: database.NullString("alipay"),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PaymentTransaction{
		OrderID:         0,
		UserID:          user.ID,
		PaymentMethodID: 1,
		Amount:          2500,
		TransactionID:   database.NullString(first.OrderNo),
		Status:          "pending",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PaymentTransaction{
		OrderID:         0,
		UserID:          user.ID,
		PaymentMethodID: 1,
		Amount:          2500,
		TransactionID:   database.NullString(second.OrderNo),
		Status:          "pending",
	}).Error; err != nil {
		t.Fatal(err)
	}

	params := map[string]string{
		"out_trade_no": second.OrderNo,
		"trade_no":     "TRADE-SAME-002",
		"total_amount": "25.00",
	}
	if _, err := processPaidRecharge(db, second.OrderNo, "alipay", 1, "TRADE-SAME-002", params, ""); err != nil {
		t.Fatalf("process second recharge: %v", err)
	}

	var firstTx models.PaymentTransaction
	if err := db.Where("transaction_id = ?", first.OrderNo).First(&firstTx).Error; err != nil {
		t.Fatal(err)
	}
	if firstTx.Status != "pending" {
		t.Fatalf("first transaction should remain pending, got %s", firstTx.Status)
	}

	var secondTx models.PaymentTransaction
	if err := db.Where("transaction_id = ?", second.OrderNo).First(&secondTx).Error; err != nil {
		t.Fatal(err)
	}
	if secondTx.Status != "success" {
		t.Fatalf("second transaction should be success, got %s", secondTx.Status)
	}
	if !secondTx.ExternalTransactionID.Valid || secondTx.ExternalTransactionID.String != "TRADE-SAME-002" {
		t.Fatalf("second transaction external id not updated: %+v", secondTx.ExternalTransactionID)
	}
}

func TestScanAdminMergedOrderRefsPaginatesAcrossOrdersAndRecharges(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "merged_user", Email: "merged@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	baseTime := time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC)
	order := models.Order{
		OrderNo:   "ORDMERGED001",
		UserID:    user.ID,
		PackageID: 0,
		Amount:    10,
		Status:    "paid",
		CreatedAt: baseTime.Add(2 * time.Minute),
	}
	recharge := models.RechargeRecord{
		UserID:    user.ID,
		OrderNo:   "RCHMERGED001",
		Amount:    20,
		Status:    "paid",
		CreatedAt: baseTime.Add(3 * time.Minute),
	}
	olderOrder := models.Order{
		OrderNo:   "ORDMERGED002",
		UserID:    user.ID,
		PackageID: 0,
		Amount:    30,
		Status:    "pending",
		CreatedAt: baseTime.Add(time.Minute),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&olderOrder).Error; err != nil {
		t.Fatal(err)
	}

	refs, total, err := scanAdminMergedOrderRefs(db, "", "", 0, 2)
	if err != nil {
		t.Fatalf("scan refs: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(refs) != 2 {
		t.Fatalf("expected two refs on first page, got %d", len(refs))
	}
	if refs[0].RecordType != "recharge" || refs[0].ID != recharge.ID {
		t.Fatalf("expected newest recharge first, got %+v", refs[0])
	}
	if refs[1].RecordType != "order" || refs[1].ID != order.ID {
		t.Fatalf("expected order second, got %+v", refs[1])
	}

	refs, total, err = scanAdminMergedOrderRefs(db, "", "paid", 0, 10)
	if err != nil {
		t.Fatalf("scan paid refs: %v", err)
	}
	if total != 2 || len(refs) != 2 {
		t.Fatalf("expected two paid refs, total=%d len=%d", total, len(refs))
	}
	for _, ref := range refs {
		if ref.ID == olderOrder.ID {
			t.Fatalf("pending order leaked into paid filter: %+v", ref)
		}
	}
}

func TestPaymentSummaryIncludesOrdersAndRecharges(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_summary", Email: "pay_summary@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	order := models.Order{
		OrderNo:     "ORDSUMMARY001",
		UserID:      user.ID,
		PackageID:   0,
		Amount:      100,
		FinalAmount: database.NullFloat64(80),
		Status:      "paid",
		CreatedAt:   now,
	}
	freeOrder := models.Order{
		OrderNo:     "ORDSUMMARYFREE001",
		UserID:      user.ID,
		PackageID:   0,
		Amount:      100,
		FinalAmount: database.NullFloat64(0),
		Status:      "paid",
		CreatedAt:   now,
	}
	recharge := models.RechargeRecord{
		UserID:    user.ID,
		OrderNo:   "RCHSUMMARY001",
		Amount:    25,
		Status:    "paid",
		CreatedAt: now,
	}
	pendingRecharge := models.RechargeRecord{
		UserID:    user.ID,
		OrderNo:   "RCHSUMMARY002",
		Amount:    10,
		Status:    "pending",
		CreatedAt: now,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&freeOrder).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pendingRecharge).Error; err != nil {
		t.Fatal(err)
	}

	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	summary := utils.CalculatePaymentSummary(db, start, end)
	if summary.Total != 4 || summary.Paid != 3 || summary.Pending != 1 || summary.RangePaid != 3 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.PaidRevenue != 105 || summary.RangeRevenue != 105 {
		t.Fatalf("unexpected summary revenue: %+v", summary)
	}
}

func TestUserPaymentSummaryIncludesOrdersAndRecharges(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "user_summary", Email: "user_summary@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	otherUser := models.User{Username: "other_summary", Email: "other_summary@example.com", Password: "x"}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatal(err)
	}

	records := []models.Order{
		{OrderNo: "ORDUSERSUM001", UserID: user.ID, PackageID: 0, Amount: 100, FinalAmount: database.NullFloat64(80), Status: "paid"},
		{OrderNo: "ORDUSERSUMFREE001", UserID: user.ID, PackageID: 0, Amount: 100, FinalAmount: database.NullFloat64(0), Status: "paid"},
		{OrderNo: "ORDUSERSUM002", UserID: user.ID, PackageID: 0, Amount: 30, Status: "pending"},
		{OrderNo: "ORDUSERSUM003", UserID: user.ID, PackageID: 0, Amount: 40, Status: "cancelled"},
		{OrderNo: "ORDUSERSUM004", UserID: otherUser.ID, PackageID: 0, Amount: 999, Status: "paid"},
	}
	for _, order := range records {
		if err := db.Create(&order).Error; err != nil {
			t.Fatal(err)
		}
	}
	recharges := []models.RechargeRecord{
		{UserID: user.ID, OrderNo: "RCHUSERSUM001", Amount: 25, Status: "paid"},
		{UserID: user.ID, OrderNo: "RCHUSERSUM002", Amount: 10, Status: "pending"},
		{UserID: otherUser.ID, OrderNo: "RCHUSERSUM003", Amount: 999, Status: "paid"},
	}
	for _, recharge := range recharges {
		if err := db.Create(&recharge).Error; err != nil {
			t.Fatal(err)
		}
	}

	summary := utils.CalculateUserPaymentSummary(db, user.ID)
	if summary.Total != 6 || summary.Pending != 2 || summary.Paid != 3 || summary.Cancelled != 1 {
		t.Fatalf("unexpected user summary counts: %+v", summary)
	}
	if summary.PaidAmount != 105 {
		t.Fatalf("unexpected user paid amount: %+v", summary)
	}
}

func TestFinalizePaidOrderBalanceRepayDeductsStoredAndRemainingBalance(t *testing.T) {
	db := setupPaymentFlowTestDB(t)
	user := models.User{Username: "pay_balance_repay", Email: "pay_balance_repay@example.com", Password: "x", Balance: 100}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	pkg := models.Package{Name: "Pro", Price: 50, DurationDays: 30, DeviceLimit: 5, IsActive: true}
	if err := db.Create(&pkg).Error; err != nil {
		t.Fatal(err)
	}
	order := models.Order{
		OrderNo:     "ORDTESTBAL001",
		UserID:      user.ID,
		PackageID:   pkg.ID,
		Amount:      50,
		FinalAmount: database.NullFloat64(30),
		Status:      "pending",
		ExtraData:   database.NullString(`{"balance_used":20,"duration_months":1}`),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	svc := orderServicePkg.NewOrderService()
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
		PaymentMethodName: "余额支付",
		BalanceAmount:     30,
	}); err != nil {
		t.Fatalf("finalize balance repay: %v", err)
	}
	if _, err := svc.FinalizePaidOrder(order.OrderNo, orderServicePkg.FinalizePaidOrderOptions{
		PaymentMethodName: "余额支付",
		BalanceAmount:     30,
	}); err != nil {
		t.Fatalf("duplicate finalize balance repay: %v", err)
	}

	var freshUser models.User
	db.First(&freshUser, user.ID)
	if freshUser.Balance != 50 {
		t.Fatalf("expected one total balance deduction of 50, got balance %.2f", freshUser.Balance)
	}
	if freshUser.TotalConsumption != 50 {
		t.Fatalf("expected total consumption 50 after paid order, got %.2f", freshUser.TotalConsumption)
	}

	var freshOrder models.Order
	db.First(&freshOrder, order.ID)
	if !freshOrder.FinalAmount.Valid || freshOrder.FinalAmount.Float64 != 30 {
		t.Fatalf("expected external final amount to stay 30 after balance repay, got %+v", freshOrder.FinalAmount)
	}

	if err := svc.ProcessRefundOrder(&freshOrder); err != nil {
		t.Fatalf("refund balance repay order: %v", err)
	}
	db.First(&freshUser, user.ID)
	if freshUser.Balance != 100 {
		t.Fatalf("expected balance restored to 100 after refund, got %.2f", freshUser.Balance)
	}
	if freshUser.TotalConsumption != 0 {
		t.Fatalf("expected total consumption restored to zero after refund, got %.2f", freshUser.TotalConsumption)
	}
}
