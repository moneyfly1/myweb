.PHONY: build run test clean deps

# 构建
build:
	go build -o bin/cboard-go cmd/server/main.go

# 运行
run:
	go run cmd/server/main.go

# 测试
test:
	go test ./...

# 清理
clean:
	rm -rf bin/
	rm -f *.db *.log

# 安装依赖
deps:
	go mod download
	go mod tidy

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

