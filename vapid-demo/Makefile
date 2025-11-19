.PHONY: keys run build clean

# 生成 VAPID 密钥
keys:
	@echo "生成 VAPID 密钥..."
	@go run cmd/generate-keys.go

# 运行服务器（需要先设置环境变量）
run:
	@echo "启动服务器..."
	@go run main.go

# 构建可执行文件
build:
	@echo "构建项目..."
	@go build -o vapid-demo main.go

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -f vapid-demo
	@go clean

# 安装依赖
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy

