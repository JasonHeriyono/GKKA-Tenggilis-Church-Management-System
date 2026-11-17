package pengaturan

import (
	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// RegisterRoutes: semua endpoint pengaturan khusus Super Admin (guards dari routes/routes.go).
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, guards ...gin.HandlerFunc) {
	pengaturan := router.Group("/pengaturan", guards...)
	{
		pengaturan.GET("", handler.List)
		pengaturan.PUT("", handler.Simpan)
	}
}

// List godoc
//
//	@Summary	Daftar pengaturan admin beserta nilai bawaan (Super Admin)
//	@Tags		pengaturan
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Body
//	@Router		/pengaturan [get]
func (handler *Handler) List(c *gin.Context) {
	hasil, err := handler.service.List(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Simpan godoc
//
//	@Summary		Ubah pengaturan admin (Super Admin)
//	@Description	Hanya kunci yang dikirim yang diubah. Berlaku untuk kuesioner yang dibuat setelahnya.
//	@Tags			pengaturan
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UpdatePengaturanRequest	true	"Pengaturan baru"
//	@Success		200		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/pengaturan [put]
func (handler *Handler) Simpan(c *gin.Context) {
	var req UpdatePengaturanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	admin, ok := currentuser.From(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
		return
	}
	hasil, err := handler.service.Simpan(c.Request.Context(), req, admin.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Pengaturan disimpan")
}
