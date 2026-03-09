package handlers

import (
	"io"
	"net/http"
	"strconv"

	"nodepass-license-unified/internal/services"
	"nodepass-license-unified/internal/utils"

	"github.com/gin-gonic/gin"
)

// CommercialHandler 商业化接口处理器。
type CommercialHandler struct {
	service *services.CommercialService
}

// NewCommercialHandler 创建商业化处理器。
func NewCommercialHandler(service *services.CommercialService) *CommercialHandler {
	return &CommercialHandler{service: service}
}

// IssueTrial 发放试用授权。
func (h *CommercialHandler) IssueTrial(c *gin.Context) {
	var req services.IssueTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	result, err := h.service.IssueTrial(&req, adminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "ISSUE_TRIAL_FAILED", err.Error())
		return
	}
	utils.Success(c, result, "试用授权发放成功")
}

// CreateRenewOrder 创建续费订单。
func (h *CommercialHandler) CreateRenewOrder(c *gin.Context) {
	var req services.CreateRenewOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	order, err := h.service.CreateRenewOrder(&req, adminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_ORDER_FAILED", err.Error())
		return
	}
	utils.Success(c, order, "续费订单创建成功")
}

// CreateUpgradeOrder 创建升级订单。
func (h *CommercialHandler) CreateUpgradeOrder(c *gin.Context) {
	var req services.CreateUpgradeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	order, err := h.service.CreateUpgradeOrder(&req, adminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_ORDER_FAILED", err.Error())
		return
	}
	utils.Success(c, order, "升级订单创建成功")
}

// CreateTransferOrder 创建转移订单。
func (h *CommercialHandler) CreateTransferOrder(c *gin.Context) {
	var req services.CreateTransferOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	order, err := h.service.CreateTransferOrder(&req, adminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "CREATE_ORDER_FAILED", err.Error())
		return
	}
	utils.Success(c, order, "转移订单创建成功")
}

// ListOrders 查询订单列表。
func (h *CommercialHandler) ListOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	licenseID, _ := strconv.ParseUint(c.Query("license_id"), 10, 64)

	result, err := h.service.ListOrders(services.OrderFilter{
		Status:    c.Query("status"),
		Action:    c.Query("action"),
		Customer:  c.Query("customer"),
		LicenseID: uint(licenseID),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, "QUERY_FAILED", err.Error())
		return
	}

	utils.Success(c, result, "ok")
}

// GetOrder 获取订单详情。
func (h *CommercialHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	item, err := h.service.GetOrder(uint(id))
	if err != nil {
		utils.Error(c, http.StatusNotFound, "NOT_FOUND", "订单不存在")
		return
	}
	utils.Success(c, item, "ok")
}

// MarkOrderPaid 手动确认支付。
func (h *CommercialHandler) MarkOrderPaid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "id 无效")
		return
	}

	var req services.MarkOrderPaidRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		if err != io.EOF {
			utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
			return
		}
	}

	item, err := h.service.MarkOrderPaid(uint(id), &req, adminIDFromContext(c))
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "MARK_PAID_FAILED", err.Error())
		return
	}

	utils.Success(c, item, "订单已确认支付")
}

// PaymentCallback 支付回调（公开）。
func (h *CommercialHandler) PaymentCallback(c *gin.Context) {
	channel := c.Param("channel")
	if channel == "" {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", "channel 不能为空")
		return
	}

	var req services.PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}

	if err := h.service.VerifyPaymentCallbackSignature(
		channel,
		&req,
		c.GetHeader(services.CallbackHeaderSignature),
		c.GetHeader(services.CallbackHeaderTimestamp),
		c.GetHeader(services.CallbackHeaderNonce),
	); err != nil {
		utils.Error(c, http.StatusUnauthorized, "INVALID_SIGNATURE", err.Error())
		return
	}

	order, err := h.service.HandlePaymentCallback(channel, &req)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "PAYMENT_CALLBACK_FAILED", err.Error())
		return
	}
	utils.Success(c, order, "回调处理成功")
}

func adminIDFromContext(c *gin.Context) uint {
	if c == nil {
		return 0
	}
	value, ok := c.Get("admin_id")
	if !ok {
		return 0
	}
	adminID, ok := value.(uint)
	if !ok {
		return 0
	}
	return adminID
}
