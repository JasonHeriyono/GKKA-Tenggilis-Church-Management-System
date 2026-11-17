package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"sim-g-backend/helpers/response"
)

// IP yang nggak request lagi selama ini dihapus dari memori
const limiterIdleTTL = 10 * time.Minute

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit membatasi request per IP. Pasang di route sensitif, mis. login:
//
//	auth.POST("/login", middleware.RateLimit(rate.Every(time.Minute/5), 5), h.Login)
func RateLimit(r rate.Limit, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	limiters := map[string]*ipLimiter{}
	lastCleanup := time.Now()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		if now.Sub(lastCleanup) > limiterIdleTTL {
			for k, v := range limiters {
				if now.Sub(v.lastSeen) > limiterIdleTTL {
					delete(limiters, k)
				}
			}
			lastCleanup = now
		}
		l, ok := limiters[ip]
		if !ok {
			l = &ipLimiter{limiter: rate.NewLimiter(r, burst)}
			limiters[ip] = l
		}
		l.lastSeen = now
		allowed := l.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Body{
				Success: false,
				Message: "Terlalu banyak percobaan, coba lagi nanti",
			})
			return
		}
		c.Next()
	}
}
