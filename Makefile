.PHONY: build run test test-all clean deps

# 运行
run:
	go run cmd/server/main.go

# 测试
test:
	@echo "运行核心模块测试..."
	go test ./cmd/... ./internal/...

# 兼容旧命令
test-all:
	@$(MAKE) test

# 清理
clean:
	rm -rf bin/
	rm -f *.db *.log

# 安装依赖
deps:
	go mod download
	go mod tidy

# 下载 GeoIP 数据库
geoip:
	@echo "正在下载 GeoIP 数据库..."
	@go run scripts/download_geoip.go .

# 构建（包含下载 GeoIP）
build: geoip
	go build -o bin/cboard-go cmd/server/main.go

# 修复依赖（生成 go.sum）
fix-deps:
	@echo "正在下载依赖..."
	go mod download
	@echo "正在整理依赖..."
	go mod tidy
	@echo "✅ 依赖已修复"
	@ls -lh go.sum 2>&1 || echo "⚠️  go.sum 文件未生成"

# 格式化代码
fmt:
	go fmt ./...

# 代码检查
lint:
	golangci-lint run

# 数据库迁移
migrate:
	go run cmd/server/main.go migrate
