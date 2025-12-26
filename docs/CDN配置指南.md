# CDN 加速配置指南

## 概述

本指南将帮助您配置国内 CDN 来加速网站访问。支持阿里云、腾讯云、七牛云等主流 CDN 服务商。

## ⚠️ 重要提示：域名备案要求

### 国内 CDN 域名备案要求

**是的，使用国内 CDN 服务，域名必须完成 ICP 备案！**

#### 1. 备案要求说明

- **必须备案**：所有使用国内 CDN 的域名都必须完成 ICP 备案
- **备案主体**：个人或企业都可以备案
- **备案时间**：通常需要 7-20 个工作日
- **备案费用**：备案本身免费，但需要购买服务器（用于备案）

#### 2. 各服务商备案要求

| 服务商 | 备案要求 | 说明 |
|--------|---------|------|
| **阿里云 CDN** | ✅ 必须备案 | 域名必须完成 ICP 备案 |
| **腾讯云 CDN** | ✅ 必须备案 | 域名必须完成 ICP 备案 |
| **七牛云 CDN** | ✅ 必须备案 | 域名必须完成 ICP 备案 |
| **又拍云 CDN** | ✅ 必须备案 | 域名必须完成 ICP 备案 |
| **百度云 CDN** | ✅ 必须备案 | 域名必须完成 ICP 备案 |

#### 3. 备案流程

**步骤 1：准备材料**
- 个人备案：身份证正反面
- 企业备案：营业执照、法人身份证、网站负责人身份证

**步骤 2：购买服务器**
- 必须购买国内服务器（用于备案）
- 推荐：阿里云、腾讯云、华为云等

**步骤 3：提交备案**
- 在服务器提供商处提交备案申请
- 填写网站信息、域名信息等

**步骤 4：等待审核**
- 初审：1-3 个工作日
- 管局审核：7-20 个工作日

**步骤 5：备案通过**
- 获得备案号（例如：京ICP备12345678号）
- 在网站底部添加备案号链接

#### 4. 备案注意事项

⚠️ **重要提醒**：
- 备案期间域名无法访问（建议使用临时域名）
- 备案信息必须真实有效
- 备案后需要保持服务器运行（否则可能被注销）
- 备案主体变更需要重新备案

#### 5. 无备案解决方案

如果您的域名**无法备案**或**不想备案**，有以下替代方案：

**方案 A：使用海外 CDN**
- Cloudflare（免费，全球加速）
- AWS CloudFront
- Google Cloud CDN
- 优点：无需备案，全球加速
- 缺点：国内访问速度可能较慢

**方案 B：使用免备案 CDN（节点在境外）**

⚠️ **重要说明**：严格来说，没有真正的"国内免备案CDN"。以下服务商提供免备案服务，但节点主要在**香港、新加坡等境外地区**，国内访问速度可能不如备案后的国内CDN。

**免备案 CDN 服务商列表：**

1. **Cloudflare（推荐）**
   - 官网：https://www.cloudflare.com
   - 特点：
     - ✅ 免费套餐可用
     - ✅ 全球节点（包括香港）
     - ✅ DDoS 防护
     - ✅ SSL 证书免费
     - ✅ 配置简单
   - 缺点：
     - ⚠️ 国内访问速度一般
     - ⚠️ 免费版功能有限
   - 适用场景：个人网站、小型项目

2. **CDN5**
   - 官网：https://www.cdn5.com
   - 特点：
     - ✅ 专注免备案 CDN
     - ✅ 香港、东南亚节点
     - ✅ 按流量计费
   - 缺点：
     - ⚠️ 国内访问速度一般
     - ⚠️ 知名度较低
   - 适用场景：未备案域名加速

3. **StoneCDN**
   - 官网：https://www.stonecdn.com
   - 特点：
     - ✅ 游戏加速优化
     - ✅ 香港、美国节点
     - ✅ 高防服务
   - 缺点：
     - ⚠️ 主要面向游戏行业
     - ⚠️ 价格相对较高
   - 适用场景：游戏、直播等

4. **速盾（Sudun）**
   - 官网：https://www.sudun.com
   - 特点：
     - ✅ 高防 CDN
     - ✅ 香港、新加坡节点
     - ✅ 适合电商、金融
     - ✅ 专业防护
   - 缺点：
     - ⚠️ 价格较高
     - ⚠️ 主要面向企业
   - 适用场景：企业级应用、高防需求

5. **又拍云（部分免备案）**
   - 官网：https://www.upyun.com
   - 特点：
     - ✅ 部分海外节点免备案
     - ✅ 国内节点需要备案
     - ✅ 价格适中
   - 缺点：
     - ⚠️ 免备案节点有限
   - 适用场景：混合部署

6. **七牛云（海外节点）**
   - 官网：https://www.qiniu.com
   - 特点：
     - ✅ 海外节点免备案
     - ✅ 国内节点需要备案
     - ✅ 对象存储 + CDN
   - 缺点：
     - ⚠️ 海外节点价格较高
   - 适用场景：混合部署

**免备案 CDN 注意事项：**
- ⚠️ 节点主要在境外（香港、新加坡等）
- ⚠️ 国内访问速度可能不如备案后的国内CDN
- ⚠️ 可能受到网络波动影响
- ⚠️ 部分服务商稳定性一般
- ⚠️ 价格可能比备案后的国内CDN高

**方案 C：自建 CDN**
- 使用多台服务器部署
- 使用 Nginx 做负载均衡
- 成本较高，维护复杂

#### 6. 推荐方案

**如果域名可以备案**：
1. ✅ 完成 ICP 备案（必须）
2. ✅ 使用国内 CDN（阿里云/腾讯云等）
3. ✅ 享受国内高速访问

**如果域名无法备案**：
1. ✅ **首选：Cloudflare**（免费，全球加速，配置简单）
2. ✅ **备选：CDN5、StoneCDN、速盾**等免备案CDN
3. ⚠️ **注意**：国内访问速度可能不如备案后的国内CDN
4. 💡 **建议**：如果可能，尽量完成备案以获得最佳体验

## 方案一：静态资源 CDN 加速（推荐）

### 1. 准备工作

#### 1.1 确认备案状态

**⚠️ 重要：在开始之前，请确认您的域名已完成 ICP 备案！**

检查备案状态：
- 登录服务器提供商控制台
- 查看备案管理页面
- 确认备案状态为"已备案"

#### 1.2 选择 CDN 服务商

**国内 CDN（需要备案）：**
- **阿里云 CDN**：https://www.aliyun.com/product/cdn
- **腾讯云 CDN**：https://cloud.tencent.com/product/cdn
- **七牛云 CDN**：https://www.qiniu.com/products/cdn
- **又拍云 CDN**：https://www.upyun.com/products/cdn

**海外 CDN（无需备案）：**
- **Cloudflare**：https://www.cloudflare.com（免费）
- **AWS CloudFront**：https://aws.amazon.com/cloudfront
- **Google Cloud CDN**：https://cloud.google.com/cdn

#### 1.3 获取 CDN 域名

在 CDN 控制台创建加速域名，例如：
- `https://cdn.yourdomain.com`（需要备案）
- `https://static.yourdomain.com`（需要备案）
- `https://assets.yourdomain.com`（需要备案）

**注意**：CDN 域名也需要完成备案，或使用已备案的主域名的子域名。

### 2. 配置 Vite 使用 CDN

#### 2.1 修改 `frontend/vite.config.js`

```javascript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import { fileURLToPath, URL } from 'node:url'

const __dirname = fileURLToPath(new URL('.', import.meta.url))

// CDN 配置 - 从环境变量读取
const CDN_URL = process.env.VITE_CDN_URL || ''

export default defineConfig({
  root: resolve(__dirname),
  publicDir: resolve(__dirname, 'public'),
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  base: CDN_URL ? `${CDN_URL}/` : '/', // 设置基础路径
  optimizeDeps: {
    esbuildOptions: {
      target: 'esnext',
    },
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: process.env.NODE_ENV === 'development',
    minify: 'terser',
    cssCodeSplit: true,
    terserOptions: {
      compress: {
        drop_console: process.env.NODE_ENV === 'production',
        drop_debugger: process.env.NODE_ENV === 'production',
      },
    },
    rollupOptions: {
      output: {
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
        assetFileNames: 'assets/[name].[hash].[ext]',
      },
    },
    chunkSizeWarningLimit: 2000,
    reportCompressedSize: false,
  },
  css: {
    preprocessorOptions: {
      scss: {
        api: 'modern-compiler',
        silenceDeprecations: ['legacy-js-api'],
      },
    },
  },
})
```

#### 2.2 创建环境变量文件

创建 `frontend/.env.production`：

```bash
# CDN 域名（带协议，不带尾部斜杠）
VITE_CDN_URL=https://cdn.yourdomain.com
# API 基础地址
VITE_API_BASE_URL=https://yourdomain.com
```

### 3. 部署流程

#### 3.1 构建前端

```bash
cd frontend
npm install
npm run build
```

构建完成后，`frontend/dist` 目录包含所有静态资源。

#### 3.2 上传静态资源到 CDN

**方式一：使用 CDN 控制台上传**
1. 登录 CDN 控制台
2. 找到您的加速域名
3. 使用文件上传功能或 FTP 上传 `frontend/dist` 目录内容

**方式二：使用 CDN 提供的 CLI 工具**

**阿里云 OSS + CDN：**
```bash
# 安装 ossutil
wget http://gosspublic.alicdn.com/ossutil/1.7.14/ossutil64
chmod 755 ossutil64

# 配置
./ossutil64 config

# 上传
./ossutil64 cp -r frontend/dist/ oss://your-bucket-name/ --update
```

**腾讯云 COS + CDN：**
```bash
# 安装 coscmd
pip install coscmd

# 配置
coscmd config -a SecretId -s SecretKey -b BucketName -r Region

# 上传
coscmd upload -rs frontend/dist/ /
```

**七牛云：**
```bash
# 安装 qshell
wget https://devtools.qiniu.com/qshell-v2.11.0-linux-amd64.tar.gz
tar xvf qshell-v2.11.0-linux-amd64.tar.gz

# 配置
./qshell account AccessKey SecretKey

# 上传
./qshell qupload2 --src-dir=frontend/dist --bucket=your-bucket
```

#### 3.3 配置 CDN 回源

在 CDN 控制台配置回源设置：
- **回源协议**：HTTPS（推荐）或 HTTP
- **回源地址**：您的服务器 IP 或域名
- **回源 Host**：您的源站域名
- **缓存规则**：
  - HTML 文件：不缓存或缓存 1 分钟
  - JS/CSS/图片等静态资源：缓存 30 天
  - 设置缓存键忽略查询参数（除了版本号）

### 4. 配置 Nginx（可选）

如果使用 CDN，Nginx 配置可以简化：

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;
    
    # API 请求代理到后端
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # 静态资源重定向到 CDN（可选）
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        return 301 https://cdn.yourdomain.com$request_uri;
    }
    
    # HTML 文件由源站提供
    location / {
        root /path/to/frontend/dist;
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
}
```

## 方案二：全站 CDN 加速

### 1. 确认备案状态

**⚠️ 重要：全站 CDN 加速同样需要域名备案！**

在开始配置前：
1. ✅ 确认主域名已完成 ICP 备案
2. ✅ 确认子域名（如 www）可以使用已备案主域名
3. ✅ 准备备案号（需要在网站底部显示）

### 2. 配置 CDN 回源

在 CDN 控制台：
1. 添加加速域名（例如：`www.yourdomain.com`）
   - **系统会自动验证备案状态**
   - 如果未备案，会提示无法添加
2. 配置源站：您的服务器 IP 或域名
3. 配置回源 Host：`yourdomain.com`
4. 配置 HTTPS 证书
   - 可以上传自有证书
   - 或使用 CDN 提供的免费证书（需要验证域名所有权）

### 2. 配置缓存规则

**推荐缓存规则：**

| 文件类型 | 缓存时间 | 说明 |
|---------|---------|------|
| HTML | 不缓存 | 确保内容更新及时 |
| JS/CSS | 30天 | 文件名包含 hash，更新时自动失效 |
| 图片 | 30天 | 静态资源长期缓存 |
| API 响应 | 不缓存 | 动态内容不缓存 |

### 3. 配置 Nginx（源站）

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    # 只允许 CDN IP 访问（可选，提高安全性）
    # allow 1.2.3.4;  # CDN IP
    # deny all;
    
    # API 请求
    location /api/ {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # 静态文件
    location / {
        root /path/to/frontend/dist;
        try_files $uri $uri/ /index.html;
        
        # 设置缓存头
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
            expires 30d;
            add_header Cache-Control "public, immutable";
        }
        
        # HTML 不缓存
        location ~* \.html$ {
            add_header Cache-Control "no-cache, no-store, must-revalidate";
        }
    }
}
```

### 4. 修改 DNS

将域名 A 记录指向 CDN 提供的 CNAME 地址。

## 方案三：使用对象存储 + CDN（最佳实践）

### 1. 架构说明

```
用户请求 → CDN → 对象存储（OSS/COS）→ 源站（仅 API）
```

### 2. 配置步骤

1. **创建对象存储桶**
   - 设置公共读权限
   - 配置自定义域名（绑定 CDN）

2. **上传静态资源**
   - 使用 CLI 工具或控制台上传
   - 设置自动同步（可选）

3. **配置 CDN**
   - 源站类型：对象存储
   - 配置缓存规则
   - 配置 HTTPS

4. **配置自动部署脚本**

创建 `scripts/deploy-cdn.sh`：

```bash
#!/bin/bash

# 配置
CDN_URL="https://cdn.yourdomain.com"
OSS_BUCKET="your-bucket-name"
OSS_REGION="oss-cn-hangzhou"

# 构建前端
echo "正在构建前端..."
cd frontend
npm run build

# 上传到 OSS（示例：阿里云）
echo "正在上传到 OSS..."
ossutil cp -r dist/ oss://${OSS_BUCKET}/ --update

# 刷新 CDN 缓存（可选）
echo "正在刷新 CDN 缓存..."
# 使用 CDN API 刷新缓存

echo "部署完成！"
```

## 性能优化建议

### 1. 启用 Gzip/Brotli 压缩

在 CDN 控制台启用压缩功能，或配置 Nginx：

```nginx
gzip on;
gzip_vary on;
gzip_min_length 1024;
gzip_types text/plain text/css text/xml text/javascript application/javascript application/xml+rss application/json;
```

### 2. 启用 HTTP/2

确保 CDN 和源站都支持 HTTP/2。

### 3. 配置缓存策略

- **静态资源**：长期缓存（30天+）
- **HTML**：不缓存或短缓存（1分钟）
- **API**：不缓存

### 4. 使用 CDN 预热

首次部署后，使用 CDN 预热功能预加载关键资源。

## 常见问题

### Q1: 域名未备案可以使用 CDN 吗？

**答**：**不可以**。国内所有 CDN 服务商都要求域名必须完成 ICP 备案。

**解决方案**：
- ✅ 完成域名备案（推荐，享受国内高速访问）
- ✅ 使用海外 CDN（如 Cloudflare，无需备案）
- ⚠️ 国内访问速度可能受影响

### Q2: 备案需要多长时间？

**答**：通常需要 7-20 个工作日
- 初审：1-3 个工作日
- 管局审核：7-20 个工作日（不同省份时间不同）

### Q3: 备案需要什么材料？

**个人备案**：
- 身份证正反面
- 手机号码
- 邮箱地址

**企业备案**：
- 营业执照
- 法人身份证
- 网站负责人身份证
- 手机号码
- 邮箱地址

### Q4: 备案期间网站可以访问吗？

**答**：**不可以**。备案期间域名无法访问，建议：
- 使用临时域名进行测试
- 或等待备案完成后再上线

### Q5: 更新后看不到新内容？

**原因**：CDN 缓存未刷新

**解决**：
1. 在 CDN 控制台手动刷新缓存
2. 使用 CDN API 自动刷新
3. 确保静态资源文件名包含 hash（Vite 已配置）

### Q6: API 请求也被缓存了？

**解决**：
1. 在 CDN 控制台配置 API 路径不缓存
2. 确保 API 响应头包含 `Cache-Control: no-cache`

### Q7: HTTPS 证书问题？

**解决**：
1. 在 CDN 控制台上传 SSL 证书
2. 或使用 CDN 提供的免费证书（需要验证域名所有权和备案状态）

### Q8: 可以使用海外服务器 + 国内 CDN 吗？

**答**：**可以，但有限制**。
- 域名仍然需要备案（使用国内服务器进行备案）
- CDN 回源到海外服务器可能较慢
- 建议使用国内服务器作为源站

## 监控和优化

### 1. 监控指标

- CDN 命中率（目标：>90%）
- 响应时间（目标：<200ms）
- 带宽使用
- 错误率

### 2. 优化建议

- 定期检查缓存命中率
- 根据访问模式调整缓存规则
- 使用 CDN 日志分析用户访问情况

## 成本优化

1. **选择合适的 CDN 套餐**
   - 按流量计费 vs 按带宽计费
   - 根据实际使用量选择

2. **优化资源大小**
   - 压缩图片
   - 使用 WebP 格式
   - 代码压缩和 Tree Shaking

3. **合理设置缓存**
   - 减少回源请求
   - 降低带宽成本

## 附录：Cloudflare 免备案 CDN 配置示例

### 1. 注册 Cloudflare 账号

1. 访问 https://www.cloudflare.com
2. 注册免费账号
3. 验证邮箱

### 2. 添加网站

1. 登录 Cloudflare 控制台
2. 点击 "Add a Site"
3. 输入您的域名（例如：`yourdomain.com`）
4. 选择免费计划（Free Plan）

### 3. 配置 DNS

Cloudflare 会自动扫描您的 DNS 记录，您需要：
1. 确认 DNS 记录正确
2. 将域名 DNS 服务器改为 Cloudflare 提供的地址
   - 例如：`ns1.cloudflare.com` 和 `ns2.cloudflare.com`
3. 在域名注册商处修改 DNS 服务器

### 4. 配置 SSL/TLS

1. 在 Cloudflare 控制台选择您的域名
2. 进入 "SSL/TLS" 设置
3. 选择 "Full" 或 "Full (strict)" 模式
4. Cloudflare 会自动提供免费 SSL 证书

### 5. 配置缓存规则

1. 进入 "Caching" 设置
2. 配置缓存级别：
   - **Standard**：标准缓存
   - **Aggressive**：激进缓存（推荐静态资源）
3. 配置页面规则：
   - `*.yourdomain.com/assets/*` → 缓存 1 个月
   - `*.yourdomain.com/*.html` → 不缓存

### 6. 配置自动压缩

1. 进入 "Speed" → "Optimization"
2. 启用 "Auto Minify"（自动压缩 JS/CSS/HTML）
3. 启用 "Brotli" 压缩

### 7. 配置 Vite 使用 Cloudflare CDN

创建 `frontend/.env.production`：

```bash
# Cloudflare CDN 域名
VITE_CDN_URL=https://yourdomain.com
# 或使用 Cloudflare 自定义域名
# VITE_CDN_URL=https://cdn.yourdomain.com
```

### 8. Cloudflare 优缺点

**优点：**
- ✅ 完全免费（基础功能）
- ✅ 无需备案
- ✅ 全球加速
- ✅ DDoS 防护
- ✅ 免费 SSL 证书
- ✅ 配置简单

**缺点：**
- ⚠️ 国内访问速度一般
- ⚠️ 免费版功能有限
- ⚠️ 可能被墙（部分地区）

### 9. Cloudflare 替代方案对比

| 服务商 | 免费额度 | 国内速度 | 稳定性 | 推荐度 |
|--------|---------|---------|--------|--------|
| **Cloudflare** | 无限 | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **CDN5** | 按流量 | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **StoneCDN** | 按流量 | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **速盾** | 按流量 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

## 总结

使用 CDN 可以显著提升网站访问速度，特别是对于国内用户。建议采用**方案三（对象存储 + CDN）**，这是最佳实践，既能提升性能，又能降低服务器负载。

### 选择建议

**如果域名可以备案：**
- ✅ 使用国内 CDN（阿里云/腾讯云/七牛云）
- ✅ 享受最佳国内访问速度
- ✅ 价格相对较低

**如果域名无法备案：**
- ✅ **首选 Cloudflare**（免费，配置简单）
- ✅ 备选：CDN5、StoneCDN、速盾等
- ⚠️ 注意国内访问速度可能受影响

如有问题，请参考各 CDN 服务商的官方文档。

