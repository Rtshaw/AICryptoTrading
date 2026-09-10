package httpapi

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"cryptotrading/internal/models"
)

func registerAutotradeRoutes(g *gin.RouterGroup, d Deps) {
	// GET /autotrade/log is the transparency trail: every candle-close strategy
	// evaluation per symbol, including holds and skips and why.
	g.GET("/autotrade/log", func(c *gin.Context) {
		symbol := c.Query("symbol")
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}

		query := `SELECT id, symbol, ts, rule_action, rule_reason, ai_signal_id, decision, skip_reason, order_id FROM auto_trade_log`
		args := []any{}
		if symbol != "" {
			query += ` WHERE symbol = $1 ORDER BY ts DESC LIMIT $2`
			args = []any{symbol, limit}
		} else {
			query += ` ORDER BY ts DESC LIMIT $1`
			args = []any{limit}
		}

		rows, err := d.Pool.Query(c.Request.Context(), query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		out := []models.AutoTradeLogEntry{}
		for rows.Next() {
			var e models.AutoTradeLogEntry
			var skipReason *string
			if err := rows.Scan(&e.ID, &e.Symbol, &e.Ts, &e.RuleAction, &e.RuleReason, &e.AISignalID, &e.Decision, &skipReason, &e.OrderID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if skipReason != nil {
				e.SkipReason = *skipReason
			}
			out = append(out, e)
		}
		c.JSON(http.StatusOK, out)
	})
}
