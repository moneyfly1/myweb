# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# cgo 依赖（mattn/go-sqlite3 需要 gcc 与 musl 头文件）
RUN apk add --no-cache gcc musl-dev

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（SQLite 驱动为 cgo 实现，必须 CGO_ENABLED=1）
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o cboard-go cmd/server/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/cboard-go .

# 暴露端口
EXPOSE 8000

# 运行应用
CMD ["./cboard-go"]
