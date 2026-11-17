package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/response"
)

// TokenParser memverifikasi access token dan mengembalikan identitas user.
// Diisi auth.Service.ParseAccessToken saat routes dirakit (middleware tidak meng-import modul auth).
type TokenParser func(token string) (currentuser.User, error)

// Auth mewajibkan header "Authorization: Bearer <access token>" yang valid,
// lalu menyimpan user yang login ke context (lihat helpers/currentuser).
func Auth(parse TokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		token = strings.TrimSpace(token)
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			response.Error(c, apperror.Unauthorized("Token tidak ditemukan, silakan login"))
			return
		}
		u, err := parse(token)
		if err != nil {
			response.Error(c, apperror.Unauthorized("Token tidak valid atau sudah kedaluwarsa"))
			return
		}
		currentuser.Set(c, u)
		c.Next()
	}
}
