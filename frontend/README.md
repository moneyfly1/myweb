# 前端项目

这是 CBoard 的前端项目，使用 Vue 3 + Element Plus + Vite 构建。

## 🚀 快速开始

### 1. 安装依赖

```bash
npm install
```

### 2. 配置环境变量

复制 `.env.example` 为 `.env`（已自动创建）：

```bash
cp .env.example .env
```

编辑 `.env` 文件，设置 API 基础 URL：

```env
VITE_API_BASE_URL=http://localhost:8000
```

### 3. 启动开发服务器

```bash
npm run dev
```

前端将在 http://localhost:5173 启动。

### 4. 构建生产版本

```bash
npm run build
```

构建后的文件将输出到 `dist/` 目录。

## 📝 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Element Plus** - Vue 3 UI 组件库
- **Vite** - 下一代前端构建工具
- **Pinia** - Vue 状态管理
- **Vue Router** - Vue 官方路由
- **Axios** - HTTP 客户端
- **Chart.js** - 图表库
- **dayjs** - 日期处理库

## 🔧 开发说明

### API 配置

前端通过 `/api/v1` 访问后端 API，开发环境下通过 Vite 代理转发到 Go 后端。

### 环境变量

- `VITE_API_BASE_URL`: API 基础 URL（开发环境默认: http://localhost:8000）

### 项目结构

```
frontend/
├── src/
│   ├── components/     # 组件
│   ├── views/          # 页面
│   ├── router/         # 路由配置
│   ├── store/          # 状态管理
│   ├── utils/          # 工具函数
│   └── styles/         # 样式文件
├── public/             # 静态资源
└── dist/               # 构建输出
```

## 📦 构建和部署

### 开发环境

```bash
npm run dev
```

### 生产环境

1. **构建前端**
```bash
npm run build
```

2. **Go 后端会自动提供静态文件服务**

Go 后端已经配置了静态文件服务，构建后的前端文件会自动被 Go 后端提供。

访问 http://localhost:8000 即可看到前端页面。

## 🔗 与后端集成

前端通过以下方式与 Go 后端集成：

1. **开发环境**: Vite 代理 `/api` 请求到 `http://localhost:8000`
2. **生产环境**: Go 后端提供静态文件服务和 API 服务

## 📚 更多信息

- 后端 API 文档: 查看 `../README.md`
- 前端集成指南: 查看 `../FRONTEND_INTEGRATION.md`

