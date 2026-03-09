package handlers

import (
	"net/http"

	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证接口。
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 登录。
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	token, user, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		utils.Error(c, http.StatusUnauthorized, "LOGIN_FAILED", err.Error())
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	}, "登录成功")
}

// Me 当前管理员信息。
func (h *AuthHandler) Me(c *gin.Context) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未登录")
		return
	}

	user, err := h.authService.GetAdminByID(adminID.(uint))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "管理员不存在")
		return
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	}, "ok")
}
