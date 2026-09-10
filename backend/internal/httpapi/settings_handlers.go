package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"cryptotrading/internal/settings"
	"cryptotrading/internal/ws"
)

func registerSettingsRoutes(g *gin.RouterGroup, d Deps) {
	g.GET("/settings", func(c *gin.Context) {
		s, err := settings.Get(c.Request.Context(), d.Pool)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, s)
	})

	// POST /settings updates the kill switch / leverage / margin amount /
	// margin type. margin_usd is the fixed per-order margin (default 5
	// USDT) - notional scales with leverage (MarginUSD * Leverage), with no
	// separate absolute notional ceiling by design.
	g.POST("/settings", func(c *gin.Context) {
		var body struct {
			AutotradeEnabled *bool    `json:"autotrade_enabled"`
			Leverage         *int     `json:"leverage"`
			MarginUSD        *float64 `json:"margin_usd"`
			MarginType       *string  `json:"margin_type"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updated, err := settings.Update(c.Request.Context(), d.Pool, settings.UpdateInput{
			AutotradeEnabled: body.AutotradeEnabled,
			Leverage:         body.Leverage,
			MarginUSD:        body.MarginUSD,
			MarginType:       body.MarginType,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		d.Hub.Broadcast(ws.Message{Type: "settings", Data: updated})
		c.JSON(http.StatusOK, updated)
	})
}
