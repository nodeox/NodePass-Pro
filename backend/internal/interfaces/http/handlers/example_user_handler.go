package handlers

import (
	"nodepass-pro/backend/internal/application/user/commands"
	"nodepass-pro/backend/internal/application/user/queries"
	"nodepass-pro/backend/internal/domain/user"
	"nodepass-pro/backend/internal/utils"
	
	"github.com/gin-gonic/gin"
)

// ExampleUserHandler 用户处理器示例（展示如何使用新架构）
type ExampleUserHandler struct {
	createUserHandler *commands.CreateUserHandler
	getUserHandler    *queries.GetUserHandler
}

// NewExampleUserHandler 创建用户处理器
func NewExampleUserHandler(
	createUserHandler *commands.CreateUserHandler,
	getUserHandler *queries.GetUserHandler,
) *ExampleUserHandler {
	return &ExampleUserHandler{
		createUserHandler: createUserHandler,
		getUserHandler:    getUserHandler,
	}
}

// CreateUser 创建用户（示例）
func (h *ExampleUserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Role     string `json:"role"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err)
		return
	}
	
	// 构建命令
	cmd := commands.CreateUserCommand{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	
	// 执行命令
	result, err := h.createUserHandler.Handle(c.Request.Context(), cmd)
	if err != nil {
		// 处理领域错误
		switch err {
		case user.ErrEmailExists:
			utils.Error(c, err)
		case user.ErrUsernameExists:
			utils.Error(c, err)
		default:
			utils.Error(c, err)
		}
		return
	}
	
	// 返回结果
	utils.Success(c, gin.H{
		"id":       result.User.ID,
		"username": result.User.Username,
		"email":    result.User.Email,
		"role":     result.User.Role,
		"status":   result.User.Status,
	})
}

// GetUser 获取用户（示例）
func (h *ExampleUserHandler) GetUser(c *gin.Context) {
	// 从认证中间件获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "未登录")
		return
	}
	
	// 构建查询
	query := queries.GetUserQuery{
		UserID: userID.(uint),
	}
	
	// 执行查询
	result, err := h.getUserHandler.Handle(c.Request.Context(), query)
	if err != nil {
		if err == user.ErrUserNotFound {
			utils.NotFound(c, "用户不存在")
		} else {
			utils.Error(c, err)
		}
		return
	}
	
	// 返回结果
	utils.Success(c, gin.H{
		"id":         result.User.ID,
		"username":   result.User.Username,
		"email":      result.User.Email,
		"role":       result.User.Role,
		"status":     result.User.Status,
		"vip_level":  result.User.VipLevel,
		"created_at": result.User.CreatedAt,
	})
}
