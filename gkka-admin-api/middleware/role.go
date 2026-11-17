package middleware

import (
	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/response"
)

// RequireRole hanya meloloskan user dengan salah satu role yang disebut. Wajib dipasang SETELAH Auth.
// Usher tidak dicek di sini: itu hak sementara dari jadwal (lihat schedule/usher_duty.go).
//
//	g := rg.Group("/users", middleware.Auth(parse), middleware.RequireRole(models.RoleSuperAdmin))
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		u, ok := currentuser.From(c)
		if !ok {
			response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
			return
		}
		if _, ok := allowed[u.Role]; !ok {
			response.Error(c, apperror.Forbidden("Anda tidak memiliki akses ke fitur ini"))
			return
		}
		c.Next()
	}
}
