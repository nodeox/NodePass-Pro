package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"nodepass-panel/backend/internal/models"
	"nodepass-panel/backend/internal/services"
	"nodepass-panel/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TelegramHandler Telegram Bot 与 Widget 登录处理器。
type TelegramHandler struct {
	telegramService *services.TelegramService
}

// NewTelegramHandler 创建 Telegram 处理器。
func NewTelegramHandler(db *gorm.DB) *TelegramHandler {
	return &TelegramHandler{
		telegramService: services.NewTelegramService(db),
	}
}

// InitBot 初始化 Telegram Bot。
func (h *TelegramHandler) InitBot(botToken string) error {
	return h.telegramService.InitBot(botToken)
}

// Webhook POST /api/v1/telegram/webhook
func (h *TelegramHandler) Webhook(c *gin.Context) {
	var update services.TelegramUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	if err := h.telegramService.HandleCommand(update); err != nil {
		writeServiceError(c, err, "TELEGRAM_WEBHOOK_FAILED")
		return
	}

	utils.SuccessResponse(c, gin.H{"processed": true}, "webhook 处理成功")
}

// Bind POST /api/v1/telegram/bind
func (h *TelegramHandler) Bind(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	email, code, expiresAt, err := h.telegramService.CreateBindCode(userID)
	if err != nil {
		writeServiceError(c, err, "TELEGRAM_BIND_FAILED")
		return
	}

	command := "/bind " + email + " " + code
	utils.Success(c, gin.H{
		"email":      email,
		"code":       code,
		"expires_at": expiresAt.Format(timeLayoutRFC3339),
		"command":    command,
	})
}

// Unbind POST /api/v1/telegram/unbind
func (h *TelegramHandler) Unbind(c *gin.Context) {
	userID, _, ok := getUserContext(c)
	if !ok {
		utils.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "未认证用户")
		return
	}

	if err := h.telegramService.UnbindAccount(userID); err != nil {
		writeServiceError(c, err, "TELEGRAM_UNBIND_FAILED")
		return
	}

	utils.SuccessResponse(c, nil, "Telegram 账户解绑成功")
}

// Login POST /api/v1/telegram/login
func (h *TelegramHandler) Login(c *gin.Context) {
	data, err := parseWidgetLoginData(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误: "+err.Error())
		return
	}

	token, user, verifyErr := h.telegramService.VerifyWidgetLogin(data)
	if verifyErr != nil {
		writeServiceError(c, verifyErr, "TELEGRAM_LOGIN_FAILED")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user":  buildTelegramLoginUser(user),
	})
}

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

func parseWidgetLoginData(c *gin.Context) (map[string]string, error) {
	raw := map[string]interface{}{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("登录数据不能为空")
	}

	if nested, exists := raw["data"]; exists {
		if nestedMap, ok := nested.(map[string]interface{}); ok && len(nestedMap) > 0 {
			raw = nestedMap
		}
	}

	return normalizeWidgetData(raw), nil
}

func normalizeWidgetData(raw map[string]interface{}) map[string]string {
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" || value == nil {
			continue
		}

		switch typed := value.(type) {
		case string:
			result[normalizedKey] = strings.TrimSpace(typed)
		case float64:
			if typed == float64(int64(typed)) {
				result[normalizedKey] = strconv.FormatInt(int64(typed), 10)
			} else {
				result[normalizedKey] = strconv.FormatFloat(typed, 'f', -1, 64)
			}
		case int:
			result[normalizedKey] = strconv.Itoa(typed)
		case int64:
			result[normalizedKey] = strconv.FormatInt(typed, 10)
		case bool:
			result[normalizedKey] = strconv.FormatBool(typed)
		default:
			result[normalizedKey] = strings.TrimSpace(fmt.Sprint(typed))
		}
	}
	return result
}

func buildTelegramLoginUser(user *models.User) gin.H {
	if user == nil {
		return gin.H{}
	}

	return gin.H{
		"id":                user.ID,
		"username":          user.Username,
		"email":             user.Email,
		"role":              user.Role,
		"status":            user.Status,
		"vip_level":         user.VipLevel,
		"vip_expires_at":    user.VipExpiresAt,
		"traffic_quota":     user.TrafficQuota,
		"traffic_used":      user.TrafficUsed,
		"telegram_id":       user.TelegramID,
		"telegram_username": user.TelegramUsername,
	}
}
