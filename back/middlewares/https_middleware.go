package middlewares

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"warehouse-web/utils"
)

// RequireHTTPS rejects non-TLS requests when REQUIRE_HTTPS is enabled.
func RequireHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isHTTPSEnforced() {
			c.Next()
			return
		}
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			c.Next()
			return
		}

		utils.JSONError(c, http.StatusUpgradeRequired, "https_required", "HTTPS is required")
		c.Abort()
	}
}

func isHTTPSEnforced() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("REQUIRE_HTTPS")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
