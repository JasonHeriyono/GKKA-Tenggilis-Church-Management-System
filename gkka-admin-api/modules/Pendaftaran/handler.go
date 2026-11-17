package pendaftaran

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/helpers/response"
	"sim-g-backend/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// RegisterRoutes mendaftarkan:
//   - PUBLIK /auth/signup (dibatasi per IP)
//   - LOGIN (role apa saja) /profil/saya dan /verifikasi
//   - SUPER ADMIN /pendaftaran/*
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, requireLogin, superAdmin gin.HandlerFunc) {
	// longgar karena jemaat bisa daftar bersama lewat WiFi gereja (IP sama): maks 50 sekaligus, lalu 1 per detik
	router.POST("/auth/signup", middleware.RateLimit(rate.Every(time.Second), 50), handler.Signup)

	router.GET("/profil/saya", requireLogin, handler.ProfilSaya)
	router.PUT("/profil/saya", requireLogin, handler.UbahProfilSaya)
	router.POST("/verifikasi", requireLogin, handler.AjukanVerifikasi)

	admin := router.Group("/pendaftaran", requireLogin, superAdmin)
	{
		admin.GET("", handler.List)
		admin.GET("/:id", handler.Get)
		admin.POST("/:id/setujui", handler.Setujui)
		admin.POST("/:id/tolak", handler.Tolak)
	}
}

// Signup godoc
//
//	@Summary		Buat akun jemaat baru (langsung masuk)
//	@Description	Akun langsung bisa dipakai dengan status BELUM_TERVERIFIKASI. Response sama seperti login (access_token, refresh_token, user). Email atau no. telepon wajib salah satu.
//	@Tags			pendaftaran
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SignupRequest	true	"Data signup"
//	@Success		201		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/auth/signup [post]
func (handler *Handler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.Signup(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Akun berhasil dibuat. Lengkapi data diri supaya bisa diverifikasi sekretariat.")
}

// ProfilSaya godoc
//
//	@Summary	Data diri sendiri + kelengkapan + status verifikasi
//	@Tags		pendaftaran
//	@Produce	json
//	@Security	BearerAuth
//	@Success	200	{object}	response.Body
//	@Router		/profil/saya [get]
func (handler *Handler) ProfilSaya(c *gin.Context) {
	user, ok := userLogin(c)
	if !ok {
		return
	}
	hasil, err := handler.service.ProfilSaya(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// UbahProfilSaya godoc
//
//	@Summary		Ubah data diri sendiri
//	@Description	Kirim form lengkap. NIK dan data keanggotaan diabaikan (hanya admin). Setelah TERVERIFIKASI, field di kolom_terkunci (data inti) juga diabaikan. Ditolak (409) saat pengajuan sedang diperiksa.
//	@Tags			pendaftaran
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		ProfilRequest	true	"Data diri"
//	@Success		200		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/profil/saya [put]
func (handler *Handler) UbahProfilSaya(c *gin.Context) {
	user, ok := userLogin(c)
	if !ok {
		return
	}
	var req ProfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.UbahProfilSaya(c.Request.Context(), user.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Data diri disimpan")
}

// AjukanVerifikasi godoc
//
//	@Summary		Ajukan verifikasi ke sekretariat
//	@Description	Data wajib (jenis kelamin, tempat & tanggal lahir, alamat KTP, status gereja) harus lengkap. Status akun jadi MENUNGGU_VERIFIKASI.
//	@Tags			pendaftaran
//	@Produce		json
//	@Security		BearerAuth
//	@Success		201	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Failure		422	{object}	response.Body
//	@Router			/verifikasi [post]
func (handler *Handler) AjukanVerifikasi(c *gin.Context) {
	user, ok := userLogin(c)
	if !ok {
		return
	}
	hasil, err := handler.service.AjukanVerifikasi(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Pengajuan verifikasi dikirim ke sekretariat")
}

// List godoc
//
//	@Summary	Daftar pengajuan verifikasi (Super Admin)
//	@Tags		pendaftaran
//	@Produce	json
//	@Security	BearerAuth
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"created_at | status, awali - untuk DESC"
//	@Param		status	query		string	false	"MENUNGGU | DISETUJUI | DITOLAK"
//	@Param		q		query		string	false	"Cari nama, email, atau no. telepon"
//	@Success	200		{object}	response.Body
//	@Router		/pendaftaran [get]
func (handler *Handler) List(c *gin.Context) {
	params, err := pagination.FromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	hasil, err := handler.service.List(c.Request.Context(), ListQuery{Params: params, Status: c.Query("status"), Q: c.Query("q")})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Get godoc
//
//	@Summary	Detail pengajuan verifikasi (Super Admin)
//	@Tags		pendaftaran
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID pengajuan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/pendaftaran/{id} [get]
func (handler *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	hasil, err := handler.service.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// Setujui godoc
//
//	@Summary		Setujui pengajuan verifikasi (Super Admin)
//	@Description	Akun jadi TERVERIFIKASI dan mendapat kode jemaat.
//	@Tags			pendaftaran
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"ID pengajuan"
//	@Param			body	body		SetujuiRequest	false	"Catatan"
//	@Success		200		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/pendaftaran/{id}/setujui [post]
func (handler *Handler) Setujui(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	admin, ok := userLogin(c)
	if !ok {
		return
	}
	var req SetujuiRequest
	if c.Request.ContentLength != 0 { // body boleh kosong
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BindError(c, err)
			return
		}
	}
	hasil, err := handler.service.Setujui(c.Request.Context(), admin.ID, id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Pengajuan disetujui")
}

// Tolak godoc
//
//	@Summary		Tolak pengajuan verifikasi (Super Admin)
//	@Description	Alasan wajib. Jemaat kembali BELUM_TERVERIFIKASI dan bisa mengajukan lagi.
//	@Tags			pendaftaran
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"ID pengajuan"
//	@Param			body	body		TolakRequest	true	"Alasan penolakan"
//	@Success		200		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/pendaftaran/{id}/tolak [post]
func (handler *Handler) Tolak(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	admin, ok := userLogin(c)
	if !ok {
		return
	}
	var req TolakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.Tolak(c.Request.Context(), admin.ID, id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Pengajuan ditolak")
}

func parseID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || n == 0 {
		response.Error(c, apperror.BadRequest("ID pengajuan tidak valid"))
		return 0, false
	}
	return uint(n), true
}

// userLogin: user dari access token (sudah dicek middleware Auth).
func userLogin(c *gin.Context) (currentuser.User, bool) {
	user, ok := currentuser.From(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
	}
	return user, ok
}
