package email

import (
	"fmt"
	"regexp"
	"strings"

	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// EmailTemplateService 邮件模板服务
type EmailTemplateService struct {
	db *gorm.DB
}

// NewEmailTemplateService 创建邮件模板服务
func NewEmailTemplateService() *EmailTemplateService {
	return &EmailTemplateService{
		db: database.GetDB(),
	}
}

// GetTemplate 获取邮件模板
func (s *EmailTemplateService) GetTemplate(name string) (*models.EmailTemplate, error) {
	var template models.EmailTemplate
	if err := database.GetDB().Where("name = ? AND is_active = ?", name, true).First(&template).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: %v", err)
	}
	return &template, nil
}

// RenderTemplate 渲染模板（替换变量）
func (s *EmailTemplateService) RenderTemplate(template *models.EmailTemplate, variables map[string]string) (string, string, error) {
	subject := template.Subject
	content := template.Content

	// 替换变量 {{variable_name}}
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

// SendTemplatedEmail 发送模板邮件
func (s *EmailTemplateService) SendTemplatedEmail(templateName string, to string, variables map[string]string) error {
	// 获取模板
	template, err := s.GetTemplate(templateName)
	if err != nil {
		return err
	}

	// 渲染模板
	subject, content, err := s.RenderTemplate(template, variables)
	if err != nil {
		return err
	}

	// 发送邮件
	emailService := NewEmailService()
	return emailService.SendEmail(to, subject, content)
}

// SendVerificationEmailWithTemplate 使用模板发送验证邮件
func (s *EmailTemplateService) SendVerificationEmailWithTemplate(to, code string) error {
	variables := map[string]string{
		"code":     code,
		"email":    to,
		"validity": "10",
	}
	return s.SendTemplatedEmail("verification", to, variables)
}

// SendPasswordResetEmailWithTemplate 使用模板发送密码重置邮件
func (s *EmailTemplateService) SendPasswordResetEmailWithTemplate(to, resetLink string) error {
	variables := map[string]string{
		"reset_link": resetLink,
		"email":      to,
	}
	return s.SendTemplatedEmail("password_reset", to, variables)
}

// SendSubscriptionEmailWithTemplate 使用模板发送订阅邮件
func (s *EmailTemplateService) SendSubscriptionEmailWithTemplate(to, subscriptionURL string) error {
	variables := map[string]string{
		"subscription_url": subscriptionURL,
		"email":            to,
	}
	return s.SendTemplatedEmail("subscription", to, variables)
}

