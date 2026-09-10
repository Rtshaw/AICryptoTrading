package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// corsMiddleware allows any origin. This is a local single-user dev tool
// (see ws.Hub's CheckOrigin for the same reasoning) - there is no
// authenticated session or cookie a hostile page could ride on, so open CORS
// trades no real security away.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func apiKeyMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing X-API-Key"})
			return
		}
		c.Next()
	}
}
