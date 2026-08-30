# ============================================================
# CBoard Docker 镜像（三阶段：后端 + 前端 + 运行）
# 修复：原 Dockerfile 只复制后端二进制，未构建/复制前端 dist，
#       后端从 ./frontend/dist 提供静态文件 → 容器内前端全部 404。
# ============================================================

# ---------- 阶段 1：后端构建 ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# cgo 依赖（mattn/go-sqlite3 需要 gcc 与 musl 头文件）
RUN apk add --no-cache gcc musl-dev

# 复制 go mod 文件（利用层缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（SQLite 驱动为 cgo 实现，必须 CGO_ENABLED=1）
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o cboard-go cmd/server/main.go

# ---------- 阶段 2：前端构建 ----------
# Vite 7 需要 Node 20+（node:18 会构建失败）
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# 复制前端依赖清单（利用层缓存）
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --legacy-peer-deps

# 复制前端源码并构建
COPY frontend/ .
# VITE_API_BASE_URL 为空时前端与后端同域（同容器部署默认即可）
RUN npm run build

# ---------- 阶段 3：运行 ----------
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /root/

# 复制后端二进制
COPY --from=builder /app/cboard-go .

# 复制前端构建产物（后端从 ./frontend/dist 提供静态文件）
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# 暴露端口
EXPOSE 8000

# 运行应用
CMD ["./cboard-go"]
