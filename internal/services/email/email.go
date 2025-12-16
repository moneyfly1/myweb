package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"

	"gorm.io/gorm"
)

// EmailService 邮件服务
type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
	tls      bool
}

// NewEmailService 创建邮件服务（从数据库读取配置）
func NewEmailService() *EmailService {
	// 优先从数据库读取配置
	db := database.GetDB()
	emailConfig := getEmailConfigFromDB(db)

	// 如果数据库中没有配置，使用环境变量
	if emailConfig["smtp_host"] == "" {
		cfg := config.AppConfig
		return &EmailService{
			host:     cfg.SMTPHost,
			port:     cfg.SMTPPort,
			username: cfg.SMTPUser,
			password: cfg.SMTPPassword,
			from:     cfg.EmailsFromEmail,
			fromName: cfg.EmailsFromName,
			tls:      cfg.SMTPTLS,
		}
	}

	// 从数据库配置创建服务
	port := 587
	if p, ok := emailConfig["smtp_port"].(int); ok {
		port = p
	} else if pStr, ok := emailConfig["smtp_port"].(string); ok {
		if _, err := fmt.Sscanf(pStr, "%d", &port); err != nil {
			port = 587
		}
	}

	useTLS := true
	if encryption, ok := emailConfig["smtp_encryption"].(string); ok {
		useTLS = encryption == "tls" || encryption == "ssl"
	}

	return &EmailService{
		host:     getStringFromConfig(emailConfig, "smtp_host", "smtp.qq.com"),
		port:     port,
		username: getStringFromConfig(emailConfig, "smtp_username", getStringFromConfig(emailConfig, "email_username", "")),
		password: getStringFromConfig(emailConfig, "smtp_password", getStringFromConfig(emailConfig, "email_password", "")),
		from:     getStringFromConfig(emailConfig, "from_email", getStringFromConfig(emailConfig, "sender_email", "")),
		fromName: getStringFromConfig(emailConfig, "sender_name", getStringFromConfig(emailConfig, "from_name", "CBoard")),
		tls:      useTLS,
	}
}

// getEmailConfigFromDB 从数据库获取邮件配置
func getEmailConfigFromDB(db *gorm.DB) map[string]interface{} {
	configMap := make(map[string]interface{})
	var configs []models.SystemConfig
	db.Where("category = ?", "email").Find(&configs)

	for _, config := range configs {
		// 尝试转换为整数（如果是端口）
		if config.Key == "smtp_port" {
			var port int
			if _, err := fmt.Sscanf(config.Value, "%d", &port); err == nil {
				configMap[config.Key] = port
			} else {
				configMap[config.Key] = config.Value
			}
		} else {
			configMap[config.Key] = config.Value
		}
	}

	return configMap
}

// getStringFromConfig 从配置中获取字符串值
func getStringFromConfig(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

// SendEmail 发送邮件
func (s *EmailService) SendEmail(to, subject, body string) error {
	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// SMTP 认证
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// 发送邮件
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	if s.tls {
		// TLS 连接
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return err
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return err
		}

		if err = client.Mail(s.from); err != nil {
			return err
		}

		if err = client.Rcpt(to); err != nil {
			return err
		}

		writer, err := client.Data()
		if err != nil {
			return err
		}

		_, err = writer.Write([]byte(message))
		if err != nil {
			return err
		}

		err = writer.Close()
		if err != nil {
			return err
		}

		return client.Quit()
	} else {
		// 普通连接
		return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
	}
}

// SendVerificationEmail 发送验证邮件（使用模板，加入队列）
func (s *EmailService) SendVerificationEmail(to, code string) error {
	// 尝试使用模板
	templateService := NewEmailTemplateService()
	template, err := templateService.GetTemplate("verification")
	if err == nil {
		// 使用模板
		variables := map[string]string{
			"code":     code,
			"email":    to,
			"validity": "10",
		}
		subject, content, err := templateService.RenderTemplate(template, variables)
		if err == nil {
			return s.QueueEmail(to, subject, content, "verification")
		}
	}

	// 回退到默认模板
	subject := "邮箱验证"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>邮箱验证</h2>
			<p>您的验证码是：<strong>%s</strong></p>
			<p>验证码有效期为 10 分钟，请勿泄露给他人。</p>
		</body>
		</html>
	`, code)

	return s.QueueEmail(to, subject, body, "verification")
}

// SendPasswordResetEmail 发送密码重置邮件（使用模板，加入队列）
func (s *EmailService) SendPasswordResetEmail(to, resetLink string) error {
	// 尝试使用模板
	templateService := NewEmailTemplateService()
	template, err := templateService.GetTemplate("password_reset")
	if err == nil {
		// 使用模板
		variables := map[string]string{
			"reset_link": resetLink,
			"email":      to,
		}
		subject, content, err := templateService.RenderTemplate(template, variables)
		if err == nil {
			return s.QueueEmail(to, subject, content, "password_reset")
		}
	}

	// 回退到默认模板
	subject := "密码重置"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>密码重置</h2>
			<p>您请求重置密码，请点击以下链接：</p>
			<p><a href="%s">%s</a></p>
			<p>如果这不是您的操作，请忽略此邮件。</p>
		</body>
		</html>
	`, resetLink, resetLink)

	return s.QueueEmail(to, subject, body, "password_reset")
}

// QueueEmail 将邮件加入队列
func (s *EmailService) QueueEmail(to, subject, content, emailType string) error {
	db := database.GetDB()

	emailQueue := models.EmailQueue{
		ToEmail:     to,
		Subject:     subject,
		Content:     content,
		ContentType: "html",
		EmailType:   emailType,
		Status:      "pending",
		MaxRetries:  3,
	}

	return db.Create(&emailQueue).Error
}

// ProcessEmailQueue 处理邮件队列
func (s *EmailService) ProcessEmailQueue() error {
	db := database.GetDB()

	var emails []models.EmailQueue
	if err := db.Where("status = ? AND retry_count < max_retries", "pending").Order("created_at ASC").Limit(10).Find(&emails).Error; err != nil {
		return err
	}

	if len(emails) == 0 {
		return nil
	}

	for i := range emails {
		email := &emails[i]
		err := s.SendEmail(email.ToEmail, email.Subject, email.Content)
		if err != nil {
			// 更新重试次数
			email.RetryCount++
			if email.RetryCount >= email.MaxRetries {
				email.Status = "failed"
				email.ErrorMessage = database.NullString(err.Error())
			} else {
				// 仍然保持 pending 状态，等待下次重试
				email.Status = "pending"
			}
			if err := db.Save(email).Error; err != nil {
				return err
			}
		} else {
			// 标记为已发送
			email.Status = "sent"
			now := time.Now()
			email.SentAt = database.NullTime(now)
			if err := db.Save(email).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
