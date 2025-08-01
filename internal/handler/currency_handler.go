package handler

import (
	"RnD-service/internal/usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CurrencyHandler struct {
	usecase usecase.CurrencyUsecase
	logger  *logrus.Logger
}

func NewRateHandler(usecase usecase.CurrencyUsecase, logger *logrus.Logger) *CurrencyHandler {
	return &CurrencyHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *CurrencyHandler) StoreRatesFromCBR(c *gin.Context) {
	if err := h.usecase.FetchAndStoreRatesFromCBR(c.Request.Context()); err != nil {
		h.logger.Errorf("Failed to fetch and store rates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rates successfully updated"})
}

func (h *CurrencyHandler) GetRateByCharCode(c *gin.Context) {
	charCode := c.Query("char_code")
	if charCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "char_code is required"})
		return
	}

	amountStr := c.Query("amount")
	amount := 1.0
	if amountStr != "" {
		var err error
		amount, err = strconv.ParseFloat(amountStr, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount format"})
			return
		}
		if amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be positive"})
			return
		}
	}

	result, err := h.usecase.GetRateByCharCode(c.Request.Context(), charCode, amount)
	if err != nil {
		h.logger.Errorf("Failed to get rate by char code %s: %v", charCode, err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	resp := usecase.CurrencyResponse{
		CharName: result.CharName,
		ValueRUB: result.ValueRUB,
	}

	c.JSON(http.StatusOK, resp)
}
