.PHONY: help build up down restart logs status clean test dev install upgrade

# 默认目标
.DEFAULT_GOAL := help

# 颜色定义
GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
BLUE   := $(shell tput -Txterm setaf 4)
RESET  := $(shell tput -Txterm sgr0)

help: ## 显示帮助信息
	@echo '$(BLUE)NodePass License Center - 快捷命令$(RESET)'
	@echo ''
	@echo '$(YELLOW)使用方法:$(RESET)'
	@echo '  make $(GREEN)<target>$(RESET)'
	@echo ''
	@echo '$(YELLOW)可用命令:$(RESET)'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}'

build: ## 构建 Docker 镜像
	@echo "$(BLUE)构建镜像...$(RESET)"
	./scripts/deploy.sh --build-only

build-image: ## 构建并保存镜像文件
	@echo "$(BLUE)构建并保存镜像...$(RESET)"
	./scripts/build-image.sh --save

load-image: ## 从文件加载镜像
	@echo "$(BLUE)加载镜像...$(RESET)"
	./scripts/build-image.sh --load

push-image: ## 推送镜像到仓库
	@echo "$(BLUE)推送镜像...$(RESET)"
	./scripts/build-image.sh --push

multi-arch: ## 构建多架构镜像
	@echo "$(BLUE)构建多架构镜像...$(RESET)"
	./scripts/build-image.sh --multi-arch --platform linux/amd64,linux/arm64

up: ## 启动服务（构建镜像）
	@echo "$(BLUE)启动服务...$(RESET)"
	./scripts/deploy.sh --up

start: ## 启动服务（不重新构建）
	@echo "$(BLUE)启动服务（跳过构建）...$(RESET)"
	./scripts/deploy.sh --up --no-build

down: ## 停止服务
	@echo "$(BLUE)停止服务...$(RESET)"
	./scripts/deploy.sh --down

restart: ## 重启服务
	@echo "$(BLUE)重启服务...$(RESET)"
	./scripts/deploy.sh --restart

logs: ## 查看日志
	./scripts/deploy.sh --logs

status: ## 查看服务状态
	./scripts/deploy.sh --status

clean: ## 清理所有数据（危险）
	@echo "$(YELLOW)⚠️  警告：此操作将删除所有数据！$(RESET)"
	./scripts/deploy.sh --clean

dev: ## 本地开发模式（后端）
	@echo "$(BLUE)启动开发模式...$(RESET)"
	go run ./cmd/server --config ./configs/config.yaml

dev-web: ## 本地开发模式（前端）
	@echo "$(BLUE)启动前端开发服务器...$(RESET)"
	cd web-ui && npm run dev

test: ## 运行测试
	@echo "$(BLUE)运行测试...$(RESET)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)测试完成$(RESET)"

test-coverage: ## 查看测试覆盖率
	@echo "$(BLUE)生成覆盖率报告...$(RESET)"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)覆盖率报告已生成: coverage.html$(RESET)"

install: ## 一键安装（生产环境）
	@echo "$(BLUE)开始安装...$(RESET)"
	bash install.sh --install

upgrade: ## 升级到最新版本
	@echo "$(BLUE)开始升级...$(RESET)"
	bash install.sh --upgrade

uninstall: ## 卸载服务
	@echo "$(YELLOW)开始卸载...$(RESET)"
	bash install.sh --uninstall

lint: ## 代码检查
	@echo "$(BLUE)运行代码检查...$(RESET)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "请先安装 golangci-lint"; exit 1; }
	golangci-lint run ./...

fmt: ## 格式化代码
	@echo "$(BLUE)格式化代码...$(RESET)"
	go fmt ./...
	cd web-ui && npm run format 2>/dev/null || true

deps: ## 安装依赖
	@echo "$(BLUE)安装后端依赖...$(RESET)"
	go mod download
	@echo "$(BLUE)安装前端依赖...$(RESET)"
	cd web-ui && npm install

build-local: ## 本地构建二进制文件
	@echo "$(BLUE)构建本地二进制文件...$(RESET)"
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.appVersion=0.2.0" -o license-center ./cmd/server
	@echo "$(GREEN)构建完成: ./license-center$(RESET)"

build-web: ## 构建前端
	@echo "$(BLUE)构建前端...$(RESET)"
	cd web-ui && npm run build
	@echo "$(GREEN)前端构建完成: web-ui/dist$(RESET)"

docker-push: ## 推送镜像到仓库
	@echo "$(BLUE)推送镜像...$(RESET)"
	docker compose build
	docker tag nodepass/license-center:0.2.0 your-registry/nodepass/license-center:0.2.0
	docker push your-registry/nodepass/license-center:0.2.0

health: ## 检查服务健康状态
	@echo "$(BLUE)检查服务健康...$(RESET)"
	@curl -sf http://127.0.0.1:8090/health && echo "$(GREEN)✓ 服务正常$(RESET)" || echo "$(YELLOW)✗ 服务异常$(RESET)"

ps: ## 查看容器进程
	docker compose ps

exec: ## 进入容器 shell
	docker compose exec license-center /bin/bash

db-shell: ## 进入数据库 shell
	docker compose exec postgres psql -U postgres -d nodepass_license

backup-db: ## 备份数据库
	@echo "$(BLUE)备份数据库...$(RESET)"
	@mkdir -p backups
	docker compose exec -T postgres pg_dump -U postgres nodepass_license > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)备份完成$(RESET)"

restore-db: ## 恢复数据库（需要指定 FILE=backup.sql）
	@if [ -z "$(FILE)" ]; then echo "$(YELLOW)请指定备份文件: make restore-db FILE=backups/backup.sql$(RESET)"; exit 1; fi
	@echo "$(BLUE)恢复数据库...$(RESET)"
	docker compose exec -T postgres psql -U postgres nodepass_license < $(FILE)
	@echo "$(GREEN)恢复完成$(RESET)"
