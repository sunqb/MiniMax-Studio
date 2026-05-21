.PHONY: dev build docker-build docker-run tidy

# 本地开发（需要 .env 文件）
dev:
	go run ./cmd/server

# 本地编译
build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/minimax-studio ./cmd/server

# Docker 构建
docker-build:
	docker build -t minimax-studio:latest .

# Docker 运行（需要 .env 文件）
docker-run:
	docker run --rm -p 8080:8080 --env-file .env minimax-studio:latest

# docker-compose 启动
up:
	docker-compose up -d

# docker-compose 停止
down:
	docker-compose down

# 整理依赖
tidy:
	go mod tidy

# 查看日志
logs:
	docker-compose logs -f minimax-studio
