package handler

import (
	"RnD-service/internal/usecase"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type CurrencyHandler struct {
	usecase usecase.RateUsecase
	logger  *logrus.Logger
}

func NewRateHandler(usecase usecase.RateUsecase, logger *logrus.Logger) *CurrencyHandler {
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

func (h *CurrencyHandler) GetHistoricalRateByCharCode(c *gin.Context) {
	valCode := c.Query("val")
	amountStr := c.Query("amount")
	dateStr := c.Query("date")

	if valCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required query parameter 'val'"})
		return
	}

	var date time.Time
	var err error
	if dateStr == "" {
		date = time.Now().Truncate(24 * time.Hour)
		h.logger.Debugf("Date parameter not provided, using default (today): %s", date.Format("2006-01-02"))
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			h.logger.WithError(err).Errorf("Invalid date format: %s", dateStr)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYY-MM-DD"})
			return
		}
		date = date.Truncate(24 * time.Hour)
	}

	amount := 1.0
	if amountStr != "" {
		parsedAmount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || parsedAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'amount' parameter, must be a positive number"})
			return
		}
		amount = parsedAmount
	}

	today := time.Now().Truncate(24 * time.Hour)
	if date.After(today) {
		h.logger.Debugf("Requested future date: %s, canceling...", date.Format("2006-01-02"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot fetch rates for future dates"})
		return
	}

	result, err := h.usecase.GetHistoricalRateByCharCode(c.Request.Context(), valCode, date, amount)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "invalid char code") || strings.Contains(errorMsg, "invalid date format") || strings.Contains(errorMsg, "invalid 'amount'") {
			statusCode = http.StatusBadRequest
		} else if strings.Contains(errorMsg, "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(errorMsg, "future dates") {
			statusCode = http.StatusBadRequest
		} else if strings.Contains(errorMsg, "no rates available") {
			statusCode = http.StatusNotFound
		}
		h.logger.WithError(err).Errorf("Failed to get historical rate for val=%s, date=%s, amount=%.2f", valCode, date.Format("2006-01-02"), amount)
		c.JSON(statusCode, gin.H{"error": errorMsg})
		return
	}

	c.JSON(http.StatusOK, result)
}
