package services

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPaymentCallbackVerifierVerify(t *testing.T) {
	verifier := NewPaymentCallbackVerifier(true, 300, map[string]string{
		"wechat": "wechat-secret",
	})

	amount := int64(500)
	req := &PaymentCallbackRequest{
		OrderNo:      "NPORD-20260309080000-ABCDE",
		PaymentTxnID: "txn-1",
		Status:       OrderStatusPaid,
		AmountCents:  &amount,
	}
	timestamp := time.Now().Unix()
	nonce := "abc"
	signature := GeneratePaymentCallbackSignature("wechat-secret", "wechat", req, int64ToString(timestamp), nonce)

	if err := verifier.Verify("wechat", req, signature, int64ToString(timestamp), nonce); err != nil {
		t.Fatalf("verify should pass: %v", err)
	}
}

func TestPaymentCallbackVerifierTimestampExpired(t *testing.T) {
	verifier := NewPaymentCallbackVerifier(true, 1, map[string]string{
		"alipay": "alipay-secret",
	})

	amount := int64(500)
	req := &PaymentCallbackRequest{
		OrderNo:      "NPORD-20260309080000-ABCDE",
		PaymentTxnID: "txn-2",
		Status:       OrderStatusPaid,
		AmountCents:  &amount,
	}
	expiredTs := time.Now().Add(-10 * time.Second).Unix()
	nonce := "abc2"
	signature := GeneratePaymentCallbackSignature("alipay-secret", "alipay", req, int64ToString(expiredTs), nonce)

	err := verifier.Verify("alipay", req, signature, int64ToString(expiredTs), nonce)
	if err == nil {
		t.Fatalf("expected timestamp expired")
	}
	if !strings.Contains(err.Error(), "回调时间戳超出允许范围") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func int64ToString(value int64) string {
	return strconv.FormatInt(value, 10)
}
