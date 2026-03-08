# NodePass-Pro API 文档配置指南

本文档说明如何为 NodePass-Pro 项目添加 Swagger/OpenAPI 文档支持。

---

## 📋 安装依赖

### 后端 (Go)

```bash
cd backend

# 安装 swag CLI 工具
go install github.com/swaggo/swag/cmd/swag@latest

# 添加依赖到项目
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

### 更新 go.mod

在 `backend/go.mod` 中添加：

```go
require (
    github.com/swaggo/gin-swagger v1.6.0
    github.com/swaggo/files v1.0.1
    github.com/swaggo/swag v1.16.2
)
```

---

## 🔧 配置 Swagger

### 1. 在 main.go 中添加 Swagger 注释

在 `backend/cmd/server/main.go` 文件顶部添加：

```go
// @title NodePass Pro API
// @version 1.0
// @description TCP/UDP 流量转发管理系统 API 文档
// @termsOfService https://nodepass.pro/terms

// @contact.name API Support
// @contact.url https://nodepass.pro/support
// @contact.email support@nodepass.pro

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey CSRFToken
// @in header
// @name X-CSRF-Token
// @description CSRF protection token
```

### 2. 在 setupRouter 中注册 Swagger 路由

```go
import (
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "nodepass-pro/backend/docs" // 导入生成的文档
)

func setupRouter(licenseManager *license.Manager) (*gin.Engine, *panelws.Hub) {
    r := gin.New()
    // ... 其他配置

    // Swagger 文档路由
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // ... 其他路由
    return r, wsHub
}
```

---

## 📝 为 API 端点添加文档注释

### 示例：认证 API

在 `backend/internal/handlers/auth_handler.go` 中添加：

```go
// Register godoc
// @Summary 用户注册
// @Description 注册新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body services.RegisterRequest true "注册信息"
// @Success 200 {object} map[string]interface{} "注册成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 409 {object} map[string]interface{} "用户名或邮箱已存在"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
    // ... 实现代码
}

// LoginV2 godoc
// @Summary 用户登录 (V2)
// @Description 使用用户名或邮箱登录，返回访问令牌和刷新令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body services.LoginRequest true "登录信息"
// @Success 200 {object} services.LoginResult "登录成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "密码错误"
// @Failure 404 {object} map[string]interface{} "用户不存在"
// @Router /auth/login/v2 [post]
func (h *AuthHandler) LoginV2(c *gin.Context) {
    // ... 实现代码
}

// Me godoc
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security CSRFToken
// @Success 200 {object} models.User "用户信息"
// @Failure 401 {object} map[string]interface{} "未认证"
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
    // ... 实现代码
}

// ChangePassword godoc
// @Summary 修改密码
// @Description 修改当前用户的登录密码
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security CSRFToken
// @Param request body object{old_password=string,new_password=string} true "密码信息"
// @Success 200 {object} map[string]interface{} "密码修改成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "旧密码错误"
// @Router /auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
    // ... 实现代码
}

// RefreshTokenV2 godoc
// @Summary 刷新访问令牌 (V2)
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body object{refresh_token=string} true "刷新令牌"
// @Success 200 {object} map[string]interface{} "刷新成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "刷新令牌无效或已过期"
// @Router /auth/refresh/v2 [post]
func (h *AuthHandler) RefreshTokenV2(c *gin.Context) {
    // ... 实现代码
}
```

### 示例：隧道 API

```go
// Create godoc
// @Summary 创建隧道
// @Description 创建新的流量转发隧道
// @Tags 隧道
// @Accept json
// @Produce json
// @Security BearerAuth
// @Security CSRFToken
// @Param request body services.CreateTunnelRequest true "隧道配置"
// @Success 200 {object} models.Tunnel "创建成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 401 {object} map[string]interface{} "未认证"
// @Router /tunnels [post]
func (h *TunnelHandler) Create(c *gin.Context) {
    // ... 实现代码
}

// List godoc
// @Summary 隧道列表
// @Description 获取用户的隧道列表
// @Tags 隧道
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "状态过滤 (running/stopped)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{} "隧道列表"
// @Failure 401 {object} map[string]interface{} "未认证"
// @Router /tunnels [get]
func (h *TunnelHandler) List(c *gin.Context) {
    // ... 实现代码
}

// Get godoc
// @Summary 获取隧道详情
// @Description 获取指定隧道的详细信息
// @Tags 隧道
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "隧道 ID"
// @Success 200 {object} models.Tunnel "隧道详情"
// @Failure 401 {object} map[string]interface{} "未认证"
// @Failure 404 {object} map[string]interface{} "隧道不存在"
// @Router /tunnels/{id} [get]
func (h *TunnelHandler) Get(c *gin.Context) {
    // ... 实现代码
}
```

---

## 🚀 生成文档

### 1. 生成 Swagger 文档

```bash
cd backend

# 生成文档（会在 backend/docs 目录生成文件）
swag init -g cmd/server/main.go -o docs

# 或者添加到 Makefile
make swagger
```

### 2. 在 Makefile 中添加命令

```makefile
.PHONY: swagger
swagger: ## 生成 Swagger API 文档
	@echo "生成 Swagger 文档..."
	cd backend && swag init -g cmd/server/main.go -o docs
	@echo "文档已生成: backend/docs/"

.PHONY: swagger-serve
swagger-serve: ## 启动服务并打开 Swagger 文档
	@echo "启动服务..."
	@echo "Swagger 文档地址: http://localhost:8080/swagger/index.html"
	cd backend && go run cmd/server/main.go
```

---

## 📖 访问文档

### 启动服务后访问

```bash
# 启动后端服务
make dev-backend

# 或
cd backend && go run cmd/server/main.go
```

### 浏览器访问

```
http://localhost:8080/swagger/index.html
```

---

## 📋 Swagger 注释语法参考

### 通用注释

```go
// @Summary 简短描述（必填）
// @Description 详细描述
// @Tags 标签分组
// @Accept json|xml|plain|html|mpfd
// @Produce json|xml|plain|html|mpfd
// @Security BearerAuth  // 需要认证
```

### 参数注释

```go
// @Param name path string true "路径参数"
// @Param name query string false "查询参数" default(value)
// @Param name header string true "请求头参数"
// @Param name body Type true "请求体"
// @Param name formData file true "文件上传"
```

### 响应注释

```go
// @Success 200 {object} Type "成功响应"
// @Failure 400 {object} Type "错误响应"
// @Router /path [get]
```

### 数据类型

```go
// 基本类型
string, integer, number, boolean, array, object

// 自定义类型
models.User
services.LoginRequest
map[string]interface{}

// 数组
[]models.User
[]string

// 嵌套对象
object{field1=string,field2=int}
```

---

## 🎯 最佳实践

### 1. 组织 API 标签

```go
// 认证相关
// @Tags 认证

// 用户管理
// @Tags 用户

// 隧道管理
// @Tags 隧道

// 节点管理
// @Tags 节点

// VIP 管理
// @Tags VIP

// 系统管理
// @Tags 系统
```

### 2. 统一响应格式

定义通用响应结构：

```go
// APIResponse 通用 API 响应
type APIResponse struct {
    Success bool        `json:"success" example:"true"`
    Message string      `json:"message,omitempty" example:"操作成功"`
    Data    interface{} `json:"data,omitempty"`
    Code    string      `json:"code,omitempty" example:"SUCCESS"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
    Success bool   `json:"success" example:"false"`
    Message string `json:"message" example:"操作失败"`
    Code    string `json:"code" example:"ERROR_CODE"`
}
```

### 3. 添加示例值

```go
type LoginRequest struct {
    Account  string `json:"account" binding:"required" example:"user@example.com"`
    Password string `json:"password" binding:"required" example:"Password123!"`
}
```

### 4. 文档版本控制

```go
// @version 1.0
// @version 2.0  // 新版本
```

---

## 🔄 CI/CD 集成

### GitHub Actions

在 `.github/workflows/docs.yml` 中添加：

```yaml
name: API Documentation

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  generate-docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install swag
        run: go install github.com/swaggo/swag/cmd/swag@latest

      - name: Generate Swagger docs
        run: |
          cd backend
          swag init -g cmd/server/main.go -o docs

      - name: Check for changes
        run: |
          git diff --exit-code backend/docs/ || (echo "Swagger docs need to be regenerated" && exit 1)
```

---

## 📚 相关资源

- [Swag 官方文档](https://github.com/swaggo/swag)
- [Gin Swagger](https://github.com/swaggo/gin-swagger)
- [OpenAPI 规范](https://swagger.io/specification/)
- [Swagger UI](https://swagger.io/tools/swagger-ui/)

---

## ✅ 检查清单

完成 API 文档配置后，确保：

- [ ] 安装了 swag CLI 工具
- [ ] 在 main.go 中添加了全局注释
- [ ] 注册了 Swagger 路由
- [ ] 为主要 API 端点添加了文档注释
- [ ] 生成了 Swagger 文档
- [ ] 可以访问 Swagger UI
- [ ] 文档内容准确完整
- [ ] 添加了 Makefile 命令
- [ ] 配置了 CI/CD 自动生成

---

**下一步**: 为所有 Handler 添加完整的 Swagger 注释，提升 API 文档覆盖率到 100%。
