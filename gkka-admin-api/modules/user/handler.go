package user

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/helpers/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mendaftarkan endpoint pengelolaan user. guards diisi middleware Auth lalu
// RequireRole(Super Admin) dari routes/routes.go.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, guards ...gin.HandlerFunc) {
	g := rg.Group("/users", guards...)
	{
		g.GET("", h.List)
		g.POST("", h.Create)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.PATCH("/:id/role", h.ChangeRole)
		g.PATCH("/:id/status", h.SetActive)
		g.PUT("/:id/password", h.SetPassword)
	}
}

// List godoc
//
//	@Summary	Daftar user (Super Admin)
//	@Tags		user
//	@Produce	json
//	@Security	BearerAuth
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"nama | kode_jemaat | created_at, awali - untuk DESC"
//	@Param		q					query		string	false	"Cari nama, kode jemaat, email, atau no. telepon"
//	@Param		role				query		string	false	"Filter role"
//	@Param		status_verifikasi	query		string	false	"BELUM_TERVERIFIKASI | MENUNGGU_VERIFIKASI | TERVERIFIKASI"
//	@Success	200		{object}	response.Body
//	@Router		/users [get]
func (h *Handler) List(c *gin.Context) {
	params, err := pagination.FromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.List(c.Request.Context(), ListQuery{
		Params:           params,
		Q:                c.Query("q"),
		Role:             c.Query("role"),
		StatusVerifikasi: c.Query("status_verifikasi"),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "")
}

// Get godoc
//
//	@Summary	Detail user (Super Admin)
//	@Tags		user
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID user"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/users/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	res, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "")
}

// Create godoc
//
//	@Summary		Tambah jemaat manual (Super Admin)
//	@Description	Jemaat yang ditambah admin langsung TERVERIFIKASI. kode_jemaat kosong = dibuat otomatis. Password kosong = belum bisa login.
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		CreateRequest	true	"Data user"
//	@Success	201		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/users [post]
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, res, "User berhasil dibuat")
}

// Update godoc
//
//	@Summary		Edit data jemaat dan data dirinya (Super Admin)
//	@Description	Semua isian identitas ditimpa (kirim form lengkap). kode_jemaat kosong = tidak diubah.
//	@Tags			user
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"ID user"
//	@Param			body	body		UpdateRequest	true	"Data user"
//	@Success		200		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/users/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "Data jemaat diperbarui")
}

// Delete godoc
//
//	@Summary		Hapus jemaat (Super Admin)
//	@Description	Hapus sementara: riwayat absensi dan pelayanan tetap ada, email/no HP dibebaskan, semua sesi login dicabut.
//	@Tags			user
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"ID user"
//	@Success		200	{object}	response.Body
//	@Failure		400	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/users/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := actorOf(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor.ID, id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Jemaat dihapus")
}

// ChangeRole godoc
//
//	@Summary	Ubah role user (Super Admin)
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int			true	"ID user"
//	@Param		body	body		RoleRequest	true	"Role baru"
//	@Success	200		{object}	response.Body
//	@Router		/users/{id}/role [patch]
func (h *Handler) ChangeRole(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := actorOf(c)
	if !ok {
		return
	}
	var req RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.ChangeRole(c.Request.Context(), actor.ID, id, req.Role)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "Role diperbarui")
}

// SetActive godoc
//
//	@Summary	Aktifkan / nonaktifkan akun (Super Admin)
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID user"
//	@Param		body	body		StatusRequest	true	"Status"
//	@Success	200		{object}	response.Body
//	@Router		/users/{id}/status [patch]
func (h *Handler) SetActive(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	actor, ok := actorOf(c)
	if !ok {
		return
	}
	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	res, err := h.svc.SetActive(c.Request.Context(), actor.ID, id, *req.IsAktif)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res, "Status akun diperbarui")
}

// SetPassword godoc
//
//	@Summary	Atur password user dan cabut semua sesinya (Super Admin)
//	@Tags		user
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID user"
//	@Param		body	body		PasswordRequest	true	"Password baru"
//	@Success	200		{object}	response.Body
//	@Router		/users/{id}/password [put]
func (h *Handler) SetPassword(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req PasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.svc.SetPassword(c.Request.Context(), id, req.Password); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Password diperbarui")
}

func parseID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || n == 0 {
		response.Error(c, apperror.BadRequest("ID user tidak valid"))
		return 0, false
	}
	return uint(n), true
}

func actorOf(c *gin.Context) (currentuser.User, bool) {
	u, ok := currentuser.From(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
	}
	return u, ok
}
