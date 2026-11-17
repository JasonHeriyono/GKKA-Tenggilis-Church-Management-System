package auth

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/response"
	"sim-g-backend/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mendaftarkan endpoint auth. requireLogin = middleware.Auth(...) dari routes/routes.go.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, requireLogin gin.HandlerFunc) {
	g := rg.Group("/auth")
	{
		// batas per IP cukup longgar karena satu wifi gereja bisa dipakai banyak jemaat sekaligus
		g.POST("/login", middleware.RateLimit(rate.Every(6*time.Second), 10), h.Login)
		g.POST("/refresh", middleware.RateLimit(rate.Every(500*time.Millisecond), 30), h.Refresh)
		g.POST("/logout", h.Logout)
		g.GET("/me", requireLogin, h.Me)
	}
}

// Login godoc
//
//	@Summary	Login dengan email, kode jemaat, atau no. telepon
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		LoginRequest	true	"Kredensial"
//	@Success	200		{object}	response.Body
//	@Failure	401		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		if apperror.From(err).Status == http.StatusUnauthorized {
			slog.WarnContext(c.Request.Context(), "login gagal",
				"ip", c.ClientIP(), "request_id", c.GetString(middleware.RequestIDKey))
		}
		response.Error(c, err)
		return
	}
	response.OK(c, res, "Login berhasil")
}

// Refresh godoc
//
//	@Summary	Tukar refresh token dengan token baru (rotasi)
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		RefreshRequest	true	"Refresh token"
//	@Success	200		{object}	response.Body
//	@Failure	401		{object}	response.Body
//	@Router		/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "Token diperbarui")
}

// Logout godoc
//
//	@Summary	Logout (cabut refresh token)
//	@Tags		auth
//	@Accept		json
//	@Produce	json
//	@Param		body	body		RefreshRequest	true	"Refresh token"
//	@Success	200		{object}	response.Body
//	@Router		/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Logout berhasil")
}

// Me godoc
//
//	@Summary	Profil user yang sedang login
//	@Tags		auth
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Body
//	@Failure	401	{object}	response.Body
//	@Router		/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	u, ok := currentuser.From(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
		return
	}
	info, err := h.svc.Me(c.Request.Context(), u.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, info, "")
}
