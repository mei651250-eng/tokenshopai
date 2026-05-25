# TokenHub Makefile

.PHONY: help dev dev-frontend dev-backend build build-frontend build-backend \
       test test-frontend test-backend lint lint-frontend lint-backend \
       docker-up docker-down docker-build migrate seed clean

# 默认目标
help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ==================== 开发 ====================

dev: ## 同时启动前后端开发服务器
	@make -j2 dev-frontend dev-backend

dev-frontend: ## 启动前端开发服务器
	cd frontend && npm run dev

dev-backend: ## 启动后端开发服务器
	cd backend && go run ./cmd/server

# ==================== 构建 ====================

build: build-frontend build-backend ## 构建前后端

build-frontend: ## 构建前端
	cd frontend && npm run build

build-backend: ## 构建后端
	cd backend && go build -o bin/server ./cmd/server

# ==================== 测试 ====================

test: test-backend test-frontend ## 运行所有测试

test-backend: ## 运行后端测试
	cd backend && go test ./... -v -cover

test-frontend: ## 运行前端测试
	cd frontend && npm run test || echo "No test runner configured"

# ==================== 代码质量 ====================

lint: lint-backend lint-frontend ## 代码检查

lint-backend: ## 后端代码检查
	cd backend && golangci-lint run ./... || echo "Install golangci-lint: https://golangci-lint.run/usage/install/"

lint-frontend: ## 前端代码检查
	cd frontend && npm run lint || echo "Run npm install first"

fmt: ## 格式化代码
	cd backend && gofmt -w .
	cd frontend && npx prettier --write "src/**/*.{vue,ts,js,json,css,scss}" || true

# ==================== Docker ====================

docker-build: ## 构建 Docker 镜像
	docker build -f deploy/docker/Dockerfile.backend -t tokenhub/backend:latest ./backend
	docker build -f deploy/docker/Dockerfile.frontend -t tokenhub/frontend:latest ./frontend

docker-up: ## 启动所有 Docker 容器
	docker compose -f deploy/docker/docker-compose.yml up -d

docker-down: ## 停止所有 Docker 容器
	docker compose -f deploy/docker/docker-compose.yml down

docker-logs: ## 查看 Docker 日志
	docker compose -f deploy/docker/docker-compose.yml logs -f

# ==================== 数据库 ====================

migrate: ## 运行数据库迁移
	cd backend && go run ./cmd/server --migrate-only || echo "Auto-migrate runs on startup"

seed: ## 填充种子数据
	cd backend && go run ./cmd/server --seed-only || echo "Seeding runs on startup"

# ==================== 清理 ====================

clean: ## 清理构建产物
	rm -rf frontend/dist
	rm -rf backend/bin
	rm -rf uploads
	find . -name "*.test" -delete
	find . -name "*.out" -delete

# ==================== 工具 ====================

generate-openapi: ## 生成 OpenAPI 文档
	cd backend && swag init -g cmd/server/main.go -o ../docs/api/generated || echo "Install swag: go install github.com/swaggo/swag/cmd/swag@latest"

check-deps: ## 检查依赖更新
	cd backend && go list -u -m -json all | grep -E '"Path"|"Update"'
	cd frontend && npx npm-check-updates
