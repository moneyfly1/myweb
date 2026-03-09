# 前端安全改进文档

## 已实施的安全改进

### 1. ✅ 修复 NPM 依赖漏洞

**修改文件：** `package.json`

**更改内容：**
```json
{
  "@typescript-eslint/eslint-plugin": "^8.20.0",  // 从 ^6.12.0 升级
  "@typescript-eslint/parser": "^8.20.0",         // 从 ^6.12.0 升级
  "@vitejs/plugin-vue": "^6.0.1"                  // 从 ^4.5.0 升级
}
```

**验证：**
```bash
cd frontend
npm audit
# 输出：found 0 vulnerabilities
```

---

### 2. ✅ 修复 iframe 沙箱漏洞

**修改文件：** `src/views/admin/EmailQueue.vue`

**问题：** iframe 使用 `sandbox="allow-same-origin"` 允许访问父页面，存在 XSS 风险

**修复前：**
```vue
<iframe
  :srcdoc="sanitizedEmailContent"
  sandbox="allow-same-origin"
></iframe>
```

**修复后：**
```vue
<iframe
  :srcdoc="sanitizedEmailContent"
  sandbox=""
></iframe>
```

**说明：** 空 sandbox 属性提供最严格的隔离，防止 iframe 内容访问父页面

---

### 3. ✅ 加强 DOMPurify 配置

**修改文件：** `src/views/admin/EmailQueue.vue`

**问题：** 使用黑名单模式（ALLOWED_TAGS: null）和 ALLOW_UNKNOWN_PROTOCOLS: true 存在安全风险

**修复前：**
```javascript
DOMPurify.sanitize(html, {
  ALLOWED_TAGS: null,              // 允许所有标签
  ALLOWED_ATTR: null,              // 允许所有属性
  ALLOW_DATA_ATTR: true,
  ALLOW_UNKNOWN_PROTOCOLS: true,   // 允许未知协议
  FORBID_TAGS: ['script']          // 黑名单模式
})
```

**修复后：**
```javascript
DOMPurify.sanitize(html, {
  ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'b', 'i', 'u', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'a', 'div', 'span', 'blockquote', 'pre', 'code', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'img', 'hr', 'center'],
  ALLOWED_ATTR: ['href', 'target', 'rel', 'class', 'id', 'src', 'alt', 'width', 'height', 'align', 'border', 'cellpadding', 'cellspacing', 'bgcolor', 'color'],
  ALLOWED_URI_REGEXP: /^(?:(?:(?:f|ht)tps?|mailto|tel|callto|sms|cid|xmpp):|[^a-z]|[a-z+.\-]+(?:[^a-z+.\-:]|$))/i,
  ALLOW_DATA_ATTR: false,          // 禁止 data-* 属性
  KEEP_CONTENT: true,
  SAFE_FOR_TEMPLATES: true,
  FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form', 'input', 'button'],
  FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onmouseout', 'onfocus', 'onblur', 'onchange', 'onsubmit', 'onmouseenter', 'onmouseleave', 'onkeydown', 'onkeyup', 'onkeypress']
})
```

**改进点：**
- ✅ 使用白名单模式（ALLOWED_TAGS）
- ✅ 移除 ALLOW_UNKNOWN_PROTOCOLS
- ✅ 禁用 data-* 属性
- ✅ 使用 ALLOWED_URI_REGEXP 限制 URL 协议
- ✅ 扩展禁止的事件处理器列表

---

### 4. ✅ 添加安全响应头

**修改文件：** `index.html`

**添加的安全头：**
```html
<meta http-equiv="X-Content-Type-Options" content="nosniff">
<meta http-equiv="X-Frame-Options" content="SAMEORIGIN">
<meta http-equiv="Referrer-Policy" content="strict-origin-when-cross-origin">
<meta http-equiv="Permissions-Policy" content="geolocation=(), microphone=(), camera=()">
```

**说明：**
- `X-Content-Type-Options: nosniff` - 防止 MIME 类型嗅探
- `X-Frame-Options: SAMEORIGIN` - 防止点击劫持
- `Referrer-Policy` - 控制 Referer 头信息泄露
- `Permissions-Policy` - 限制浏览器功能访问

---

## 建议的后端配置

### 1. 服务器端安全响应头

**建议在 Go 后端添加以下响应头：**

```go
// 在 cmd/server/main.go 或中间件中添加
func SecurityHeadersMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 防止 MIME 类型嗅探
        c.Header("X-Content-Type-Options", "nosniff")

        // 防止点击劫持
        c.Header("X-Frame-Options", "SAMEORIGIN")

        // XSS 保护
        c.Header("X-XSS-Protection", "1; mode=block")

        // Referrer 策略
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

        // 权限策略
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

        // HSTS (仅在 HTTPS 环境下启用)
        if c.Request.TLS != nil {
            c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        }

        // Content Security Policy
        csp := "default-src 'self'; " +
               "script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
               "style-src 'self' 'unsafe-inline'; " +
               "img-src 'self' data: https:; " +
               "font-src 'self' data:; " +
               "connect-src 'self'; " +
               "frame-ancestors 'self'; " +
               "base-uri 'self'; " +
               "form-action 'self'"
        c.Header("Content-Security-Policy", csp)

        c.Next()
    }
}
```

### 2. Cookie 安全配置

**建议配置：**

```go
// 设置 Cookie 时使用安全选项
http.SetCookie(w, &http.Cookie{
    Name:     "session_token",
    Value:    token,
    Path:     "/",
    HttpOnly: true,        // 防止 JavaScript 访问
    Secure:   true,        // 仅通过 HTTPS 传输
    SameSite: http.SameSiteStrictMode,  // 防止 CSRF
    MaxAge:   86400,       // 24小时
})

// CSRF Token Cookie
http.SetCookie(w, &http.Cookie{
    Name:     "csrf_token",
    Value:    csrfToken,
    Path:     "/",
    HttpOnly: false,       // 需要 JavaScript 读取
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
    MaxAge:   86400,
})
```

---

## 安全检查清单

### 前端安全 ✅

- [x] NPM 依赖无已知漏洞
- [x] 所有 v-html 使用都经过 DOMPurify 净化
- [x] DOMPurify 使用白名单模式
- [x] iframe 使用严格的 sandbox 属性
- [x] 添加安全响应头（meta 标签）
- [x] Token 存储使用过期机制
- [x] CSRF Token 机制已实现
- [x] 敏感数据使用前缀隔离

### 后端安全（建议）

- [ ] 实施服务器端安全响应头
- [ ] Cookie 使用 HttpOnly 和 Secure 标志
- [ ] 实施 SameSite Cookie 策略
- [ ] 配置 CORS 白名单
- [ ] 实施速率限制
- [ ] 输入验证和输出编码
- [ ] SQL 注入防护（使用参数化查询）
- [ ] 日志记录和监控

---

## 测试验证

### 1. 依赖漏洞扫描

```bash
cd frontend
npm audit
# 预期输出：found 0 vulnerabilities
```

### 2. XSS 测试

测试 v-html 是否正确净化：

```javascript
// 测试用例
const maliciousHTML = '<img src=x onerror=alert(1)>'
const sanitized = DOMPurify.sanitize(maliciousHTML, config)
// 预期：<img src="x"> (移除 onerror)
```

### 3. iframe 隔离测试

```javascript
// 在 iframe 中尝试访问父页面
window.parent.document
// 预期：抛出 SecurityError
```

### 4. 安全头验证

使用浏览器开发者工具检查响应头：

```
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
Referrer-Policy: strict-origin-when-cross-origin
```

---

## 持续安全维护

### 1. 定期依赖更新

```bash
# 每月检查依赖更新
npm outdated

# 更新依赖
npm update

# 审计漏洞
npm audit
```

### 2. 安全扫描工具

推荐使用以下工具：

- **npm audit** - 依赖漏洞扫描
- **ESLint Security Plugin** - 代码安全检查
- **OWASP ZAP** - Web 应用安全扫描
- **Snyk** - 持续安全监控

### 3. 代码审查检查点

- 所有用户输入必须验证和净化
- 避免使用 eval() 和 Function()
- 谨慎使用 v-html 和 innerHTML
- 检查第三方库的安全性
- 确保敏感数据不在客户端存储

---

## 参考资源

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [DOMPurify Documentation](https://github.com/cure53/DOMPurify)
- [Content Security Policy](https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP)
- [Vue.js Security Best Practices](https://vuejs.org/guide/best-practices/security.html)
- [npm Security Best Practices](https://docs.npmjs.com/packages-and-modules/securing-your-code)

---

## 联系与支持

如发现安全问题，请通过以下方式报告：

- 创建 GitHub Issue（非敏感问题）
- 发送邮件至安全团队（敏感问题）

**注意：** 请勿在公开渠道披露未修复的安全漏洞。
