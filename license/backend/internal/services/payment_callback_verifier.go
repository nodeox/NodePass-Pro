package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// CallbackHeaderSignature 支付回调签名头。
	CallbackHeaderSignature = "X-Callback-Signature"
	// CallbackHeaderTimestamp 支付回调时间戳头（Unix 秒）。
	CallbackHeaderTimestamp = "X-Callback-Timestamp"
	// CallbackHeaderNonce 支付回调随机串头。
	CallbackHeaderNonce = "X-Callback-Nonce"
)

// PaymentCallbackVerifier 支付回调签名验证器。
type PaymentCallbackVerifier struct {
	strict           bool
	toleranceSeconds int
	secrets          map[string]string
}

// NewPaymentCallbackVerifier 创建验证器。
func NewPaymentCallbackVerifier(strict bool, toleranceSeconds int, secrets map[string]string) *PaymentCallbackVerifier {
	copied := make(map[string]string, len(secrets))
	for key, value := range secrets {
		normalizedKey := strings.TrimSpace(strings.ToLower(key))
		trimmedValue := strings.TrimSpace(value)
		if normalizedKey == "" || trimmedValue == "" {
			continue
		}
		copied[normalizedKey] = trimmedValue
	}
	if toleranceSeconds <= 0 {
		toleranceSeconds = 300
	}
	return &PaymentCallbackVerifier{
		strict:           strict,
		toleranceSeconds: toleranceSeconds,
		secrets:          copied,
	}
}

// Verify 验证支付回调签名。
func (v *PaymentCallbackVerifier) Verify(channel string, req *PaymentCallbackRequest, signature, timestamp, nonce string) error {
	if v == nil {
		return nil
	}

	normalizedChannel := strings.TrimSpace(strings.ToLower(channel))
	secret := v.secretForChannel(normalizedChannel)
	if secret == "" {
		if v.strict {
			return fmt.Errorf("未配置支付渠道 %s 的回调密钥", normalizedChannel)
		}
		return nil
	}

	trimmedSignature := strings.TrimSpace(strings.ToLower(signature))
	if trimmedSignature == "" {
		return errors.New("缺少回调签名")
	}

	ts := strings.TrimSpace(timestamp)
	if ts == "" {
		return errors.New("缺少回调时间戳")
	}

	tsValue, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("回调时间戳格式无效")
	}

	now := time.Now().Unix()
	diff := now - tsValue
	if diff < 0 {
		diff = -diff
	}
	if diff > int64(v.toleranceSeconds) {
		return errors.New("回调时间戳超出允许范围")
	}

	expected := GeneratePaymentCallbackSignature(secret, normalizedChannel, req, ts, strings.TrimSpace(nonce))
	if !secureCompare(expected, trimmedSignature) {
		return errors.New("回调签名校验失败")
	}
	return nil
}

// GeneratePaymentCallbackSignature 生成支付回调签名（HEX 小写）。
func GeneratePaymentCallbackSignature(secret, channel string, req *PaymentCallbackRequest, timestamp, nonce string) string {
	signingText := BuildPaymentCallbackSigningText(channel, req, timestamp, nonce)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write([]byte(signingText))
	return hex.EncodeToString(mac.Sum(nil))
}

// BuildPaymentCallbackSigningText 构造签名原文。
func BuildPaymentCallbackSigningText(channel string, req *PaymentCallbackRequest, timestamp, nonce string) string {
	amount := ""
	orderNo := ""
	status := ""
	paymentTxnID := ""
	if req != nil {
		orderNo = strings.TrimSpace(req.OrderNo)
		status = strings.TrimSpace(strings.ToLower(req.Status))
		paymentTxnID = strings.TrimSpace(req.PaymentTxnID)
		if req.AmountCents != nil {
			amount = strconv.FormatInt(*req.AmountCents, 10)
		}
	}

	return fmt.Sprintf(
		"channel=%s&order_no=%s&status=%s&amount_cents=%s&payment_txn_id=%s&timestamp=%s&nonce=%s",
		strings.TrimSpace(strings.ToLower(channel)),
		orderNo,
		status,
		amount,
		paymentTxnID,
		strings.TrimSpace(timestamp),
		strings.TrimSpace(nonce),
	)
}

func (v *PaymentCallbackVerifier) secretForChannel(channel string) string {
	if v == nil {
		return ""
	}
	if secret := strings.TrimSpace(v.secrets[channel]); secret != "" {
		return secret
	}
	return strings.TrimSpace(v.secrets["default"])
}

func secureCompare(left, right string) bool {
	leftBytes := []byte(strings.TrimSpace(strings.ToLower(left)))
	rightBytes := []byte(strings.TrimSpace(strings.ToLower(right)))
	if len(leftBytes) != len(rightBytes) {
		return false
	}
	return hmac.Equal(leftBytes, rightBytes)
}
