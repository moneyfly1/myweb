package email

import (
	"fmt"
	"regexp"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/gorm"
)

type EmailTemplateService struct {
	db *gorm.DB
}

func NewEmailTemplateService() *EmailTemplateService {
	return &EmailTemplateService{
		db: database.GetDB(),
	}
}

func (s *EmailTemplateService) GetTemplate(name string) (*models.EmailTemplate, error) {
	var template models.EmailTemplate
	if err := database.GetDB().Where("name = ? AND is_active = ?", name, true).First(&template).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: %v", err)
	}
	return &template, nil
}

func (s *EmailTemplateService) RenderTemplate(template *models.EmailTemplate, variables map[string]string) (string, string, error) {
	subject := template.Subject
	content := template.Content

	re := regexp.MustCompile(`\{\{(\w+)\}\}`)

	subject = re.ReplaceAllStringFunc(subject, func(match string) string {
		varName := strings.Trim(match, "{}")
		if val, ok := variables[varName]; ok {
			return val
		}
		return match
	})

	content = re.ReplaceAllStringFunc(content, func(match string) string {
		varName := strings.Trim(match, "{}")
		if val, ok := variables[varName]; ok {
			return val
		}
		return match
	})

	return subject, content, nil
}
