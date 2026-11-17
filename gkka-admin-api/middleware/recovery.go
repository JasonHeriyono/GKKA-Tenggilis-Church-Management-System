package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/debuginfo"
	"sim-g-backend/helpers/response"
)

// Recovery biar server nggak mati kalau ada panic, stack trace-nya masuk log.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic",
					"request_id", c.GetString(RequestIDKey),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"panic", fmt.Sprint(r),
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, response.Body{
					Success: false,
					Message: "Terjadi kesalahan pada server",
					Debug:   debuginfo.Salin(c.Request.Context(), fmt.Errorf("panic: %v", r)), // isi pesan panic, khusus dev
				})
			}
		}()
		c.Next()
	}
}
