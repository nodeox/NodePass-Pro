package handlers

import (
	"net/http"
	"strings"

	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuthHandler 认证相关处理器。
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler 创建认证处理器。
func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(db),
	}
}

// Register POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		writeServiceError(c, err, "REGISTER_FAILED")
		return
	}

	result, loginErr := h.authService.Login(&services.LoginRequest{
		Account:  strings.TrimSpace(req.Email),
		Password: req.Password,
	})
	if loginErr != nil {
		utils.SuccessResponse(c, gin.H{"user": user}, "注册成功")
		return
	}

	utils.SuccessResponse(c, result, "注册成功")
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	result, err := h.authService.Login(&req)
	if err != nil {
		writeServiceError(c, err, "LOGIN_FAILED")
		return
	}

	utils.Success(c, result)
}

// Me GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	user, err := h.authService.GetMe(userID)
	if err != nil {
		writeServiceError(c, err, "GET_ME_FAILED")
		return
	}

	utils.Success(c, user)
}

// Refresh POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	token, err := h.authService.RefreshToken(userID)
	if err != nil {
		writeServiceError(c, err, "REFRESH_TOKEN_FAILED")
		return
	}

	utils.Success(c, gin.H{"token": token})
}

// ChangePassword PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		writeServiceError(c, err, "CHANGE_PASSWORD_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "密码修改成功")
}
