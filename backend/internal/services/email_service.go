package services

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"nodepass-pro/backend/internal/models"
	"nodepass-pro/backend/internal/utils"

	"gorm.io/gorm"
)

var ErrSMTPNotEnabled = errors.New("smtp not enabled")

type SMTPConfig struct {
	Enabled    bool
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	ReplyTo    string
	Encryption string
	SkipVerify bool
}

type EmailService struct {
	db *gorm.DB
}

func NewEmailService(db *gorm.DB) *EmailService {
	return &EmailService{db: db}
}

func (s *EmailService) SendEmailChangeCode(targetEmail string, code string, expiresAt time.Time) error {
	targetEmail = strings.TrimSpace(targetEmail)
	if targetEmail == "" {
		return fmt.Errorf("%w: 收件人邮箱不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateEmail(targetEmail); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	body := strings.Builder{}
	body.WriteString("您好，\n\n")
	body.WriteString("您正在修改 NodePass 账户邮箱。\n")
	body.WriteString("验证码：")
	body.WriteString(code)
	body.WriteString("\n")
	body.WriteString("有效期至：")
	body.WriteString(expiresAt.Local().Format("2006-01-02 15:04:05"))
	body.WriteString("\n\n")
	body.WriteString("如非本人操作，请忽略此邮件。")

	return s.Send(targetEmail, "NodePass 邮箱验证码", body.String())
}

func (s *EmailService) Send(to string, subject string, body string) error {
	cfg, err := s.LoadSMTPConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return ErrSMTPNotEnabled
	}
	if err := validateSMTPConfig(cfg); err != nil {
		return err
	}

	to = strings.TrimSpace(to)
	if err := utils.ValidateEmail(to); err != nil {
		return fmt.Errorf("%w: 收件人邮箱格式错误: %v", ErrInvalidParams, err)
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = "NodePass 通知"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("%w: 邮件内容不能为空", ErrInvalidParams)
	}

	message, err := buildEmailMessage(cfg, to, subject, body)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := buildSMTPAuth(cfg)

	switch cfg.Encryption {
	case "ssl":
		return sendWithSMTPS(address, cfg, auth, to, message)
	case "none":
		return smtp.SendMail(address, auth, cfg.FromEmail, []string{to}, message)
	default:
		return sendWithSTARTTLS(address, cfg, auth, to, message)
	}
}

func (s *EmailService) LoadSMTPConfig() (*SMTPConfig, error) {
	keys := []string{
		"smtp_enabled",
		"smtp_host",
		"smtp_port",
		"smtp_username",
		"smtp_password",
		"smtp_from_email",
		"smtp_from_name",
		"smtp_reply_to",
		"smtp_encryption",
		"smtp_skip_verify",
	}

	items := make([]models.SystemConfig, 0)
	if err := s.db.Model(&models.SystemConfig{}).Where("key IN ?", keys).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询 SMTP 配置失败: %w", err)
	}

	lookup := make(map[string]string, len(items))
	for _, item := range items {
		if item.Value != nil {
			lookup[item.Key] = strings.TrimSpace(*item.Value)
		}
	}

	cfg := &SMTPConfig{
		Enabled:    false,
		Host:       lookup["smtp_host"],
		Port:       587,
		Username:   lookup["smtp_username"],
		Password:   lookup["smtp_password"],
		FromEmail:  lookup["smtp_from_email"],
		FromName:   lookup["smtp_from_name"],
		ReplyTo:    lookup["smtp_reply_to"],
		Encryption: "starttls",
		SkipVerify: false,
	}

	if raw := lookup["smtp_enabled"]; raw != "" {
		enabled, err := parseBooleanConfig(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: smtp_enabled 配置非法", ErrInvalidParams)
		}
		cfg.Enabled = enabled
	}
	if raw := lookup["smtp_port"]; raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: smtp_port 配置非法", ErrInvalidParams)
		}
		cfg.Port = port
	}
	if raw := lookup["smtp_encryption"]; raw != "" {
		cfg.Encryption = strings.ToLower(raw)
	}
	if raw := lookup["smtp_skip_verify"]; raw != "" {
		skipVerify, err := parseBooleanConfig(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: smtp_skip_verify 配置非法", ErrInvalidParams)
		}
		cfg.SkipVerify = skipVerify
	}

	return cfg, nil
}

func validateSMTPConfig(cfg *SMTPConfig) error {
	if cfg == nil {
		return fmt.Errorf("%w: SMTP 配置为空", ErrInvalidParams)
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("%w: smtp_host 不能为空", ErrInvalidParams)
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("%w: smtp_port 无效", ErrInvalidParams)
	}
	if strings.TrimSpace(cfg.FromEmail) == "" {
		return fmt.Errorf("%w: smtp_from_email 不能为空", ErrInvalidParams)
	}
	if err := utils.ValidateEmail(cfg.FromEmail); err != nil {
		return fmt.Errorf("%w: smtp_from_email 无效: %v", ErrInvalidParams, err)
	}
	if strings.TrimSpace(cfg.ReplyTo) != "" {
		if err := utils.ValidateEmail(cfg.ReplyTo); err != nil {
			return fmt.Errorf("%w: smtp_reply_to 无效: %v", ErrInvalidParams, err)
		}
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Encryption)) {
	case "none", "starttls", "ssl":
	default:
		return fmt.Errorf("%w: smtp_encryption 仅支持 none/starttls/ssl", ErrInvalidParams)
	}
	return nil
}

func buildSMTPAuth(cfg *SMTPConfig) smtp.Auth {
	if cfg == nil {
		return nil
	}
	username := strings.TrimSpace(cfg.Username)
	if username == "" {
		return nil
	}
	return smtp.PlainAuth("", username, cfg.Password, cfg.Host)
}

func buildEmailMessage(cfg *SMTPConfig, to string, subject string, body string) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: SMTP 配置为空", ErrInvalidParams)
	}

	fromAddress := mail.Address{
		Name:    strings.TrimSpace(cfg.FromName),
		Address: strings.TrimSpace(cfg.FromEmail),
	}
	if fromAddress.Name == "" {
		fromAddress.Name = "NodePass"
	}
	toAddress := mail.Address{Address: to}

	header := map[string]string{
		"From":         fromAddress.String(),
		"To":           toAddress.String(),
		"Subject":      encodeRFC2047(subject),
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	if strings.TrimSpace(cfg.ReplyTo) != "" {
		header["Reply-To"] = strings.TrimSpace(cfg.ReplyTo)
	}

	var buffer bytes.Buffer
	for key, value := range header {
		buffer.WriteString(key)
		buffer.WriteString(": ")
		buffer.WriteString(value)
		buffer.WriteString("\r\n")
	}
	buffer.WriteString("\r\n")
	buffer.WriteString(body)

	return buffer.Bytes(), nil
}

func encodeRFC2047(text string) string {
	return mime.BEncoding.Encode("UTF-8", text)
}

func sendWithSTARTTLS(address string, cfg *SMTPConfig, auth smtp.Auth, to string, message []byte) error {
	client, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer client.Close()

	tlsConfig := &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.SkipVerify,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("SMTP STARTTLS 握手失败: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 身份认证失败: %w", err)
		}
	}
	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM 失败: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO 失败: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA 失败: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("写入 SMTP 消息失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("提交 SMTP 消息失败: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP QUIT 失败: %w", err)
	}
	return nil
}

func sendWithSMTPS(address string, cfg *SMTPConfig, auth smtp.Auth, to string, message []byte) error {
	conn, err := tls.Dial("tcp", address, &tls.Config{
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.SkipVerify,
	})
	if err != nil {
		return fmt.Errorf("连接 SMTPS 服务器失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTPS 客户端失败: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTPS 身份认证失败: %w", err)
		}
	}
	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("SMTPS MAIL FROM 失败: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTPS RCPT TO 失败: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTPS DATA 失败: %w", err)
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("写入 SMTPS 消息失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("提交 SMTPS 消息失败: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTPS QUIT 失败: %w", err)
	}
	return nil
}
