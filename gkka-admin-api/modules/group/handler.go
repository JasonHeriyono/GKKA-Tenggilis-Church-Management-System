package group

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

// RegisterRoutes mendaftarkan endpoint komunitas. guards diisi middleware Auth lalu
// RequireRole(Super Admin) dari routes/routes.go.
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, guards ...gin.HandlerFunc) {
	// publik (tanpa login): hanya nama dan deskripsi komunitas, daftar anggota tetap khusus Super Admin
	router.GET("/komunitas", handler.List)

	komunitas := router.Group("/komunitas", guards...)
	{
		komunitas.POST("", handler.Create)
		komunitas.GET("/:id", handler.Get)
		komunitas.PUT("/:id", handler.Update)
		komunitas.DELETE("/:id", handler.Delete)
		komunitas.POST("/:id/anggota", handler.AddAnggota)
		komunitas.PATCH("/:id/anggota/:userId", handler.SetKetua)
		komunitas.DELETE("/:id/anggota/:userId", handler.RemoveAnggota)
	}
}

// List godoc
//
//	@Summary	Daftar komunitas (publik)
//	@Tags		komunitas
//	@Produce	json
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"nama_komunitas | created_at, awali - untuk DESC"
//	@Param		q		query		string	false	"Cari nama komunitas"
//	@Success	200		{object}	response.Body
//	@Router		/komunitas [get]
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
//	@Summary	Detail komunitas beserta anggota (Super Admin)
//	@Tags		komunitas
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID komunitas"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/komunitas/{id} [get]
func (handler *Handler) Get(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	hasil, err := handler.service.Get(c.Request.Context(), komunitasID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Create godoc
//
//	@Summary	Tambah komunitas (Super Admin)
//	@Tags		komunitas
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		SaveRequest	true	"Data komunitas"
//	@Success	201		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/komunitas [post]
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
	response.Created(c, hasil, "Komunitas berhasil dibuat")
}

// Update godoc
//
//	@Summary	Ubah komunitas (Super Admin)
//	@Tags		komunitas
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int			true	"ID komunitas"
//	@Param		body	body		SaveRequest	true	"Data komunitas"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Router		/komunitas/{id} [put]
func (handler *Handler) Update(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.Update(c.Request.Context(), komunitasID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Komunitas diperbarui")
}

// Delete godoc
//
//	@Summary	Hapus komunitas (Super Admin)
//	@Tags		komunitas
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID komunitas"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/komunitas/{id} [delete]
func (handler *Handler) Delete(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := handler.service.Delete(c.Request.Context(), komunitasID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Komunitas dihapus")
}

// AddAnggota godoc
//
//	@Summary	Tambah anggota komunitas (Super Admin)
//	@Tags		komunitas
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID komunitas"
//	@Param		body	body		AnggotaRequest	true	"Anggota"
//	@Success	201		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Router		/komunitas/{id}/anggota [post]
func (handler *Handler) AddAnggota(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req AnggotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.AddAnggota(c.Request.Context(), komunitasID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Anggota ditambahkan")
}

// SetKetua godoc
//
//	@Summary	Jadikan / batalkan ketua komunitas (Super Admin)
//	@Tags		komunitas
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID komunitas"
//	@Param		userId	path		int				true	"ID user"
//	@Param		body	body		KetuaRequest	true	"Status ketua"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Router		/komunitas/{id}/anggota/{userId} [patch]
func (handler *Handler) SetKetua(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	var req KetuaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.SetKetua(c.Request.Context(), komunitasID, userID, *req.IsKetua)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Status ketua diperbarui")
}

// RemoveAnggota godoc
//
//	@Summary	Keluarkan anggota dari komunitas (Super Admin)
//	@Tags		komunitas
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int	true	"ID komunitas"
//	@Param		userId	path		int	true	"ID user"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Router		/komunitas/{id}/anggota/{userId} [delete]
func (handler *Handler) RemoveAnggota(c *gin.Context) {
	komunitasID, ok := parseID(c, "id")
	if !ok {
		return
	}
	userID, ok := parseID(c, "userId")
	if !ok {
		return
	}
	if err := handler.service.RemoveAnggota(c.Request.Context(), komunitasID, userID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Anggota dikeluarkan")
}

// parseID membaca angka dari URL (mis. :id); kirim 400 jika bukan angka positif.
func parseID(c *gin.Context, namaParam string) (uint, bool) {
	angka, err := strconv.ParseUint(c.Param(namaParam), 10, 32)
	if err != nil || angka == 0 {
		response.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return uint(angka), true
}
