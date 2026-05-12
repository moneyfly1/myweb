package promotion

import (
	"cboard-go/internal/models"
	"cboard-go/internal/utils"
	"encoding/json"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ApplyDiscount(userID uint, packageID uint, baseAmount float64) (float64, *models.PromotionParticipation, error) {
	now := utils.GetBeijingTime()

	var participation models.PromotionParticipation
	query := s.db.Where("user_id = ? AND status = ? AND reward_type = ? AND (order_id IS NULL OR order_id = 0) AND (expire_at IS NULL OR expire_at > ?)",
		userID, "pending", "discount", now).
		Preload("Promotion")

	if err := query.Order("created_at ASC").First(&participation).Error; err != nil {
		return 0, nil, nil
	}

	if !promotionAppliesToPackage(participation.Promotion, packageID) {
		return 0, nil, nil
	}
	if participation.Promotion.MinAmount > 0 && baseAmount < participation.Promotion.MinAmount {
		return 0, nil, nil
	}

	discountAmount := calculateDiscountAmount(participation, baseAmount)
	if discountAmount <= 0 {
		return 0, nil, nil
	}

	return utils.RoundFloat(discountAmount, 2), &participation, nil
}

func promotionAppliesToPackage(promotion models.Promotion, packageID uint) bool {
	if !promotion.PackageIDs.Valid || promotion.PackageIDs.String == "" {
		return true
	}

	var packageIDs []uint
	if err := json.Unmarshal([]byte(promotion.PackageIDs.String), &packageIDs); err != nil {
		return true
	}
	for _, id := range packageIDs {
		if id == packageID {
			return true
		}
	}
	return false
}

func calculateDiscountAmount(participation models.PromotionParticipation, baseAmount float64) float64 {
	switch participation.Promotion.DiscountType {
	case "percentage":
		discountAmount := baseAmount * (participation.RewardValue / 100)
		if participation.Promotion.MaxDiscount > 0 && discountAmount > participation.Promotion.MaxDiscount {
			return participation.Promotion.MaxDiscount
		}
		return discountAmount
	case "fixed":
		if participation.RewardValue > baseAmount {
			return baseAmount
		}
		return participation.RewardValue
	default:
		return 0
	}
}
