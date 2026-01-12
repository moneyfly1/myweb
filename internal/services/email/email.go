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

// EmailService 邮件服务
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

// NewEmailService 创建邮件服务（从数据库读取配置）
func NewEmailService() *EmailService {
	// 优先从数据库读取配置
	db := database.GetDB()
	emailConfig := getEmailConfigFromDB(db)

	// 如果数据库中没有配置，使用环境变量
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

	// 从数据库配置创建服务
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

	// 根据加密方式设置默认端口
	if port == 587 && encryption == "ssl" {
		port = 465
	} else if port == 465 && encryption == "tls" {
		port = 587
	}

	// 获取发件人邮箱，优先使用 from_email，如果没有则使用 email_username
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

// encodeSubject 编码邮件标题（支持中文和emoji）
func encodeSubject(subject string) string {
	// 检查是否包含非ASCII字符
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

	// 使用RFC 2047编码（Q-encoding）
	encoded := mime.QEncoding.Encode("UTF-8", subject)
	// 如果编码后的长度超过75字符，需要分段编码
	if len(encoded) > 75 {
		// 分段处理长标题
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
				// 当前行已满，开始新行
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

// SendEmail 发送邮件
func (s *EmailService) SendEmail(to, subject, body string) error {
	// 验证基本配置
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

	// 构建邮件头
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", encodeSubject(s.fromName), s.from)
	headers["To"] = to
	headers["Subject"] = encodeSubject(subject)
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

	if s.encryption == "ssl" {
		// SSL 连接（端口465）：直接建立TLS连接
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
		// TLS 连接（端口587）：先建立普通连接，然后使用STARTTLS升级
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("连接SMTP服务器失败: %v", err)
		}
		defer client.Close()

		// 发送EHLO命令
		if err = client.Hello("localhost"); err != nil {
			return fmt.Errorf("发送EHLO失败: %v", err)
		}

		// 启动TLS
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("启动TLS失败: %v", err)
		}

		// 认证
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %v", err)
		}

		// 设置发件人
		if err = client.Mail(s.from); err != nil {
			return fmt.Errorf("设置发件人失败: %v", err)
		}

		// 设置收件人
		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %v", err)
		}

		// 发送邮件内容
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
		// 无加密连接（不推荐，仅用于测试）
		return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(message))
	}
}

// getTemplateContent 获取模板内容（支持回退）
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

// SendVerificationEmail 发送验证邮件（立即发送，验证码需要实时性，同时记录到队列）
func (s *EmailService) SendVerificationEmail(to, code string) error {
	// 验证邮件配置
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

	// 验证码邮件立即发送（验证码需要实时性）
	err := s.SendEmail(to, subject, content)

	// 无论发送成功与否，都记录到队列中（用于追踪和管理）
	queueErr := s.QueueEmail(to, subject, content, "verification")
	if queueErr != nil {
		// 记录队列失败，但不影响发送流程
		if err == nil {
			// 发送成功但队列失败，记录警告但不返回错误
			return nil
		}
		// 发送失败且队列也失败
		return fmt.Errorf("发送验证码邮件失败: %v，加入队列也失败: %v", err, queueErr)
	}

	// 如果发送成功，更新队列状态为已发送
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

// SendPasswordResetEmail 发送密码重置邮件（使用模板，加入队列）
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
			// 记录详细错误日志
			utils.LogErrorMsg("发送队列邮件失败: ID=%d, To=%s, Type=%s, Retry=%d/%d, Error=%v",
				email.ID, email.ToEmail, email.EmailType, email.RetryCount+1, email.MaxRetries, err)

			// 更新重试次数
			email.RetryCount++
			if email.RetryCount >= email.MaxRetries {
				email.Status = "failed"
				email.ErrorMessage = database.NullString(err.Error())
				utils.LogErrorMsg("邮件发送最终失败: ID=%d, To=%s, Type=%s, Error=%v",
					email.ID, email.ToEmail, email.EmailType, err)
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
			utils.LogInfo("邮件发送成功: ID=%d, To=%s, Type=%s", email.ID, email.ToEmail, email.EmailType)
			if err := db.Save(email).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
