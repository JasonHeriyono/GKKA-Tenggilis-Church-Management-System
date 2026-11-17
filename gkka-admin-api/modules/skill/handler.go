package skill

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/helpers/response"
)

// Handler: penghubung HTTP <-> service (baca URL/JSON, kirim response).
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// RegisterRoutes mendaftarkan endpoint keahlian. guards diisi middleware Auth lalu
// RequireRole(Super Admin) dari routes/routes.go.
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, guards ...gin.HandlerFunc) {
	// publik (tanpa login): form signup butuh daftar keahlian untuk dipilih
	router.GET("/keahlian", handler.List)

	keahlian := router.Group("/keahlian", guards...)
	{
		keahlian.POST("", handler.Create)
		keahlian.GET("/:id", handler.Get)
		keahlian.PUT("/:id", handler.Update)
		keahlian.DELETE("/:id", handler.Delete)
		keahlian.POST("/:id/jemaat", handler.AddJemaat)
		keahlian.DELETE("/:id/jemaat/:userId", handler.RemoveJemaat)
	}

	// keahlian milik satu jemaat (sisi user)
	userKeahlian := router.Group("/users/:id/keahlian", guards...)
	{
		userKeahlian.GET("", handler.ListByUser)
		userKeahlian.PUT("", handler.ReplaceByUser)
	}
}

// List godoc
//
//	@Summary	Daftar keahlian (publik, dipakai form signup)
//	@Tags		keahlian
//	@Produce	json
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"nama_keahlian | created_at, awali - untuk DESC"
//	@Param		q		query		string	false	"Cari nama keahlian"
//	@Success	200		{object}	response.Body
//	@Router		/keahlian [get]
func (handler *Handler) List(c *gin.Context) {
	params, err := pagination.FromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	hasil, err := handler.service.List(c.Request.Context(), ListQuery{
		Params: params,
		Q:      c.Query("q"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Get godoc
//
//	@Summary	Detail keahlian beserta jemaat pemiliknya (Super Admin)
//	@Tags		keahlian
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID keahlian"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/keahlian/{id} [get]
func (handler *Handler) Get(c *gin.Context) {
	keahlianID, ok := parseID(c, "id")
	if !ok {
		return
	}
	hasil, err := handler.service.Get(c.Request.Context(), keahlianID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Create godoc
//
//	@Summary	Tambah keahlian (Super Admin)
//	@Tags		keahlian
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		SaveRequest	true	"Data keahlian"
//	@Success	201		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/keahlian [post]
func (handler *Handler) Create(c *gin.Context) {
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Keahlian berhasil dibuat")
}

// Update godoc
//
//	@Summary	Ubah nama keahlian (Super Admin; keahlian sistem tidak bisa diubah)
//	@Tags		keahlian
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int			true	"ID keahlian"
//	@Param		body	body		SaveRequest	true	"Data keahlian"
//	@Success	200		{object}	response.Body
//	@Failure	403		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Router		/keahlian/{id} [put]
func (handler *Handler) Update(c *gin.Context) {
	keahlianID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.Update(c.Request.Context(), keahlianID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Keahlian diperbarui")
}

// Delete godoc
//
//	@Summary	Hapus keahlian (soft delete; Super Admin)
//	@Tags		keahlian
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID keahlian"
//	@Success	200	{object}	response.Body
//	@Failure	403	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Failure	409	{object}	response.Body
//	@Router		/keahlian/{id} [delete]
func (handler *Handler) Delete(c *gin.Context) {
	keahlianID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := handler.service.Delete(c.Request.Context(), keahlianID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Keahlian dihapus")
}

// AddJemaat godoc
//
//	@Summary	Beri keahlian ke satu jemaat (Super Admin)
//	@Tags		keahlian
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID keahlian"
//	@Param		body	body		JemaatRequest	true	"Jemaat"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/keahlian/{id}/jemaat [post]
func (handler *Handler) AddJemaat(c *gin.Context) {
	keahlianID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req JemaatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.AddJemaat(c.Request.Context(), keahlianID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Keahlian diberikan")
}

// RemoveJemaat godoc
//
//	@Summary	Lepas keahlian dari satu jemaat (Super Admin)
//	@Tags		keahlian
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int	true	"ID keahlian"
//	@Param		userId	path		int	true	"ID user"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Router		/keahlian/{id}/jemaat/{userId} [delete]
func (handler *Handler) RemoveJemaat(c *gin.Context) {
	keahlianID, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := handler.service.RemoveJemaat(c.Request.Context(), keahlianID, userID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Keahlian dilepas")
}

// ListByUser godoc
//
//	@Summary	Keahlian milik satu jemaat (Super Admin)
//	@Tags		keahlian
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID user"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/users/{id}/keahlian [get]
func (handler *Handler) ListByUser(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	hasil, err := handler.service.ListByUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// ReplaceByUser godoc
//
//	@Summary	Ganti seluruh keahlian satu jemaat (Super Admin)
//	@Tags		keahlian
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID user"
//	@Param		body	body		ReplaceRequest	true	"Daftar id keahlian (kosong = lepas semua)"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/users/{id}/keahlian [put]
func (handler *Handler) ReplaceByUser(c *gin.Context) {
	userID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req ReplaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.ReplaceByUser(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Keahlian jemaat diperbarui")
}

func parseID(c *gin.Context, namaParam string) (uint, bool) {
	angka, err := strconv.ParseUint(c.Param(namaParam), 10, 32)
	if err != nil || angka == 0 {
		response.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return uint(angka), true
}
