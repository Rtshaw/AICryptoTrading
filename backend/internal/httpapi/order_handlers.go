package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"cryptotrading/internal/models"
)

func registerOrderRoutes(g *gin.RouterGroup, d Deps) {
	// POST /orders places a manual MARKET order, routed through the exact
	// same $10-notional-cap sizing/validation as the automatic engine
	// (autotrader.PlaceManualOrder). It bypasses the kill switch (a manual
	// action is an explicit override) but not the cap.
	g.POST("/orders", func(c *gin.Context) {
		var body struct {
			Symbol string `json:"symbol" binding:"required"`
			Side   string `json:"side" binding:"required,oneof=BUY SELL"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order, err := d.Trader.PlaceManualOrder(c.Request.Context(), body.Symbol, models.OrderSide(body.Side))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, order)
	})

	// POST /orders/flatten closes an open position at market. Always
	// allowed regardless of the kill switch, and exempt from the cap.
	g.POST("/orders/flatten", func(c *gin.Context) {
		var body struct {
			Symbol string `json:"symbol" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order, err := d.Trader.Flatten(c.Request.Context(), body.Symbol)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, order)
	})

	// POST /orders/:symbol/protect manually backfills STOP_MARKET/
	// TAKE_PROFIT_MARKET closePosition algo orders onto symbol's CURRENT
	// open position - the recovery path for a position that ended up
	// unprotected (autotrader.AttachProtectiveOrders; see
	// autotrader.placeProtectiveOrders for why entry-time attachment can
	// fail without rolling back the entry). Errors if there's no open
	// position for symbol.
	g.POST("/orders/:symbol/protect", func(c *gin.Context) {
		var body struct {
			StopLoss   float64 `json:"stop_loss" binding:"required"`
			TakeProfit float64 `json:"take_profit" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		symbol := c.Param("symbol")
		if err := d.Trader.AttachProtectiveOrders(c.Request.Context(), symbol, body.StopLoss, body.TakeProfit); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}
