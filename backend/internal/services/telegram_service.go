package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/config"
	"nodepass-pro/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	telegramBindCodeTTL = 10 * time.Minute
)

// TelegramService Telegram Bot 与 Widget 登录服务。
type TelegramService struct {
	db         *gorm.DB
	httpClient *http.Client
	botToken   string
	vipService *VIPService
}

// TelegramUpdate Webhook Update。
type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

// TelegramMessage Telegram 消息体。
type TelegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Text      string        `json:"text"`
}

// TelegramUser Telegram 用户。
type TelegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type bindCodePayload struct {
	UserID    uint   `json:"user_id"`
	Email     string `json:"email"`
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewTelegramService 创建 Telegram 服务实例。
func NewTelegramService(db *gorm.DB) *TelegramService {
	return &TelegramService{
		db: db,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		vipService: NewVIPService(db),
	}
}

// InitBot 初始化 Telegram Bot，并注册 Webhook。
func (s *TelegramService) InitBot(botToken string) error {
	token := strings.TrimSpace(botToken)
	if token == "" {
		return fmt.Errorf("%w: botToken 不能为空", ErrInvalidParams)
	}
	s.botToken = token

	cfg := config.GlobalConfig
	if cfg == nil {
		return fmt.Errorf("%w: 配置未初始化", ErrInvalidParams)
	}

	webhookURL := strings.TrimSpace(cfg.Telegram.WebhookURL)
	if webhookURL == "" {
		return fmt.Errorf("%w: telegram.webhook_url 未配置", ErrInvalidParams)
	}
	if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
		return fmt.Errorf("%w: telegram.webhook_url 格式错误", ErrInvalidParams)
	}

	form := url.Values{}
	form.Set("url", webhookURL)
	secretToken := resolveTelegramWebhookSecret(cfg, token)
	if secretToken == "" {
		return fmt.Errorf("%w: telegram.secret_token 生成失败", ErrInvalidParams)
	}
	form.Set("secret_token", secretToken)
	if _, err := s.callTelegramAPI("setWebhook", form); err != nil {
		return err
	}
	return nil
}

// HandleCommand 处理 Telegram 命令。
func (s *TelegramService) HandleCommand(update TelegramUpdate) error {
	if update.Message == nil || update.Message.From == nil {
		return nil
	}
	text := strings.TrimSpace(update.Message.Text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return nil
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	command := normalizeCommand(parts[0])
	switch command {
	case "/start":
		message := "欢迎使用 NodePass Panel Telegram 助手。\n可用命令：\n/start\n/bind <email> <验证码>\n/status\n/unbind"
		return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), message)
	case "/bind":
		if len(parts) < 3 {
			return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "用法：/bind <email> <验证码>")
		}
		email := strings.ToLower(strings.TrimSpace(parts[1]))
		code := strings.ToUpper(strings.TrimSpace(parts[2]))
		userID, err := s.verifyAndConsumeBindCode(email, code)
		if err != nil {
			// 记录详细错误到日志
			zap.L().Error("Telegram 绑定失败",
				zap.Error(err),
				zap.String("email", email),
				zap.Int64("telegram_id", update.Message.From.ID))
			// 返回通用错误消息
			if sendErr := s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "绑定失败，请检查邮箱和验证码是否正确"); sendErr != nil {
				return fmt.Errorf("发送绑定失败通知失败: %w", sendErr)
			}
			return nil
		}
		if err = s.BindAccount(strconv.FormatInt(update.Message.From.ID, 10), update.Message.From.Username, userID); err != nil {
			zap.L().Error("Telegram 账户绑定失败",
				zap.Error(err),
				zap.Uint("user_id", userID),
				zap.Int64("telegram_id", update.Message.From.ID))
			if sendErr := s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "绑定失败，请稍后重试"); sendErr != nil {
				return fmt.Errorf("发送绑定失败通知失败: %w", sendErr)
			}
			return nil
		}
		return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "绑定成功，已关联到你的 NodePass 账户。")
	case "/status":
		user, err := s.getUserByTelegramID(strconv.FormatInt(update.Message.From.ID, 10))
		if err != nil {
			return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "未绑定账户，请先在面板生成验证码后执行 /bind。")
		}
		usagePercent := 0.0
		if user.TrafficQuota > 0 {
			usagePercent = float64(user.TrafficUsed) / float64(user.TrafficQuota) * 100
		}
		msg := fmt.Sprintf(
			"账户状态：%s\nVIP 等级：%d\n流量使用：%d / %d (%.2f%%)",
			user.Status, user.VipLevel, user.TrafficUsed, user.TrafficQuota, usagePercent,
		)
		return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), msg)
	case "/unbind":
		user, err := s.getUserByTelegramID(strconv.FormatInt(update.Message.From.ID, 10))
		if err != nil {
			return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "当前 Telegram 未绑定任何账户。")
		}
		if err = s.UnbindAccount(user.ID); err != nil {
			zap.L().Error("Telegram 解绑失败",
				zap.Error(err),
				zap.Uint("user_id", user.ID),
				zap.Int64("telegram_id", update.Message.From.ID))
			return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "解绑失败，请稍后重试")
		}
		return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "解绑成功。")
	default:
		return s.SendNotification(strconv.FormatInt(update.Message.From.ID, 10), "不支持的命令。可用命令：/start /bind /status /unbind")
	}
}

// BindAccount 绑定 Telegram 账户。
func (s *TelegramService) BindAccount(telegramID string, telegramUsername string, userID uint) error {
	telegramID = strings.TrimSpace(telegramID)
	if telegramID == "" || userID == 0 {
		return fmt.Errorf("%w: 参数无效", ErrInvalidParams)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	var exists models.User
	if err := s.db.Where("telegram_id = ? AND id <> ?", telegramID, userID).First(&exists).Error; err == nil {
		return fmt.Errorf("%w: 该 Telegram 已绑定其他账户", ErrConflict)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询 Telegram 绑定关系失败: %w", err)
	}

	trimmedUsername := strings.TrimSpace(telegramUsername)
	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"telegram_id":       telegramID,
		"telegram_username": nullableString(trimmedUsername),
	}).Error
}

// UnbindAccount 解绑 Telegram。
func (s *TelegramService) UnbindAccount(userID uint) error {
	if userID == 0 {
		return fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}
	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"telegram_id":       nil,
		"telegram_username": nil,
	}).Error
}

// VerifyWidgetLogin 验证 Telegram Widget 登录并返回 JWT。
func (s *TelegramService) VerifyWidgetLogin(data map[string]string) (string, *models.User, error) {
	if len(data) == 0 {
		return "", nil, fmt.Errorf("%w: 登录数据为空", ErrInvalidParams)
	}

	cfg := config.GlobalConfig
	if cfg == nil || strings.TrimSpace(cfg.Telegram.BotToken) == "" {
		return "", nil, fmt.Errorf("%w: Telegram Bot Token 未配置", ErrInvalidParams)
	}

	hash := strings.TrimSpace(data["hash"])
	if hash == "" {
		return "", nil, fmt.Errorf("%w: 缺少 hash", ErrInvalidParams)
	}

	authDateRaw := strings.TrimSpace(data["auth_date"])
	authDate, err := strconv.ParseInt(authDateRaw, 10, 64)
	if err != nil || authDate <= 0 {
		return "", nil, fmt.Errorf("%w: auth_date 无效", ErrInvalidParams)
	}
	authTime := time.Unix(authDate, 0)
	if time.Since(authTime) > 24*time.Hour || authTime.After(time.Now().Add(2*time.Minute)) {
		return "", nil, fmt.Errorf("%w: 登录数据已过期", ErrUnauthorized)
	}

	if !verifyTelegramWidgetSignature(cfg.Telegram.BotToken, data) {
		return "", nil, fmt.Errorf("%w: Telegram 签名校验失败", ErrUnauthorized)
	}

	telegramID := strings.TrimSpace(data["id"])
	if telegramID == "" {
		return "", nil, fmt.Errorf("%w: 缺少 id", ErrInvalidParams)
	}

	user, err := s.getUserByTelegramID(telegramID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, err
		}
		user, err = s.autoRegisterTelegramUser(data)
		if err != nil {
			return "", nil, err
		}
	} else {
		username := strings.TrimSpace(data["username"])
		if username != "" && (user.TelegramUsername == nil || *user.TelegramUsername != username) {
			if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Update("telegram_username", username).Error; err != nil {
				// 更新用户名失败不影响登录流程，仅记录日志
				// 可以考虑在上层记录
			}
		}
	}

	token, err := generateUserJWT(user.ID, user.Role)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

// SendNotification 发送 Telegram 消息。
func (s *TelegramService) SendNotification(telegramID string, message string) error {
	if strings.TrimSpace(telegramID) == "" {
		return fmt.Errorf("%w: telegramID 不能为空", ErrInvalidParams)
	}
	if strings.TrimSpace(message) == "" {
		return nil
	}
	if strings.TrimSpace(s.botToken) == "" {
		cfg := config.GlobalConfig
		if cfg != nil {
			s.botToken = strings.TrimSpace(cfg.Telegram.BotToken)
		}
	}
	if strings.TrimSpace(s.botToken) == "" {
		return fmt.Errorf("%w: bot token 未初始化", ErrInvalidParams)
	}

	form := url.Values{}
	form.Set("chat_id", telegramID)
	form.Set("text", message)
	_, err := s.callTelegramAPI("sendMessage", form)
	return err
}

// CreateBindCode 为用户生成绑定验证码。
func (s *TelegramService) CreateBindCode(userID uint) (string, string, time.Time, error) {
	if userID == 0 {
		return "", "", time.Time{}, fmt.Errorf("%w: 用户 ID 无效", ErrInvalidParams)
	}

	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", time.Time{}, fmt.Errorf("%w: 用户不存在", ErrNotFound)
		}
		return "", "", time.Time{}, fmt.Errorf("查询用户失败: %w", err)
	}

	code, err := generateBindCode(6)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("生成绑定验证码失败: %w", err)
	}

	expireAt := time.Now().Add(telegramBindCodeTTL)
	userEmail := strings.ToLower(strings.TrimSpace(user.Email))
	payload := bindCodePayload{
		UserID:    user.ID,
		Email:     userEmail,
		Code:      code,
		ExpiresAt: expireAt.Unix(),
	}
	payloadText, _ := json.Marshal(payload)
	key := "telegram_bind_code:" + userEmail
	value := string(payloadText)
	description := "Telegram 绑定验证码"

	var systemConfig models.SystemConfig
	err = s.db.Where("key = ?", key).First(&systemConfig).Error
	if err == nil {
		if err = s.db.Model(&models.SystemConfig{}).Where("id = ?", systemConfig.ID).Updates(map[string]interface{}{
			"value":       value,
			"description": description,
		}).Error; err != nil {
			return "", "", time.Time{}, fmt.Errorf("保存绑定验证码失败: %w", err)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		systemConfig = models.SystemConfig{
			Key:         key,
			Value:       &value,
			Description: &description,
		}
		if err = s.db.Create(&systemConfig).Error; err != nil {
			return "", "", time.Time{}, fmt.Errorf("保存绑定验证码失败: %w", err)
		}
	} else {
		return "", "", time.Time{}, fmt.Errorf("保存绑定验证码失败: %w", err)
	}

	return userEmail, code, expireAt, nil
}

func (s *TelegramService) verifyAndConsumeBindCode(email string, code string) (uint, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	code = strings.ToUpper(strings.TrimSpace(code))
	if email == "" || code == "" {
		return 0, fmt.Errorf("%w: email 或验证码不能为空", ErrInvalidParams)
	}

	key := "telegram_bind_code:" + email
	var systemConfig models.SystemConfig
	if err := s.db.Where("key = ?", key).First(&systemConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("%w: 未找到绑定验证码", ErrNotFound)
		}
		return 0, fmt.Errorf("查询绑定验证码失败: %w", err)
	}
	if systemConfig.Value == nil {
		return 0, fmt.Errorf("%w: 绑定验证码无效", ErrInvalidParams)
	}

	var payload bindCodePayload
	if err := json.Unmarshal([]byte(*systemConfig.Value), &payload); err != nil {
		return 0, fmt.Errorf("%w: 绑定验证码格式错误", ErrInvalidParams)
	}

	if payload.Email != email || payload.Code != code {
		return 0, fmt.Errorf("%w: 验证码错误", ErrUnauthorized)
	}
	if time.Now().Unix() > payload.ExpiresAt {
		return 0, fmt.Errorf("%w: 验证码已过期", ErrUnauthorized)
	}

	// 删除已使用的验证码
	if err := s.db.Where("id = ?", systemConfig.ID).Delete(&models.SystemConfig{}).Error; err != nil {
		// 删除失败不影响绑定流程，但应该记录日志
		// 可以考虑在上层记录
	}
	return payload.UserID, nil
}

func (s *TelegramService) getUserByTelegramID(telegramID string) (*models.User, error) {
	var user models.User
	err := s.db.Where("telegram_id = ?", strings.TrimSpace(telegramID)).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *TelegramService) autoRegisterTelegramUser(data map[string]string) (*models.User, error) {
	telegramID := strings.TrimSpace(data["id"])
	usernameSeed := strings.TrimSpace(data["username"])
	if usernameSeed == "" {
		usernameSeed = "tg_" + telegramID
	}
	username, err := s.generateUniqueUsername(usernameSeed)
	if err != nil {
		return nil, err
	}
	email, err := s.generateUniqueEmail("tg_" + telegramID + "@telegram.local")
	if err != nil {
		return nil, err
	}

	passwordSeed, err := generateBindCode(16)
	if err != nil {
		return nil, fmt.Errorf("生成随机密码失败: %w", err)
	}
	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(passwordSeed), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成密码哈希失败: %w", err)
	}

	freeLevel, err := s.vipService.getLevelByLevel(0)
	if err != nil {
		return nil, err
	}

	telegramUsername := nullableString(strings.TrimSpace(data["username"]))
	user := &models.User{
		Username:                username,
		Email:                   email,
		PasswordHash:            string(passwordHashBytes),
		Role:                    "user",
		Status:                  "normal",
		VipLevel:                freeLevel.Level,
		VipExpiresAt:            nil,
		TrafficQuota:            freeLevel.TrafficQuota,
		TrafficUsed:             0,
		MaxRules:                freeLevel.MaxRules,
		MaxBandwidth:            freeLevel.MaxBandwidth,
		MaxSelfHostedEntryNodes: freeLevel.MaxSelfHostedEntryNodes,
		MaxSelfHostedExitNodes:  freeLevel.MaxSelfHostedExitNodes,
		TelegramID:              &telegramID,
		TelegramUsername:        telegramUsername,
	}
	if err = s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("自动注册 Telegram 用户失败: %w", err)
	}
	return user, nil
}

func (s *TelegramService) generateUniqueUsername(base string) (string, error) {
	base = normalizeIdentifier(base)
	if base == "" {
		base = "tg_user"
	}

	for i := 0; i < 1000; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s_%d", base, i)
		}
		var count int64
		if err := s.db.Model(&models.User{}).Where("username = ?", candidate).Count(&count).Error; err != nil {
			return "", fmt.Errorf("校验用户名失败: %w", err)
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("生成唯一用户名失败")
}

func (s *TelegramService) generateUniqueEmail(base string) (string, error) {
	base = strings.ToLower(strings.TrimSpace(base))
	if base == "" || !strings.Contains(base, "@") {
		base = "tg_user@telegram.local"
	}

	parts := strings.SplitN(base, "@", 2)
	local := normalizeIdentifier(parts[0])
	domain := parts[1]
	if local == "" {
		local = "tg_user"
	}

	for i := 0; i < 1000; i++ {
		candidateLocal := local
		if i > 0 {
			candidateLocal = fmt.Sprintf("%s_%d", local, i)
		}
		candidate := candidateLocal + "@" + domain
		var count int64
		if err := s.db.Model(&models.User{}).Where("email = ?", candidate).Count(&count).Error; err != nil {
			return "", fmt.Errorf("校验邮箱失败: %w", err)
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("生成唯一邮箱失败")
}

func (s *TelegramService) callTelegramAPI(method string, form url.Values) (map[string]any, error) {
	if strings.TrimSpace(s.botToken) == "" {
		return nil, fmt.Errorf("%w: bot token 未初始化", ErrInvalidParams)
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", s.botToken, method)
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建 Telegram 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 Telegram API 失败: %w", err)
	}
	// 安全地关闭响应体
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	var body map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("解析 Telegram 响应失败: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("Telegram API 返回错误状态: %d", resp.StatusCode)
	}
	if ok, exists := body["ok"].(bool); exists && !ok {
		return nil, fmt.Errorf("Telegram API 返回失败")
	}
	return body, nil
}

func generateBindCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}

	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	charset := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	builder := strings.Builder{}
	builder.Grow(length)
	for _, b := range raw {
		builder.WriteByte(charset[int(b)%len(charset)])
	}
	return builder.String(), nil
}

func resolveTelegramWebhookSecret(cfg *config.Config, botToken string) string {
	if cfg == nil {
		return ""
	}
	secretToken := strings.TrimSpace(cfg.Telegram.SecretToken)
	if secretToken != "" {
		return secretToken
	}

	// 未显式配置时基于 bot token 派生稳定 secret，避免 webhook 裸露。
	sum := sha256.Sum256([]byte("nodepass-webhook:" + strings.TrimSpace(botToken)))
	secretToken = hex.EncodeToString(sum[:16])
	cfg.Telegram.SecretToken = secretToken
	return secretToken
}

func normalizeCommand(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if idx := strings.Index(raw, "@"); idx > 0 {
		raw = raw[:idx]
	}
	return strings.ToLower(raw)
}

func verifyTelegramWidgetSignature(botToken string, data map[string]string) bool {
	hash := strings.TrimSpace(data["hash"])
	if hash == "" {
		return false
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		if key == "hash" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, data[key]))
	}
	dataCheckString := strings.Join(pairs, "\n")

	secretHash := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretHash[:])
	mac.Write([]byte(dataCheckString))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(expected)), []byte(strings.ToLower(hash)))
}

func generateUserJWT(userID uint, role string) (string, error) {
	cfg := config.GlobalConfig
	if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
		return "", fmt.Errorf("%w: JWT 配置无效", ErrInvalidParams)
	}

	expireHours := cfg.JWT.ExpireTime
	// 限制最大过期时间为 1 小时，默认 15 分钟
	expireMinutes := 15 // 默认 15 分钟
	if expireHours > 0 {
		expireMinutes = expireHours * 60
		if expireMinutes > 60 {
			expireMinutes = 60
		}
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Duration(expireMinutes) * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func normalizeIdentifier(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}

	builder := strings.Builder{}
	for _, char := range input {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			builder.WriteRune(char)
		}
	}
	result := builder.String()
	result = strings.Trim(result, "_")
	if result == "" {
		return ""
	}
	return result
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
