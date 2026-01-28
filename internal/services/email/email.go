package email

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/models"
	"cboard-go/internal/utils"

	"gorm.io/gorm"
)

type EmailService struct {
	host       string
	port       int
	username   string
	password   string
	from       string
	fromName   string
	tls        bool
	encryption string // "tls", "ssl", "none"
}

func NewEmailService() *EmailService {
	db := database.GetDB()
	emailConfig := getEmailConfigFromDB(db)

	if emailConfig["smtp_host"] == "" {
		cfg := config.AppConfig
		encryption := "tls"
		if cfg.SMTPTLS {
			encryption = "tls"
		}
		return &EmailService{
			host:       cfg.SMTPHost,
			port:       cfg.SMTPPort,
			username:   cfg.SMTPUser,
			password:   cfg.SMTPPassword,
			from:       cfg.EmailsFromEmail,
			fromName:   cfg.EmailsFromName,
			tls:        cfg.SMTPTLS,
			encryption: encryption,
		}
	}

	port := 587
	if p, ok := emailConfig["smtp_port"].(int); ok {
		port = p
	} else if pStr, ok := emailConfig["smtp_port"].(string); ok {
		if _, err := fmt.Sscanf(pStr, "%d", &port); err != nil {
			port = 587
		}
	}

	encryption := "tls"
	if enc, ok := emailConfig["smtp_encryption"].(string); ok && enc != "" {
		encryption = enc
	}
	useTLS := encryption == "tls" || encryption == "ssl"

	if port == 587 && encryption == "ssl" {
		port = 465
	} else if port == 465 && encryption == "tls" {
		port = 587
	}

	fromEmail := getStringFromConfig(emailConfig, "from_email", "")
	if fromEmail == "" {
		fromEmail = getStringFromConfig(emailConfig, "sender_email", "")
	}
	if fromEmail == "" {
		fromEmail = getStringFromConfig(emailConfig, "email_username", "")
	}

	return &EmailService{
		host:       getStringFromConfig(emailConfig, "smtp_host", "smtp.qq.com"),
		port:       port,
		username:   getStringFromConfig(emailConfig, "smtp_username", getStringFromConfig(emailConfig, "email_username", "")),
		password:   getStringFromConfig(emailConfig, "smtp_password", getStringFromConfig(emailConfig, "email_password", "")),
		from:       fromEmail,
		fromName:   getStringFromConfig(emailConfig, "sender_name", getStringFromConfig(emailConfig, "from_name", "CBoard")),
		tls:        useTLS,
		encryption: encryption,
	}
}

func getEmailConfigFromDB(db *gorm.DB) map[string]interface{} {
	configMap := make(map[string]interface{})
	var configs []models.SystemConfig
	db.Where("category = ?", "email").Find(&configs)

	for _, config := range configs {
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

func getStringFromConfig(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

func encodeSubject(subject string) string {
	needsEncoding := false
	for _, r := range subject {
		if r > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return subject
	}

	encoded := mime.QEncoding.Encode("UTF-8", subject)
	if len(encoded) > 75 {
		var parts []string
		words := strings.Fields(subject)
		currentLine := ""

		for _, word := range words {
			testLine := currentLine
			if testLine != "" {
				testLine += " " + word
			} else {
				testLine = word
			}

			encodedTest := mime.QEncoding.Encode("UTF-8", testLine)
			if len(encodedTest) > 75 && currentLine != "" {
				parts = append(parts, mime.QEncoding.Encode("UTF-8", currentLine))
				currentLine = word
			} else {
				currentLine = testLine
			}
		}

		if currentLine != "" {
			parts = append(parts, mime.QEncoding.Encode("UTF-8", currentLine))
		}

		return strings.Join(parts, "\r\n ")
	}

	return encoded
}

func (s *EmailService) SendEmail(to, subject, body string) error {
	if s.host == "" {
		return fmt.Errorf("SMTP服务器地址未配置")
	}
	if s.username == "" {
		return fmt.Errorf("SMTP用户名未配置")
	}
	if s.password == "" {
		return fmt.Errorf("SMTP密码未配置")
	}
	if s.from == "" {
		return fmt.Errorf("发件人邮箱未配置")
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", encodeSubject(s.fromName), s.from)
	headers["To"] = to
	headers["Subject"] = encodeSubject(subject)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	if s.encryption == "ssl" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("SSL连接失败: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("创建SMTP客户端失败: %v", err)
		}
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %v", err)
		}

		if err = client.Mail(s.from); err != nil {
			return fmt.Errorf("设置发件人失败: %v", err)
		}

		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %v", err)
		}

		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("创建数据写入器失败: %v", err)
		}

		_, err = writer.Write([]byte(message))
		if err != nil {
			writer.Close()
			return fmt.Errorf("写入邮件内容失败: %v", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("关闭数据写入器失败: %v", err)
		}

		return client.Quit()
	} else if s.encryption == "tls" {
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("连接SMTP服务器失败: %v", err)
		}
		defer client.Close()

		if err = client.Hello("localhost"); err != nil {
			return fmt.Errorf("发送EHLO失败: %v", err)
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("启动TLS失败: %v", err)
		}

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %v", err)
		}

		if err = client.Mail(s.from); err != nil {
			return fmt.Errorf("设置发件人失败: %v", err)
		}

		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %v", err)
		}

		writer, err := client.Data()
		if err != nil {
			return fmt.Errorf("创建数据写入器失败: %v", err)
		}

		_, err = writer.Write([]byte(message))
		if err != nil {
			writer.Close()
			return fmt.Errorf("写入邮件内容失败: %v", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("关闭数据写入器失败: %v", err)
		}

		return client.Quit()
	} else {
		return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
	}
}

func (s *EmailService) getTemplateContent(templateName string, variables map[string]string, fallbackBuilder func() (string, string)) (string, string) {
	templateService := NewEmailTemplateService()
	template, err := templateService.GetTemplate(templateName)
	if err == nil {
		subject, content, err := templateService.RenderTemplate(template, variables)
		if err == nil {
			return subject, content
		}
	}
	return fallbackBuilder()
}

func (s *EmailService) SendVerificationEmail(to, code string) error {
	if s.host == "" || s.username == "" || s.password == "" {
		return fmt.Errorf("邮件配置不完整，请先配置SMTP设置")
	}

	subject, content := s.getTemplateContent("verification", map[string]string{
		"code":     code,
		"email":    to,
		"validity": "10",
	}, func() (string, string) {
		templateBuilder := NewEmailTemplateBuilder()
		content := templateBuilder.GetVerificationCodeTemplate("用户", code)
		return "注册验证码", content
	})

	err := s.SendEmail(to, subject, content)

	queueErr := s.QueueEmail(to, subject, content, "verification")
	if queueErr != nil {
		if err == nil {
			return nil
		}
		return fmt.Errorf("发送验证码邮件失败: %v，加入队列也失败: %v", err, queueErr)
	}

	if err == nil {
		db := database.GetDB()
		var emailQueue models.EmailQueue
		if err := db.Where("to_email = ? AND subject = ? AND email_type = ? AND status = ?", to, subject, "verification", "pending").Order("created_at DESC").First(&emailQueue).Error; err == nil {
			emailQueue.Status = "sent"
			emailQueue.SentAt = database.NullTime(time.Now())
			db.Save(&emailQueue)
		}
	}

	if err != nil {
		return fmt.Errorf("发送验证码邮件失败: %v，已加入队列稍后重试", err)
	}

	return nil
}

func (s *EmailService) SendPasswordResetEmail(to, resetLink string) error {
	subject, content := s.getTemplateContent("password_reset", map[string]string{
		"reset_link": resetLink,
		"email":      to,
	}, func() (string, string) {
		templateBuilder := NewEmailTemplateBuilder()
		content := templateBuilder.GetPasswordResetTemplate("用户", resetLink)
		return "密码重置", content
	})

	return s.QueueEmail(to, subject, content, "password_reset")
}

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
			utils.LogErrorMsg("发送队列邮件失败: ID=%d, To=%s, Type=%s, Retry=%d/%d, Error=%v",
				email.ID, email.ToEmail, email.EmailType, email.RetryCount+1, email.MaxRetries, err)

			email.RetryCount++
			if email.RetryCount >= email.MaxRetries {
				email.Status = "failed"
				email.ErrorMessage = database.NullString(err.Error())
				utils.LogErrorMsg("邮件发送最终失败: ID=%d, To=%s, Type=%s, Error=%v",
					email.ID, email.ToEmail, email.EmailType, err)
			} else {
				email.Status = "pending"
			}
			if err := db.Save(email).Error; err != nil {
				return err
			}
		} else {
			email.Status = "sent"
			now := time.Now()
			email.SentAt = database.NullTime(now)
			utils.LogInfo("邮件发送成功: ID=%d, To=%s, Type=%s", email.ID, email.ToEmail, email.EmailType)
			if err := db.Save(email).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
