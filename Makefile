# Makefile for NodePass-Pro

.PHONY: help
help: ## 显示帮助信息
	@echo "NodePass-Pro 开发工具"
	@echo ""
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================
# 测试相关命令
# ============================================

.PHONY: test
test: ## 运行后端所有测试
	@echo "运行后端测试..."
	cd backend && go test -v ./internal/...

.PHONY: test-coverage
test-coverage: ## 生成后端测试覆盖率报告
	@echo "生成测试覆盖率报告..."
	cd backend && go test -v -coverprofile=coverage.out ./internal/...
	cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: backend/coverage.html"

.PHONY: test-coverage-summary
test-coverage-summary: ## 显示测试覆盖率摘要
	@echo "测试覆盖率摘要:"
	cd backend && go test -coverprofile=coverage.out ./internal/... > /dev/null 2>&1
	cd backend && go tool cover -func=coverage.out | grep total

.PHONY: test-race
test-race: ## 运行竞态检测测试
	@echo "运行竞态检测测试..."
	cd backend && go test -v -race ./internal/...

.PHONY: test-short
test-short: ## 运行快速测试（跳过耗时测试）
	@echo "运行快速测试..."
	cd backend && go test -v -short ./internal/...

.PHONY: test-verbose
test-verbose: ## 运行详细测试输出
	@echo "运行详细测试..."
	cd backend && go test -v -cover ./internal/... 2>&1 | tee test-output.log

.PHONY: test-services
test-services: ## 仅测试服务层
	@echo "测试服务层..."
	cd backend && go test -v -cover ./internal/services/...

.PHONY: test-middleware
test-middleware: ## 仅测试中间件
	@echo "测试中间件..."
	cd backend && go test -v -cover ./internal/middleware/...

.PHONY: test-handlers
test-handlers: ## 仅测试处理器
	@echo "测试处理器..."
	cd backend && go test -v -cover ./internal/handlers/...

.PHONY: test-frontend
test-frontend: ## 运行前端测试
	@echo "运行前端测试..."
	cd frontend && npm test

.PHONY: test-frontend-coverage
test-frontend-coverage: ## 生成前端测试覆盖率报告
	@echo "生成前端测试覆盖率报告..."
	cd frontend && npm test -- --coverage

.PHONY: test-integration
test-integration: ## 运行集成测试
	@echo "运行集成测试..."
	chmod +x tests/integration_test.sh
	./tests/integration_test.sh

.PHONY: test-all
test-all: test test-frontend test-integration ## 运行所有测试

.PHONY: test-watch
test-watch: ## 监听模式运行测试
	@echo "监听模式运行测试..."
	cd backend && go test -v ./internal/... -count=1 -timeout=30s

.PHONY: test-benchmark
test-benchmark: ## 运行性能基准测试
	@echo "运行性能基准测试..."
	cd backend && go test -bench=. -benchmem ./internal/...

# ============================================
# 代码质量检查
# ============================================

.PHONY: lint
lint: ## 运行代码检查
	@echo "运行后端代码检查..."
	cd backend && golangci-lint run ./...
	@echo "运行前端代码检查..."
	cd frontend && npm run lint

.PHONY: fmt
fmt: ## 格式化代码
	@echo "格式化后端代码..."
	cd backend && go fmt ./...
	@echo "格式化前端代码..."
	cd frontend && npm run lint -- --fix

.PHONY: vet
vet: ## 运行 go vet
	@echo "运行 go vet..."
	cd backend && go vet ./...

.PHONY: check
check: fmt vet lint test ## 运行所有检查

# ============================================
# 构建相关命令
# ============================================

.PHONY: build
build: ## 构建后端和前端
	@echo "构建后端..."
	cd backend && go build -o server ./cmd/server
	@echo "构建前端..."
	cd frontend && npm run build

.PHONY: build-backend
build-backend: ## 仅构建后端
	@echo "构建后端..."
	cd backend && go build -o server ./cmd/server

.PHONY: build-frontend
build-frontend: ## 仅构建前端
	@echo "构建前端..."
	cd frontend && npm run build

.PHONY: build-nodeclient
build-nodeclient: ## 构建节点客户端
	@echo "构建节点客户端..."
	./scripts/build-nodeclient-downloads.sh

# ============================================
# 开发相关命令
# ============================================

.PHONY: dev
dev: ## 启动开发环境
	@echo "启动开发环境..."
	docker compose up -d postgres redis
	@echo "等待数据库启动..."
	sleep 3
	@echo "启动后端..."
	cd backend && go run ./cmd/server/main.go &
	@echo "启动前端..."
	cd frontend && npm run dev

.PHONY: dev-backend
dev-backend: ## 仅启动后端开发服务器
	@echo "启动后端开发服务器..."
	cd backend && go run ./cmd/server/main.go

.PHONY: dev-frontend
dev-frontend: ## 仅启动前端开发服务器
	@echo "启动前端开发服务器..."
	cd frontend && npm run dev

.PHONY: install
install: ## 安装依赖
	@echo "安装后端依赖..."
	cd backend && go mod download
	@echo "安装前端依赖..."
	cd frontend && npm install

.PHONY: clean
clean: ## 清理构建产物
	@echo "清理构建产物..."
	rm -f backend/server
	rm -f backend/coverage.out
	rm -f backend/coverage.html
	rm -rf frontend/dist
	rm -rf frontend/coverage
	rm -f backend/test-output.log

# ============================================
# Docker 相关命令
# ============================================

.PHONY: docker-up
docker-up: ## 启动 Docker 服务
	docker compose up -d

.PHONY: docker-down
docker-down: ## 停止 Docker 服务
	docker compose down

.PHONY: docker-logs
docker-logs: ## 查看 Docker 日志
	docker compose logs -f

.PHONY: docker-build
docker-build: ## 构建 Docker 镜像
	docker compose build

.PHONY: docker-restart
docker-restart: docker-down docker-up ## 重启 Docker 服务

# ============================================
# 数据库相关命令
# ============================================

.PHONY: db-migrate
db-migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	cd backend && go run ./cmd/migrate/main.go up

.PHONY: db-rollback
db-rollback: ## 回滚数据库迁移
	@echo "回滚数据库迁移..."
	cd backend && go run ./cmd/migrate/main.go down

.PHONY: db-reset
db-reset: ## 重置数据库
	@echo "重置数据库..."
	docker compose down -v
	docker compose up -d postgres redis
	sleep 3
	$(MAKE) db-migrate

# ============================================
# 部署相关命令
# ============================================

.PHONY: deploy
deploy: ## 部署到生产环境
	@echo "部署到生产环境..."
	./scripts/deploy.sh

.PHONY: deploy-dev
deploy-dev: ## 部署到开发环境
	@echo "部署到开发环境..."
	./scripts/deploy.sh --env dev

# ============================================
# 工具命令
# ============================================

.PHONY: version
version: ## 显示版本信息
	@cat VERSION

.PHONY: version-check
version-check: ## 检查版本一致性
	@./check-version.sh

.PHONY: version-sync
version-sync: ## 同步所有组件版本
	@./sync-version.sh

.PHONY: version-info
version-info: ## 显示详细版本信息
	@echo "=========================================="
	@echo "NodePass-Pro 版本信息"
	@echo "=========================================="
	@echo ""
	@echo "根目录版本: $$(cat VERSION)"
	@echo "后端版本: $$(grep 'var Version' backend/internal/version/version.go | sed 's/.*\"\(.*\)\".*/\1/')"
	@echo "前端版本: $$(grep '\"version\"' frontend/package.json | head -1 | sed 's/.*: \"\(.*\)\".*/\1/')"
	@echo "节点客户端版本: $$(grep 'var clientVersion' nodeclient/internal/agent/agent.go | sed 's/.*\"\(.*\)\".*/\1/')"
	@echo "授权中心版本: $$(grep '\"version\"' license-center/web-ui/package.json | head -1 | sed 's/.*: \"\(.*\)\".*/\1/')"
	@echo ""

.PHONY: version-bump
version-bump: ## 升级版本号
	@./sync-version.sh

.PHONY: license-verify
license-verify: ## 验证授权
	@python3 scripts/license-verify.py

.PHONY: generate-jwt-secret
generate-jwt-secret: ## 生成 JWT 密钥
	@echo "生成 JWT 密钥:"
	@openssl rand -base64 48

# ============================================
# CI/CD 相关命令
# ============================================

.PHONY: ci-test
ci-test: ## CI 环境测试
	@echo "运行 CI 测试..."
	$(MAKE) test-coverage
	$(MAKE) test-race
	$(MAKE) test-frontend-coverage

.PHONY: ci-lint
ci-lint: ## CI 环境代码检查
	@echo "运行 CI 代码检查..."
	$(MAKE) lint

.PHONY: ci-build
ci-build: ## CI 环境构建
	@echo "运行 CI 构建..."
	$(MAKE) build

.PHONY: ci
ci: ci-lint ci-test ci-build ## 运行完整 CI 流程

# ============================================
# 文档相关命令
# ============================================

.PHONY: docs
docs: ## 生成 API 文档
	@echo "生成 API 文档..."
	cd backend && swag init -g cmd/server/main.go

.PHONY: docs-serve
docs-serve: ## 启动文档服务器
	@echo "启动文档服务器..."
	cd docs && python3 -m http.server 8000

# ============================================
# 默认命令
# ============================================

.DEFAULT_GOAL := help
