package handlers

import (
	"net/http"
	"strings"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/services"
	"nodepass-pro/backend/internal/utils"

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

// SendEmailChangeCode POST /api/v1/auth/email/code
func (h *AuthHandler) SendEmailChangeCode(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		Password       string `json:"password"`
		NewEmail       string `json:"new_email"`
		NewEmailAlt    string `json:"newEmail"`
		NewEmailLegacy string `json:"email"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.NewEmail) == "" {
		req.NewEmail = strings.TrimSpace(req.NewEmailAlt)
	}
	if strings.TrimSpace(req.NewEmail) == "" {
		req.NewEmail = strings.TrimSpace(req.NewEmailLegacy)
	}

	result, err := h.authService.SendEmailChangeCode(userID, req.Password, req.NewEmail)
	if err != nil {
		writeServiceError(c, err, "SEND_EMAIL_CODE_FAILED")
		return
	}

	// 仅非生产环境返回调试验证码，便于联调。
	if cfg := config.GlobalConfig; cfg != nil && strings.EqualFold(cfg.Server.Mode, "release") {
		result.DebugCode = ""
	}
	if result.Sent {
		utils.SuccessResponse(c, result, "验证码已发送")
		return
	}
	utils.SuccessResponse(c, result, "SMTP 未启用，已生成调试验证码")
}

// ChangeEmail PUT /api/v1/auth/email
func (h *AuthHandler) ChangeEmail(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	type requestPayload struct {
		NewEmail       string `json:"new_email"`
		NewEmailAlt    string `json:"newEmail"`
		NewEmailLegacy string `json:"email"`
		Code           string `json:"code"`
		CodeAlt        string `json:"verify_code"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}
	if strings.TrimSpace(req.NewEmail) == "" {
		req.NewEmail = strings.TrimSpace(req.NewEmailAlt)
	}
	if strings.TrimSpace(req.NewEmail) == "" {
		req.NewEmail = strings.TrimSpace(req.NewEmailLegacy)
	}
	if strings.TrimSpace(req.Code) == "" {
		req.Code = strings.TrimSpace(req.CodeAlt)
	}

	if err := h.authService.ChangeEmail(userID, req.NewEmail, req.Code); err != nil {
		writeServiceError(c, err, "CHANGE_EMAIL_FAILED")
		return
	}
	utils.SuccessResponse(c, nil, "邮箱修改成功")
}

// LoginV2 POST /api/v1/auth/login/v2
// 新版登录接口，返回 access token 和 refresh token
func (h *AuthHandler) LoginV2(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.authService.LoginWithRefreshToken(&req, ipAddress, userAgent)
	if err != nil {
		writeServiceError(c, err, "LOGIN_FAILED")
		return
	}

	utils.SuccessResponse(c, result, "登录成功")
}

// RefreshToken POST /api/v1/auth/refresh/v2
// 使用 refresh token 刷新 access token
func (h *AuthHandler) RefreshTokenV2(c *gin.Context) {
	type requestPayload struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	result, err := h.authService.RefreshAccessToken(req.RefreshToken, ipAddress, userAgent)
	if err != nil {
		writeServiceError(c, err, "REFRESH_TOKEN_FAILED")
		return
	}

	utils.SuccessResponse(c, result, "刷新成功")
}

// Logout POST /api/v1/auth/logout
// 登出（撤销 refresh token）
func (h *AuthHandler) Logout(c *gin.Context) {
	type requestPayload struct {
		RefreshToken string `json:"refresh_token"`
	}
	var req requestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有提供 refresh token，也算登出成功
		utils.SuccessResponse(c, nil, "登出成功")
		return
	}

	if req.RefreshToken != "" {
		if err := h.authService.RevokeRefreshToken(req.RefreshToken); err != nil {
			// 撤销失败也不影响登出
			utils.SuccessResponse(c, nil, "登出成功")
			return
		}
	}

	utils.SuccessResponse(c, nil, "登出成功")
}

// RevokeAllTokens POST /api/v1/auth/revoke-all
// 撤销当前用户的所有 refresh tokens
func (h *AuthHandler) RevokeAllTokens(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	if err := h.authService.RevokeAllUserTokens(userID); err != nil {
		writeServiceError(c, err, "REVOKE_TOKENS_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "已撤销所有登录会话")
}
