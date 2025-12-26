package email

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"cboard-go/internal/core/config"
	"cboard-go/internal/core/database"
	"cboard-go/internal/utils"
)

// EmailTemplateBuilder 邮件模板构建器
type EmailTemplateBuilder struct{}

// NewEmailTemplateBuilder 创建邮件模板构建器
func NewEmailTemplateBuilder() *EmailTemplateBuilder {
	return &EmailTemplateBuilder{}
}

// GetBaseURL 获取基础URL（公开方法）
func (b *EmailTemplateBuilder) GetBaseURL() string {
	return b.getBaseURL()
}

// getBaseURL 获取基础URL（内部方法）
func (b *EmailTemplateBuilder) getBaseURL() string {
	// 从数据库配置获取域名（使用公共函数）
	db := database.GetDB()
	if db != nil {
		domain := utils.GetDomainFromDB(db)
		if domain != "" {
			return utils.FormatDomainURL(domain)
		}
	}

	// 从环境变量获取
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		return baseURL
	}

	// 从配置文件获取
	if config.AppConfig.BaseURL != "" {
		return config.AppConfig.BaseURL
	}

	// 默认值
	return "http://localhost:5173"
}

// GetBaseTemplate 获取基础邮件模板
func (b *EmailTemplateBuilder) GetBaseTemplate(title, content, footerText string) string {
	currentYear := time.Now().Year()
	siteName := "网络服务"

	baseTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {margin: 0; padding: 0; font-family: "Helvetica Neue", Helvetica, Arial, sans-serif; background-color: #f4f4f4; color: #333;}
        .email-container {max-width: 600px; margin: 0 auto; background-color: #ffffff; box-shadow: 0 4px 12px rgba(0,0,0,0.1);}
        .header {background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px 20px; text-align: center;}
        .header h1 {margin: 0; font-size: 28px; font-weight: 300;}
        .header .subtitle {margin: 10px 0 0 0; font-size: 16px; opacity: 0.9;}
        .content {padding: 40px 30px;}
        .content h2 {color: #333; font-size: 24px; margin-bottom: 20px; font-weight: 400;}
        .content p {line-height: 1.6; margin-bottom: 16px; color: #555;}
        .info-box {background-color: #f8f9fa; border-left: 4px solid #667eea; padding: 20px; margin: 20px 0; border-radius: 4px;}
        .info-table {width: 100%; border-collapse: collapse; margin: 20px 0;}
        .info-table th, .info-table td {padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef;}
        .info-table th {background-color: #f8f9fa; font-weight: 600; color: #495057; width: 30%;}
        .btn {display: inline-block; padding: 12px 30px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; text-decoration: none; border-radius: 25px; font-weight: 500; margin: 20px 0; transition: all 0.3s ease;}
        .btn:hover {transform: translateY(-2px); box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);}
        .warning-box {background-color: #fff3cd; border: 1px solid #ffeaa7; border-radius: 4px; padding: 15px; margin: 20px 0; color: #856404;}
        .success-box {background-color: #d4edda; border: 1px solid #c3e6cb; border-radius: 4px; padding: 15px; margin: 20px 0; color: #155724;}
        .footer {background-color: #f8f9fa; padding: 30px; text-align: center; border-top: 1px solid #e9ecef;}
        .footer p {margin: 5px 0; color: #6c757d; font-size: 14px;}
        .url-list {margin: 15px 0;}
        .url-item {background-color: #f8f9fa; border: 1px solid #e9ecef; border-radius: 6px; padding: 15px; margin: 10px 0; border-left: 4px solid #667eea;}
        .url-item strong {color: #333; font-size: 14px; display: block; margin-bottom: 8px;}
        .url-code {background-color: #ffffff; border: 1px solid #dee2e6; border-radius: 4px; padding: 10px; margin: 5px 0; word-break: break-all; font-family: 'Courier New', monospace; font-size: 12px; color: #495057; display: block; line-height: 1.4;}
        @media only screen and (max-width: 600px) {
            .email-container {width: 100% !important;}
            .content {padding: 20px !important;}
            .header {padding: 20px !important;}
            .header h1 {font-size: 24px !important;}
        }
    </style>
</head>
<body>
    <div class="email-container">
        <div class="header">
            <h1>{{.SiteName}}</h1>
            <p class="subtitle">{{.Title}}</p>
        </div>
        <div class="content">{{.Content}}</div>
        <div class="footer">
            <p><strong>{{.SiteName}}</strong></p>
            <p>{{.FooterText}}</p>
            <p style="font-size: 12px; color: #999;">此邮件由系统自动发送，请勿直接回复</p>
            <p style="font-size: 12px; color: #999;">© {{.CurrentYear}} {{.SiteName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("base").Parse(baseTemplate)
	if err != nil {
		return fmt.Sprintf(`<html><body><h2>%s</h2>%s</body></html>`, title, content)
	}

	var buf bytes.Buffer
	data := map[string]interface{}{
		"Title":       title,
		"Content":     template.HTML(content),
		"FooterText":  footerText,
		"SiteName":    siteName,
		"CurrentYear": currentYear,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf(`<html><body><h2>%s</h2>%s</body></html>`, title, content)
	}

	return buf.String()
}

// GetVerificationCodeTemplate 获取验证码邮件模板
func (b *EmailTemplateBuilder) GetVerificationCodeTemplate(username, verificationCode string) string {
	title := "注册验证码"
	content := fmt.Sprintf(`<h2>📧 您的注册验证码</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>感谢您注册我们的服务！请使用以下验证码完成注册：</p>
            <div style="text-align: center; margin: 30px 0;">
                <div style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 20px 40px; border-radius: 8px; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
                    <div style="font-size: 32px; font-weight: bold; color: #ffffff; letter-spacing: 8px; font-family: 'Courier New', monospace;">%s</div>
                </div>
            </div>
            <div class="info-box">
                <p><strong>📋 使用说明：</strong></p>
                <ul>
                    <li>此验证码有效期为 <strong>10分钟</strong></li>
                    <li>请在注册页面输入此验证码完成注册</li>
                    <li>验证码仅限本次使用，使用后自动失效</li>
                    <li>如果验证码过期，请重新获取</li>
                </ul>
            </div>
            <div class="warning-box">
                <p><strong>⚠️ 安全提示：</strong></p>
                <p>请勿将验证码告知他人。如果这不是您本人的操作，请忽略此邮件。您的账户安全对我们非常重要。</p>
            </div>`, username, verificationCode)

	return b.GetBaseTemplate(title, content, "完成注册，开启您的专属网络体验")
}

// GetPasswordResetTemplate 获取密码重置邮件模板
func (b *EmailTemplateBuilder) GetPasswordResetTemplate(username, resetLink string) string {
	title := "密码重置"
	content := fmt.Sprintf(`<h2>您的密码重置请求</h2>
            <p>亲爱的 %s，</p>
            <p>我们收到了您的密码重置请求。如果这不是您本人的操作，请忽略此邮件。</p>
            <div class="info-box">
                <h3>📋 重置信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>重置链接有效期</th><td style="color: #ffc107; font-weight: bold;">1小时</td></tr>
                    <tr><th>链接使用次数</th><td>仅可使用一次</td></tr>
                </table>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="btn">重置密码</a>
            </div>
            <div class="warning-box">
                <h3>⚠️ 安全提醒</h3>
                <ul>
                    <li>此重置链接仅在1小时内有效</li>
                    <li>链接仅可使用一次，使用后自动失效</li>
                    <li>如果链接失效，请重新申请密码重置</li>
                    <li>如果按钮无法点击，请复制以下链接到浏览器中打开：</li>
                </ul>
                <div style="margin-top: 15px; padding: 10px; background: #f8f9fa; border-radius: 4px; word-break: break-all;">
                    <code style="color: #667eea; font-size: 12px;">%s</code>
                </div>
            </div>
            <div class="info-box">
                <p><strong>💡 密码安全建议：</strong></p>
                <ul>
                    <li>建议设置强密码，包含字母、数字和特殊字符</li>
                    <li>密码长度建议在8-50个字符之间</li>
                    <li>不要使用过于简单的密码，如"123456"、"password"等</li>
                    <li>定期更换密码以确保账户安全</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如果您没有请求重置密码，请忽略此邮件</p>`, username, username, resetLink, resetLink)

	return b.GetBaseTemplate(title, content, "保护您的账户安全")
}

// GetPasswordResetVerificationCodeTemplate 获取密码重置验证码邮件模板
func (b *EmailTemplateBuilder) GetPasswordResetVerificationCodeTemplate(username, verificationCode string) string {
	title := "密码重置验证码"
	content := fmt.Sprintf(`<h2>🔐 您的密码重置验证码</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>您正在重置账户密码，请使用以下验证码完成重置：</p>
            <div style="text-align: center; margin: 30px 0;">
                <div style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 20px 40px; border-radius: 8px; box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);">
                    <div style="font-size: 32px; font-weight: bold; color: #ffffff; letter-spacing: 8px; font-family: 'Courier New', monospace;">%s</div>
                </div>
            </div>
            <div class="info-box">
                <p><strong>📋 使用说明：</strong></p>
                <ul>
                    <li>此验证码有效期为 <strong>10分钟</strong></li>
                    <li>请在密码重置页面输入此验证码和新密码完成重置</li>
                    <li>验证码仅限本次使用，使用后自动失效</li>
                    <li>如果验证码过期，请重新获取</li>
                </ul>
            </div>
            <div class="warning-box">
                <p><strong>⚠️ 安全提示：</strong></p>
                <p>请勿将验证码告知他人。如果这不是您本人的操作，请立即忽略此邮件并联系客服。您的账户安全对我们非常重要。</p>
            </div>`, username, verificationCode)

	return b.GetBaseTemplate(title, content, "安全重置您的账户密码")
}

// GetSubscriptionTemplate 获取订阅信息邮件模板
func (b *EmailTemplateBuilder) GetSubscriptionTemplate(username, universalURL, clashURL, expireTime string, remainingDays, deviceLimit, currentDevices int) string {
	title := "服务配置信息"

	urlList := ""
	if universalURL != "" {
		urlList += fmt.Sprintf(`<div class="url-item">
                        <strong>🔗 通用配置地址（推荐）：</strong>
                        <p style="margin: 5px 0; color: #666; font-size: 12px;">适用于大部分客户端，包括手机和电脑</p>
                        <code class="url-code">%s</code>
                    </div>`, universalURL)
	}
	if clashURL != "" {
		urlList += fmt.Sprintf(`<div class="url-item">
                        <strong>⚡ Clash 类型软件专用地址：</strong>
                        <p style="margin: 5px 0; color: #666; font-size: 12px;">适用于 Clash、ClashX、Clash for Windows 等 Clash 类型软件</p>
                        <code class="url-code">%s</code>
                    </div>`, clashURL)
	}

	remainingColor := "#e74c3c"
	if remainingDays > 7 {
		remainingColor = "#27ae60"
	}

	content := fmt.Sprintf(`<h2>您的服务配置信息</h2>
            <p>亲爱的 %s，</p>
            <p>您的服务配置已生成完成，请查收以下信息：</p>
            <div class="success-box">
                <h3>📡 订阅信息</h3>
                <table class="info-table">
                    <tr><th>到期时间</th><td style="color: %s; font-weight: bold;">%s</td></tr>
                    <tr><th>剩余时长</th><td style="color: %s; font-weight: bold;">%d 天</td></tr>
                    <tr><th>允许最大设备数</th><td style="color: #27ae60; font-weight: bold;">%d 台设备</td></tr>
                    <tr><th>当前使用设备</th><td>%d / %d</td></tr>
                </table>
            </div>
            <div class="success-box">
                <h3>🔗 配置地址</h3>
                <div class="url-list">%s</div>
            </div>
            <div class="warning-box">
                <p><strong>⚠️ 安全提醒：</strong></p>
                <ul>
                    <li>请妥善保管您的配置地址，切勿分享给他人</li>
                    <li>如发现地址泄露，请及时联系客服重置</li>
                    <li>建议定期更换配置地址以确保安全</li>
                    <li>服务到期前会收到续费提醒邮件</li>
                </ul>
            </div>`, username, remainingColor, expireTime, remainingColor, remainingDays, deviceLimit, currentDevices, deviceLimit, urlList)

	return b.GetBaseTemplate(title, content, "享受高速稳定的网络服务")
}

// GetOrderConfirmationTemplate 获取订单确认邮件模板
func (b *EmailTemplateBuilder) GetOrderConfirmationTemplate(username, orderNo, packageName string, amount float64, paymentMethod, orderTime string) string {
	title := "订单确认"
	content := fmt.Sprintf(`<h2>✅ 订单确认</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>感谢您的购买！您的订单已成功创建，详情如下：</p>
            <div class="info-box">
                <h3>📋 订单详情</h3>
                <table class="info-table">
                    <tr><th>订单号</th><td><strong>%s</strong></td></tr>
                    <tr><th>套餐名称</th><td>%s</td></tr>
                    <tr><th>订单金额</th><td style="color: #e74c3c; font-weight: bold; font-size: 18px;">¥%.2f</td></tr>
                    <tr><th>支付方式</th><td>%s</td></tr>
                    <tr><th>下单时间</th><td>%s</td></tr>
                    <tr><th>订单状态</th><td><span style="color: #ffc107; font-weight: bold;">待支付</span></td></tr>
                </table>
            </div>
            <div class="warning-box">
                <p><strong>⏰ 重要提醒：</strong></p>
                <ul>
                    <li>请尽快完成支付，订单将在24小时后自动取消</li>
                    <li>支付成功后，服务将自动激活，无需额外操作</li>
                    <li>支付完成后，您将收到包含订阅地址的确认邮件</li>
                    <li>如有任何疑问，请及时联系客服</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">感谢您选择我们的服务！</p>`, username, orderNo, packageName, amount, paymentMethod, orderTime)

	return b.GetBaseTemplate(title, content, "开启您的专属网络体验")
}

// GetPaymentSuccessTemplate 获取支付成功邮件模板
func (b *EmailTemplateBuilder) GetPaymentSuccessTemplate(username, orderNo, packageName string, amount float64, paymentMethod, paymentTime string) string {
	title := "支付成功通知"
	content := fmt.Sprintf(`<h2>🎉 支付成功！</h2>
            <p>亲爱的 %s，</p>
            <p>您的支付已成功处理，感谢您的购买！</p>
            <div class="success-box">
                <h3>✅ 支付确认</h3>
                <table class="info-table">
                    <tr><th>订单号</th><td><strong>%s</strong></td></tr>
                    <tr><th>套餐名称</th><td><strong>%s</strong></td></tr>
                    <tr><th>支付金额</th><td style="color: #27ae60; font-weight: bold; font-size: 18px;">¥%.2f</td></tr>
                    <tr><th>支付方式</th><td>%s</td></tr>
                    <tr><th>支付时间</th><td>%s</td></tr>
                    <tr><th>订单状态</th><td style="color: #27ae60; font-weight: bold;">✅ 已支付</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>✨ 服务已激活：</strong></p>
                <ul>
                    <li>✅ 您的订阅已自动激活</li>
                    <li>✅ 配置地址已生成并可用</li>
                    <li>✅ 可以立即开始使用服务</li>
                    <li>💡 您可以查看订阅详情获取配置地址</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如有任何问题，请随时联系我们的客服团队</p>`, username, orderNo, packageName, amount, paymentMethod, paymentTime)

	return b.GetBaseTemplate(title, content, "感谢您的信任")
}

// GetWelcomeTemplate 获取欢迎邮件模板
func (b *EmailTemplateBuilder) GetWelcomeTemplate(username, email, loginURL string, hasPassword bool, password string) string {
	title := "欢迎加入我们！"

	passwordRow := ""
	if hasPassword && password != "" {
		passwordRow = fmt.Sprintf(`<tr><th>登录密码</th><td style="color: #667eea; font-weight: bold; font-size: 16px;">%s</td></tr>`, password)
	}

	content := fmt.Sprintf(`<h2>您的账户注册成功</h2>
            <p>亲爱的 %s，</p>
            <p>欢迎加入我们的网络服务平台！您的账户已成功创建，现在可以开始使用我们的服务了。</p>
            <div class="info-box">
                <h3>📋 账户信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>注册邮箱</th><td>%s</td></tr>
                    %s
                    <tr><th>登录地址</th><td><a href="%s" style="color: #667eea; text-decoration: none;">%s</a></td></tr>
                </table>
            </div>
            <div class="warning-box">
                <h3>⚠️ 重要提示</h3>
                <ul>
                    <li>请妥善保管您的登录密码，建议您登录后及时修改密码</li>
                    <li>为了账户安全，建议设置强密码，包含字母、数字和特殊字符</li>
                    <li>不要将密码泄露给他人，避免账户被盗用</li>
                </ul>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="btn">立即登录</a>
            </div>`, username, username, email, passwordRow, loginURL, loginURL, loginURL)

	return b.GetBaseTemplate(title, content, "期待为您提供优质服务")
}

// GetUserCreatedTemplate 获取管理员创建用户通知邮件模板
func (b *EmailTemplateBuilder) GetUserCreatedTemplate(username, email, password, expireTime string, deviceLimit int) string {
	title := "账户创建通知"
	loginURL := fmt.Sprintf("%s/login", b.getBaseURL())

	expireDisplay := expireTime
	if expireTime == "" || expireTime == "未设置" {
		expireDisplay = "未设置"
	}

	content := fmt.Sprintf(`<h2>您的账户已创建</h2>
            <p>亲爱的 %s，</p>
            <p>管理员已为您创建账户，以下是您的账户信息：</p>
            <div class="info-box">
                <h3>📋 账户信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>注册邮箱</th><td>%s</td></tr>
                    <tr><th>登录密码</th><td style="color: #667eea; font-weight: bold; font-size: 16px;">%s</td></tr>
                    <tr><th>登录地址</th><td><a href="%s" style="color: #667eea; text-decoration: none;">%s</a></td></tr>
                </table>
            </div>
            <div class="success-box">
                <h3>📡 服务信息</h3>
                <table class="info-table">
                    <tr><th>有效期</th><td style="color: #27ae60; font-weight: bold;">%s</td></tr>
                    <tr><th>允许最大设备数</th><td style="color: #27ae60; font-weight: bold;">%d 台设备</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <h3>⚠️ 重要提示</h3>
                <ul>
                    <li>请妥善保管您的登录密码，建议您登录后及时修改密码</li>
                    <li>为了账户安全，建议设置强密码，包含字母、数字和特殊字符</li>
                    <li>不要将密码泄露给他人，避免账户被盗用</li>
                    <li>服务到期时间为：<strong>%s</strong></li>
                    <li>您最多可以同时使用 <strong>%d 台设备</strong>连接服务</li>
                </ul>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="btn">立即登录</a>
            </div>`, username, username, email, password, loginURL, loginURL, expireDisplay, deviceLimit, expireDisplay, deviceLimit, loginURL)

	return b.GetBaseTemplate(title, content, "期待为您提供优质服务")
}

// GetPasswordChangedTemplate 获取密码修改成功邮件模板
func (b *EmailTemplateBuilder) GetPasswordChangedTemplate(username, changeTime, loginURL string) string {
	title := "密码修改成功"
	content := fmt.Sprintf(`<h2>您的密码已修改</h2>
            <p>亲爱的 %s，</p>
            <p>您的账户密码已成功修改。如果这不是您本人的操作，请立即联系客服。</p>
            <div class="info-box">
                <h3>📋 修改信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>修改时间</th><td>%s</td></tr>
                    <tr><th>修改状态</th><td style="color: #27ae60; font-weight: bold;">✅ 修改成功</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <h3>⚠️ 安全提醒</h3>
                <ul>
                    <li>如果这不是您本人的操作，请立即登录账户修改密码</li>
                    <li>建议定期更换密码以确保账户安全</li>
                    <li>不要使用过于简单的密码，如"123456"、"password"等</li>
                    <li>如发现账户异常，请及时联系客服</li>
                </ul>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="btn">立即登录</a>
            </div>
            <div class="info-box">
                <p><strong>💡 温馨提示：</strong></p>
                <ul>
                    <li>新密码已立即生效，请使用新密码登录</li>
                    <li>建议设置强密码，包含字母、数字和特殊字符</li>
                    <li>妥善保管您的账户信息，不要泄露给他人</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如有任何问题，请随时联系我们的客服团队</p>`, username, username, changeTime, loginURL)

	return b.GetBaseTemplate(title, content, "保护您的账户安全")
}

// GetSubscriptionResetTemplate 获取订阅重置邮件模板
func (b *EmailTemplateBuilder) GetSubscriptionResetTemplate(username, universalURL, clashURL, expireTime, resetTime, resetReason string) string {
	title := "订阅重置通知"

	urlList := ""
	if universalURL != "" {
		urlList += fmt.Sprintf(`<div class="url-item">
                        <strong>🔗 通用配置地址（推荐）：</strong>
                        <p style="margin: 5px 0; color: #666; font-size: 12px;">适用于大部分客户端，包括手机和电脑</p>
                        <code class="url-code">%s</code>
                    </div>`, universalURL)
	}
	if clashURL != "" {
		urlList += fmt.Sprintf(`<div class="url-item">
                        <strong>⚡ 移动端专用地址：</strong>
                        <p style="margin: 5px 0; color: #666; font-size: 12px;">专为移动设备优化，支持规则分流</p>
                        <code class="url-code">%s</code>
                    </div>`, clashURL)
	}

	baseURL := b.getBaseURL()

	content := fmt.Sprintf(`<h2>🔄 您的订阅已重置</h2>
            <p>亲爱的 %s，</p>
            <p>您的订阅地址已被重置，请使用新的订阅地址更新您的客户端配置。</p>
            <div class="info-box">
                <h3>📋 重置信息</h3>
                <table class="info-table">
                    <tr><th>重置时间</th><td><strong>%s</strong></td></tr>
                    <tr><th>重置原因</th><td>%s</td></tr>
                    <tr><th>订阅状态</th><td style="color: #27ae60; font-weight: bold;">✅ 已激活</td></tr>
                    <tr><th>到期时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="success-box">
                <h3>🔗 新的订阅地址</h3>
                <div class="url-list">%s</div>
            </div>
            <div class="warning-box">
                <h3>⚠️ 重要提醒</h3>
                <ul style="line-height: 2;">
                    <li><strong>立即更新</strong>：请立即更新您的客户端配置，使用新的订阅地址</li>
                    <li><strong>旧地址失效</strong>：旧的订阅地址已失效，将无法使用</li>
                    <li><strong>妥善保管</strong>：请妥善保管新的订阅地址，不要分享给他人</li>
                    <li><strong>设备清理</strong>：所有设备记录已清空，需要重新连接</li>
                    <li><strong>如有疑问</strong>：如有任何疑问，请及时联系客服</li>
                </ul>
            </div>
            <div class="info-box">
                <h3>📖 更新步骤</h3>
                <ol style="line-height: 2;">
                    <li>复制上方新的订阅地址</li>
                    <li>在客户端中删除旧的订阅配置</li>
                    <li>添加新的订阅配置</li>
                    <li>更新并测试连接</li>
                </ol>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s/dashboard" class="btn">查看订阅详情</a>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如有任何问题，请随时联系我们的客服团队</p>`, username, resetTime, resetReason, expireTime, urlList, baseURL)

	return b.GetBaseTemplate(title, content, "请及时更新您的客户端配置")
}

// GetAccountDeletionTemplate 获取账户删除确认邮件模板
func (b *EmailTemplateBuilder) GetAccountDeletionTemplate(username, deletionDate, reason, dataRetentionPeriod string) string {
	title := "账号删除确认"
	content := fmt.Sprintf(`<h2>账号删除确认</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>您的账号删除请求已收到，我们对此表示遗憾。</p>
            <div class="info-box">
                <table class="info-table">
                    <tr><th>删除原因</th><td>%s</td></tr>
                    <tr><th>删除时间</th><td>%s</td></tr>
                    <tr><th>数据保留期</th><td>%s</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <p><strong>重要提醒：</strong></p>
                <ul>
                    <li>您的账号将在数据保留期结束后永久删除</li>
                    <li>删除后无法恢复，请谨慎操作</li>
                    <li>如有疑问，请在保留期内联系客服</li>
                </ul>
            </div>
            <p>感谢您曾经选择我们的服务！</p>`, username, reason, deletionDate, dataRetentionPeriod)

	return b.GetBaseTemplate(title, content, "感谢您曾经选择我们的服务")
}

// GetAccountDeletionWarningTemplate 获取账户删除警告邮件模板
func (b *EmailTemplateBuilder) GetAccountDeletionWarningTemplate(username, email, lastLogin string, daysUntilDeletion int) string {
	title := "账号删除提醒"
	baseURL := b.getBaseURL()
	loginURL := fmt.Sprintf("%s/login", baseURL)

	content := fmt.Sprintf(`<h2>⚠️ 账号删除提醒</h2>
            <p>亲爱的 %s，</p>
            <p>我们注意到您的账号已经<strong>30天未登录</strong>，且<strong>没有有效的付费套餐</strong>。</p>
            <div class="warning-box">
                <h3>📋 账号状态</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>注册邮箱</th><td>%s</td></tr>
                    <tr><th>最后登录</th><td>%s</td></tr>
                    <tr><th>订阅状态</th><td style="color: #e74c3c; font-weight: bold;">无有效套餐</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <h3>⚠️ 重要通知</h3>
                <p>根据我们的账号管理政策，您的账号将在<strong style="color: #e74c3c;">%d天后</strong>被自动删除。</p>
                <p>如果您希望保留账号，请：</p>
                <ol style="line-height: 2;">
                    <li>立即登录账号（<a href="%s">点击登录</a>）</li>
                    <li>购买并激活有效的服务套餐</li>
                    <li>账号将自动保留</li>
                </ol>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s" class="btn">立即登录</a>
            </div>
            <div class="info-box">
                <p><strong>💡 温馨提示：</strong></p>
                <ul>
                    <li>账号删除后，所有数据将无法恢复</li>
                    <li>包括订阅记录、订单记录、设备记录等</li>
                    <li>如有任何疑问，请及时联系客服</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如有任何问题，请随时联系我们的客服团队</p>`, username, username, email, lastLogin, daysUntilDeletion, loginURL, loginURL)

	return b.GetBaseTemplate(title, content, "请及时登录以保留您的账号")
}

// GetExpirationReminderTemplate 获取到期提醒续费邮件模板
func (b *EmailTemplateBuilder) GetExpirationReminderTemplate(username, packageName, expireDate string, remainingDays, deviceLimit, currentDevices int, isExpired bool) string {
	title := "订阅已到期"
	if !isExpired {
		title = "订阅即将到期"
	}

	baseURL := b.getBaseURL()

	var headerContent string
	if isExpired {
		headerContent = fmt.Sprintf(`<h2>⚠️ 服务已到期</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>您的服务已于 <strong style="color: #e74c3c;">%s</strong> 到期。</p>
            <div class="warning-box">
                <p><strong>服务已暂停：</strong></p>
                <ul>
                    <li>您的配置地址已停止更新</li>
                    <li>无法获取最新的节点配置</li>
                    <li>请及时续费以恢复服务</li>
                </ul>
            </div>`, username, expireDate)
	} else {
		headerContent = fmt.Sprintf(`<h2>服务即将到期</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>您的服务将于 <strong style="color: #ffc107;">%s</strong> 到期。</p>
            <div class="warning-box">
                <p><strong>温馨提醒：</strong></p>
                <ul>
                    <li>为避免服务中断，请提前续费</li>
                    <li>到期后配置地址将停止更新</li>
                    <li>续费后服务将自动恢复</li>
                </ul>
            </div>`, username, expireDate)
	}

	remainingDaysRow := ""
	if !isExpired && remainingDays > 0 {
		remainingDaysRow = fmt.Sprintf(`<tr><th>剩余天数</th><td style="color: #ffc107; font-weight: bold;">%d 天</td></tr>`, remainingDays)
	}

	warningBox := ""
	if isExpired {
		warningBox = `<div class="warning-box">
                <p><strong>服务状态:</strong></p>
                <ul>
                    <li>订阅地址已停止更新,无法获取最新节点</li>
                    <li>现有配置可能暂时可用,但建议尽快续费</li>
                    <li>续费后服务将立即恢复</li>
                </ul>
            </div>`
	}

	buttonText := "查看订阅详情"
	if isExpired {
		buttonText = "立即续费"
	}

	content := fmt.Sprintf(`%s
            <div class="info-box">
                <h3>📋 订阅详情</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>套餐名称</th><td>%s</td></tr>
                    <tr><th>到期时间</th><td style="color: #e74c3c; font-weight: bold; font-size: 16px;">%s</td></tr>
                    %s
                    <tr><th>设备限制</th><td>%d 台设备</td></tr>
                    <tr><th>当前设备</th><td>%d / %d</td></tr>
                </table>
            </div>
            %s
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s/dashboard" class="btn">%s</a>
            </div>
            <div class="info-box">
                <p><strong>💡 续费说明：</strong></p>
                <ul>
                    <li>续费后，订阅地址将立即恢复更新</li>
                    <li>所有客户端配置无需修改，可直接使用</li>
                    <li>支持多种支付方式，支付成功后自动激活</li>
                </ul>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">如有任何问题，请随时联系我们的客服团队</p>`, headerContent, username, packageName, expireDate, remainingDaysRow, deviceLimit, currentDevices, deviceLimit, warningBox, baseURL, buttonText)

	return b.GetBaseTemplate(title, content, "我们期待继续为您服务")
}

// GetRenewalConfirmationTemplate 获取续费成功邮件模板
func (b *EmailTemplateBuilder) GetRenewalConfirmationTemplate(username, packageName, oldExpiryDate, newExpiryDate, renewalDate string, amount float64) string {
	title := "续费成功"
	baseURL := b.getBaseURL()

	content := fmt.Sprintf(`<h2>🎉 续费成功！</h2>
            <p>亲爱的用户 <strong>%s</strong>，</p>
            <p>恭喜！您的服务续费已成功完成，服务时间已自动延长。</p>
            <div class="success-box">
                <h3>✅ 续费详情</h3>
                <table class="info-table">
                    <tr><th>套餐名称</th><td><strong>%s</strong></td></tr>
                    <tr><th>原到期时间</th><td style="color: #999; text-decoration: line-through;">%s</td></tr>
                    <tr><th>新到期时间</th><td style="color: #27ae60; font-weight: bold; font-size: 16px;">%s</td></tr>
                    <tr><th>续费金额</th><td style="color: #e74c3c; font-weight: bold;">¥%.2f</td></tr>
                    <tr><th>续费时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>📋 服务说明：</strong></p>
                <ul>
                    <li>✅ 您的服务已成功续费，可立即继续使用</li>
                    <li>✅ 订阅配置地址保持不变，无需重新配置</li>
                    <li>✅ 所有客户端配置将继续正常工作</li>
                    <li>💡 建议定期更新订阅配置以获取最新节点信息</li>
                </ul>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s/dashboard" class="btn">查看订阅详情</a>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">感谢您的续费，祝您使用愉快！</p>`, username, packageName, oldExpiryDate, newExpiryDate, amount, renewalDate, baseURL)

	return b.GetBaseTemplate(title, content, "开启您的专属网络体验")
}

// GetMarketingEmailTemplate 获取营销邮件模板
func (b *EmailTemplateBuilder) GetMarketingEmailTemplate(title, content string) string {
	baseURL := b.getBaseURL()

	emailContent := fmt.Sprintf(`<h2>%s</h2>
            <div class="info-box">
                <div style="line-height: 1.8; color: #555;">%s</div>
            </div>
            <div style="text-align: center; margin: 30px 0;">
                <a href="%s/dashboard" class="btn">查看详情</a>
            </div>
            <p style="text-align: center; color: #666; font-size: 14px;">此邮件来自 网络服务</p>`, title, strings.ReplaceAll(content, "\n", "<br>"), baseURL)

	return b.GetBaseTemplate(title, emailContent, "感谢您的关注")
}

// GetBroadcastNotificationTemplate 获取广播通知邮件模板
func (b *EmailTemplateBuilder) GetBroadcastNotificationTemplate(title, content string) string {
	emailContent := fmt.Sprintf(`<div class="content">
                <h2>%s</h2>
                <div style="line-height: 1.8; color: #555;">%s</div>
            </div>`, title, strings.ReplaceAll(content, "\n", "<br>"))

	return b.GetBaseTemplate(title, emailContent, "此邮件由系统自动发送，请勿回复。")
}

// GetAdminNotificationTemplate 获取管理员通知邮件模板（规范工整格式）
func (b *EmailTemplateBuilder) GetAdminNotificationTemplate(notificationType, title, body string, data map[string]interface{}) string {
	var content string

	switch notificationType {
	case "order_paid":
		orderNo := getStringFromData(data, "order_no", "N/A")
		username := getStringFromData(data, "username", "N/A")
		amount := getFloatFromData(data, "amount", 0)
		packageName := getStringFromData(data, "package_name", "未知套餐")
		paymentMethod := getStringFromData(data, "payment_method", "未知")
		paymentTime := getStringFromData(data, "payment_time", "N/A")
		content = fmt.Sprintf(`<h2>💰 新订单支付成功</h2>
            <p>系统检测到一笔新的订单支付，详情如下：</p>
            <div class="success-box">
                <h3>📋 订单信息</h3>
                <table class="info-table">
                    <tr><th>订单号</th><td><strong style="font-family: 'Courier New', monospace;">%s</strong></td></tr>
                    <tr><th>用户账号</th><td>%s</td></tr>
                    <tr><th>套餐名称</th><td><strong>%s</strong></td></tr>
                    <tr><th>支付金额</th><td style="color: #27ae60; font-weight: bold; font-size: 18px;">¥%.2f</td></tr>
                    <tr><th>支付方式</th><td>%s</td></tr>
                    <tr><th>支付时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>💡 提示：</strong>订单已自动处理，订阅已激活，用户可立即使用服务。</p>
            </div>`, orderNo, username, packageName, amount, paymentMethod, paymentTime)

	case "user_registered":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		registerTime := getStringFromData(data, "register_time", "N/A")
		content = fmt.Sprintf(`<h2>👤 新用户注册</h2>
            <p>系统检测到新用户注册，详情如下：</p>
            <div class="info-box">
                <h3>📋 用户信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>注册邮箱</th><td>%s</td></tr>
                    <tr><th>注册时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>💡 提示：</strong>新用户已自动创建默认订阅，可引导用户购买套餐激活服务。</p>
            </div>`, username, email, registerTime)

	case "password_reset":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		resetTime := getStringFromData(data, "reset_time", "N/A")
		content = fmt.Sprintf(`<h2>🔐 用户重置密码</h2>
            <p>系统检测到用户重置密码操作，详情如下：</p>
            <div class="warning-box">
                <h3>📋 重置信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>用户邮箱</th><td>%s</td></tr>
                    <tr><th>重置时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <p><strong>⚠️ 安全提醒：</strong>如非用户本人操作，请及时检查账户安全。</p>
            </div>`, username, email, resetTime)

	case "subscription_sent":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		sendTime := getStringFromData(data, "send_time", "N/A")
		content = fmt.Sprintf(`<h2>📧 用户发送订阅</h2>
            <p>系统检测到用户发送订阅邮件，详情如下：</p>
            <div class="info-box">
                <h3>📋 发送信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>用户邮箱</th><td>%s</td></tr>
                    <tr><th>发送时间</th><td>%s</td></tr>
                </table>
            </div>`, username, email, sendTime)

	case "subscription_reset":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		resetTime := getStringFromData(data, "reset_time", "N/A")
		content = fmt.Sprintf(`<h2>🔄 用户重置订阅</h2>
            <p>系统检测到用户重置订阅地址，详情如下：</p>
            <div class="info-box">
                <h3>📋 重置信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>用户邮箱</th><td>%s</td></tr>
                    <tr><th>重置时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>💡 提示：</strong>订阅地址已重置，旧地址已失效，用户设备记录已清空。</p>
            </div>`, username, email, resetTime)

	case "subscription_expired":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		expireTime := getStringFromData(data, "expire_time", "N/A")
		content = fmt.Sprintf(`<h2>⏰ 订阅已过期</h2>
            <p>系统检测到用户订阅已过期，详情如下：</p>
            <div class="warning-box">
                <h3>📋 过期信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>用户邮箱</th><td>%s</td></tr>
                    <tr><th>过期时间</th><td style="color: #e74c3c; font-weight: bold;">%s</td></tr>
                </table>
            </div>
            <div class="warning-box">
                <p><strong>💡 提示：</strong>用户订阅已过期，建议引导用户续费以恢复服务。</p>
            </div>`, username, email, expireTime)

	case "user_created":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		createdBy := getStringFromData(data, "created_by", "N/A")
		createTime := getStringFromData(data, "create_time", "N/A")
		content = fmt.Sprintf(`<h2>📋 管理员创建用户</h2>
            <p>系统检测到管理员创建新用户，详情如下：</p>
            <div class="success-box">
                <h3>📋 账户信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong style="font-family: 'Courier New', monospace;">%s</strong></td></tr>
                    <tr><th>注册邮箱</th><td>%s</td></tr>
                    <tr><th>创建者</th><td>👤 %s</td></tr>
                    <tr><th>创建时间</th><td>⏰ %s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>✅ 用户账户已成功创建</strong></p>
            </div>`, username, email, createdBy, createTime)

	case "subscription_created":
		username := getStringFromData(data, "username", "N/A")
		email := getStringFromData(data, "email", "N/A")
		packageName := getStringFromData(data, "package_name", "未知套餐")
		createTime := getStringFromData(data, "create_time", "N/A")
		content = fmt.Sprintf(`<h2>📦 订阅创建</h2>
            <p>系统检测到新订阅创建，详情如下：</p>
            <div class="success-box">
                <h3>📋 订阅信息</h3>
                <table class="info-table">
                    <tr><th>用户账号</th><td><strong>%s</strong></td></tr>
                    <tr><th>用户邮箱</th><td>%s</td></tr>
                    <tr><th>套餐名称</th><td><strong>%s</strong></td></tr>
                    <tr><th>创建时间</th><td>%s</td></tr>
                </table>
            </div>
            <div class="info-box">
                <p><strong>💡 提示：</strong>订阅已创建并激活，用户可立即使用服务。</p>
            </div>`, username, email, packageName, createTime)

	default:
		content = fmt.Sprintf(`<div class="content">
                <h2>%s</h2>
                <div style="line-height: 1.8; color: #555;">%s</div>
            </div>`, title, strings.ReplaceAll(body, "\n", "<br>"))
	}

	return b.GetBaseTemplate(title, content, "此邮件由系统自动发送，请勿回复。")
}

// Helper functions for template
func getStringFromData(data map[string]interface{}, key string, defaultValue string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

func getFloatFromData(data map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultValue
}
