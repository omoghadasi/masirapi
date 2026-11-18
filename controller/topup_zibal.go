package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// zibalCallbackPath must match the public route registered in the router.
const zibalCallbackPath = "/api/user/zibal/callback"

// zibalMinAmountRial is the gateway's documented lower bound for a payment
// request (result 105), so amounts below it are rejected before the round trip.
const zibalMinAmountRial = 1000

func isZibalTopUpEnabled() bool {
	return isPaymentComplianceConfirmed() && setting.IsZibalEnabled()
}

func isZibalWebhookEnabled() bool {
	return isZibalTopUpEnabled()
}

func GetZibalClient() *service.ZibalClient {
	if !setting.IsZibalEnabled() {
		return nil
	}
	zibal := setting.GetZibalSetting()
	return service.NewZibalClient(zibal.MerchantID, zibal.Sandbox)
}

type ZibalPayRequest struct {
	// Amount is the top-up quantity in the unit the recharge form uses.
	Amount int64 `json:"amount"`
	// PaymentMethod must be the Zibal method.
	PaymentMethod string `json:"payment_method"`
}

// RequestZibalPay reserves a gateway payment and returns its checkout URL.
// Zibal settles an order only after the merchant confirms it server-side, so
// the order stays pending until ZibalCallback verifies the payment.
func RequestZibalPay(c *gin.Context) {
	var req ZibalPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		paymentError(c, common.NewMessage("Invalid parameters"))
		return
	}
	if req.PaymentMethod != model.PaymentMethodZibal {
		paymentError(c, common.NewMessage("Unsupported payment channel"))
		return
	}
	minAmount := max(getMinTopup(), setting.GetZibalSetting().MinTopUp)
	if req.Amount < minAmount {
		paymentError(c, common.NewMessage("Top-up amount cannot be less than {{min}}", map[string]any{"min": minAmount}))
		return
	}

	id := c.GetInt("id")
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}
	group, err := model.GetUserGroup(id, true)
	if err != nil {
		paymentError(c, common.NewMessage("Failed to get the user group"))
		return
	}
	client := GetZibalClient()
	if client == nil {
		paymentError(c, common.NewMessage("Payment information has not been configured by administrator"))
		return
	}

	payMoney := getPayMoney(req.Amount, group)
	amountRial := setting.ZibalRialAmount(payMoney)
	if amountRial < zibalMinAmountRial {
		paymentError(c, common.NewMessage("Payment amount is too low"))
		return
	}

	checkout, err := client.Request(c.Request.Context(), amountRial, fmt.Sprintf("Top-up user_id=%d", id), service.GetCallbackAddress()+zibalCallbackPath)
	if err != nil {
		logger.LogError(c.Request.Context(), common.LogText("Zibal failed to start payment user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		paymentError(c, common.NewMessage("Failed to start payment"))
		return
	}

	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		dAmount := decimal.NewFromInt(amount)
		dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)
		amount = dAmount.Div(dQuotaPerUnit).IntPart()
	}
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          amount,
		Money:           payMoney,
		TradeNo:         strconv.FormatInt(checkout.TrackID, 10),
		PaymentMethod:   model.PaymentMethodZibal,
		PaymentProvider: model.PaymentProviderZibal,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), common.LogText("Zibal failed to create top-up order user_id=%d trade_no=%d amount=%d error=%q", id, checkout.TrackID, amount, err.Error()))
		paymentError(c, common.NewMessage("Failed to create order"))
		return
	}
	logger.LogInfo(c.Request.Context(), common.LogText("Zibal top-up order created user_id=%d trade_no=%d amount=%d money=%.2f", id, checkout.TrackID, amount, payMoney))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"pay_link": checkout.PayURL,
		},
	})
}

// ZibalCallback handles the browser redirect the gateway performs after the
// payer leaves its payment page. The query status only decides whether the
// gateway is asked to confirm; quota is credited exclusively on a successful
// server-side verification, so a forged success=1 grants nothing.
func ZibalCallback(c *gin.Context) {
	if !isZibalWebhookEnabled() {
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback rejected reason=gateway_disabled path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	trackIDText := c.Query("trackId")
	if trackIDText == "" {
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback trackId is missing path=%q client_ip=%s", c.Request.RequestURI, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}
	trackID, err := strconv.ParseInt(trackIDText, 10, 64)
	if err != nil || trackID <= 0 {
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback trackId is invalid track_id=%q client_ip=%s", trackIDText, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	if c.Query("success") != "1" {
		logger.LogInfo(c.Request.Context(), common.LogText("Zibal callback payment not completed trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		if err := model.UpdatePendingTopUpStatus(trackIDText, model.PaymentProviderZibal, common.TopUpStatusFailed); err != nil && !errors.Is(err, model.ErrTopUpNotFound) {
			logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback failed to mark order unpaid trade_no=%d client_ip=%s error=%q", trackID, c.ClientIP(), err.Error()))
		}
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	topUp := model.GetTopUpByTradeNo(trackIDText)
	if topUp == nil {
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback order not found trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}
	if topUp.Status == common.TopUpStatusSuccess {
		logger.LogInfo(c.Request.Context(), common.LogText("Zibal callback duplicate redirect ignored trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=success"))
		return
	}
	if topUp.Status != common.TopUpStatusPending {
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal callback order status invalid trade_no=%d status=%s client_ip=%s", trackID, topUp.Status, c.ClientIP()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=fail"))
		return
	}

	verification, err := GetZibalClient().Verify(c.Request.Context(), trackID)
	if err != nil {
		// The gateway may still have captured the money, so the order stays
		// pending for the administrator to reconcile or complete manually.
		logger.LogWarn(c.Request.Context(), common.LogText("Zibal verification failed trade_no=%d user_id=%d client_ip=%s error=%q", trackID, topUp.UserId, c.ClientIP(), err.Error()))
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=pending"))
		return
	}

	LockOrder(trackIDText)
	defer UnlockOrder(trackIDText)
	alreadyDone, err := model.RechargeZibal(trackIDText, c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTopUpNotFound):
			logger.LogWarn(c.Request.Context(), common.LogText("Zibal top-up order not found trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		case errors.Is(err, model.ErrPaymentMethodMismatch):
			logger.LogWarn(c.Request.Context(), common.LogText("Zibal order payment gateway mismatch trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		case errors.Is(err, model.ErrTopUpStatusInvalid):
			logger.LogWarn(c.Request.Context(), common.LogText("Zibal order status invalid trade_no=%d client_ip=%s", trackID, c.ClientIP()))
		default:
			logger.LogError(c.Request.Context(), common.LogText("Zibal top-up processing failed trade_no=%d client_ip=%s error=%q", trackID, c.ClientIP(), err.Error()))
		}
		c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=pending"))
		return
	}

	if verification.AlreadyVerified {
		logger.LogInfo(c.Request.Context(), common.LogText("Zibal payment verified earlier trade_no=%d ref_number=%d amount=%d already_done=%v client_ip=%s", trackID, verification.RefNumber, verification.Amount, alreadyDone, c.ClientIP()))
	} else {
		logger.LogInfo(c.Request.Context(), common.LogText("Zibal top-up settled trade_no=%d ref_number=%d amount=%d already_done=%v client_ip=%s", trackID, verification.RefNumber, verification.Amount, alreadyDone, c.ClientIP()))
	}
	c.Redirect(http.StatusFound, paymentReturnPath("/wallet?pay=success"))
}
