package middleware

import (
	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/debuginfo"
)

// Debug (khusus dev) menyiapkan catatan query per request, nanti ikut dikirim di field "debug" response.
// Wajib dipasang SETELAH RequestID.
func Debug() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := debuginfo.Mulai(c.Request.Context(), c.GetString(RequestIDKey))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
