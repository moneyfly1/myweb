# ============================================================
# CBoard Docker 镜像（三阶段：后端构建 + 前端构建 + 运行）
#
# 实测修复记录（干净 Debian 12 + Docker 20.10.24 上真实构建验证）：
#   1) 原 `FROM golang:1.24-alpine` 与 go.mod 的 `go 1.25.0` 不匹配，而官方
#      golang 镜像把 GOTOOLCHAIN 锁成 local（不允许自动下载工具链），
#      于是构建第 5 步就失败：
#        go: go.mod requires go >= 1.25.0 (running go 1.24.13; GOTOOLCHAIN=local)
#      现改为 golang:1.25-alpine（与 go.mod 完全一致）。
#   2) 原 `node:20-alpine` 正好卡在 Vite 7 的要求下界（^20.19.0 || >=22.12.0），
#      现改为 node:22-alpine（与一键脚本安装的 Node 22.12 对齐，避免浮动 tag 漂移）。
#   3) 前端优先按 lockfile 做可复现安装（npm ci），失败再回落 npm install。
#   4) 运行镜像补上 /root/data、/root/uploads 目录，并加 HEALTHCHECK（/health）。
#   5) alpine/musl 下 go-sqlite3 需要 CGO_CFLAGS="-D_LARGEFILE64_SOURCE"，
#      否则 sqlite3-binding.c 报 "'pread64' undeclared / unknown type name 'off64_t'"。
#   6) 新增 GOPROXY / NPM_REGISTRY 两个 build-arg（默认官方源），国内构建可切镜像加速：
#      docker build --build-arg GOPROXY=https://goproxy.cn,direct \
#                   --build-arg NPM_REGISTRY=https://registry.npmmirror.com -t cboard .
#
# GeoIP：GeoLite2-City.mmdb 有 62MB，.dockerignore 已排除，不打进镜像；
# 应用会在 ./data/GeoLite2-City.mmdb（即宿主 ./data 挂载目录）按需下载并持久化。
# ============================================================

# ---------- 阶段 1：后端构建 ----------
FROM golang:1.25-alpine AS builder

# 国内网络可用 --build-arg GOPROXY=https://goproxy.cn,direct 加速模块下载
ARG GOPROXY=https://proxy.golang.org,direct

WORKDIR /app

# cgo 依赖（mattn/go-sqlite3 需要 gcc 与 musl 头文件）
RUN apk add --no-cache gcc musl-dev

# 复制 go mod 文件（利用层缓存）
COPY go.mod go.sum ./
RUN go env -w GOPROXY="${GOPROXY}" && go mod download

# 复制源代码
COPY . .

# 构建应用（SQLite 驱动为 cgo 实现，必须 CGO_ENABLED=1）
# CGO_CFLAGS 里的 _LARGEFILE64_SOURCE 是 alpine/musl 下的必需项：musl 默认不声明
# pread64/pwrite64/off64_t，go-sqlite3 的 sqlite3-binding.c 会直接编译失败（实测报错：
# "error: 'pread64' undeclared here / unknown type name 'off64_t'"）。
RUN CGO_ENABLED=1 GOOS=linux CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    go build -trimpath -ldflags="-s -w" -o cboard-go cmd/server/main.go

# ---------- 阶段 2：前端构建 ----------
FROM node:22-alpine AS frontend-builder

# 国内网络可用 --build-arg NPM_REGISTRY=https://registry.npmmirror.com 加速依赖安装
ARG NPM_REGISTRY=https://registry.npmjs.org

WORKDIR /app/frontend

# 复制前端依赖清单（利用层缓存）
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm config set registry "${NPM_REGISTRY}" && \
    (if [ -f package-lock.json ]; then \
        npm ci --legacy-peer-deps || npm install --legacy-peer-deps; \
    else \
        npm install --legacy-peer-deps; \
    fi)

# 复制前端源码并构建
COPY frontend/ .
# VITE_API_BASE_URL 为空时前端与后端同域（同容器部署默认即可）
RUN npm run build

# ---------- 阶段 3：运行 ----------
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl libgcc
ENV TZ=Asia/Shanghai

WORKDIR /root/

# 复制后端二进制
COPY --from=builder /app/cboard-go .

# 复制前端构建产物（后端从 ./frontend/dist 提供静态文件；缺了会整站 404）
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# 数据与上传目录（compose 会把宿主目录挂到这里）
RUN mkdir -p /root/data /root/uploads

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8000/health || exit 1

CMD ["./cboard-go"]
