package currentuser

import "github.com/gin-gonic/gin"

const ctxKey = "current_user"

// User adalah identitas yang diambil dari access token.
type User struct {
	ID   uint
	Role string // nama role, mis. models.RoleSuperAdmin
}

// Set dipanggil middleware Auth setelah token valid.
func Set(c *gin.Context, u User) { c.Set(ctxKey, u) }

// From mengambil user yang sedang login; ok=false jika request belum melewati middleware Auth.
func From(c *gin.Context) (User, bool) {
	v, exists := c.Get(ctxKey)
	if !exists {
		return User{}, false
	}
	u, ok := v.(User)
	return u, ok
}
