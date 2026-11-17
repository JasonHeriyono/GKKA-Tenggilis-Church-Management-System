package schedule

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/helpers/response"
)

// maxUploadSize: batas ukuran file import (5 MB).
const maxUploadSize = 5 << 20

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mendaftarkan endpoint jadwal. requireLogin = middleware.Auth dan superAdmin =
// RequireRole(Super Admin) dari routes/routes.go.
func RegisterRoutes(rg *gin.RouterGroup, svc *Service, requireLogin, superAdmin gin.HandlerFunc) {
	h := NewHandler(svc)
	g := rg.Group("/schedules", requireLogin, superAdmin)
	{
		g.POST("/import/preview", h.PreviewImport)
		g.POST("/import", h.ImportSchedules)
		g.GET("", h.ListKegiatan)
	}
	

	// publik (tanpa login): jemaat dan tamu bisa melihat jadwal kegiatan di aplikasi
	rg.GET("/kegiatan", h.ListKegiatan)
	rg.GET("/kegiatan/:id", h.GetKegiatan)

	// ubah data kegiatan dan atur petugasnya: khusus Super Admin
	kegiatan := rg.Group("/kegiatan", requireLogin, superAdmin)
	{
		kegiatan.POST("", h.CreateKegiatan)
		kegiatan.PUT("/:id", h.UpdateKegiatan)
		kegiatan.DELETE("/:id", h.DeleteKegiatan)

		kegiatan.GET("/:id/petugas", h.ListPetugas)
		kegiatan.POST("/:id/petugas", h.TugaskanPetugas)
		kegiatan.DELETE("/:id/petugas/:id_penjadwalan", h.HapusPetugas)
	}

	// tugas pelayanan milik sendiri: cukup login (Jemaat, Usher, dan role lain)
	rg.GET("/jadwal/saya", requireLogin, h.JadwalSaya)
}

// PreviewImport godoc
//
//	@Summary	Preview import jadwal dari template Excel
//	@Description	Memvalidasi file per baris dan mendeteksi bentrok waktu. Tidak menyimpan data.
//	@Tags		schedule
//	@Accept		multipart/form-data
//	@Produce	json
//	@Security	BearerAuth
//	@Param		file	formData	file	true	"Template Excel (.xlsx), maks 5 MB"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Failure	413		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/schedules/import/preview [post]
func (h *Handler) PreviewImport(c *gin.Context) {
	// batasi body sebelum dibaca; sisakan ruang untuk overhead multipart
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(1<<20))

	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("File wajib diunggah pada field \"file\" (maks 5 MB)"))
		return
	}
	if fh.Size > maxUploadSize {
		response.Error(c, apperror.New(http.StatusRequestEntityTooLarge, "Ukuran file maksimal 5 MB"))
		return
	}
	if strings.ToLower(filepath.Ext(fh.Filename)) != ".xlsx" {
		response.Error(c, apperror.BadRequest("Format file harus .xlsx"))
		return
	}

	f, err := fh.Open()
	if err != nil {
		response.Error(c, apperror.Internal(err))
		return
	}
	defer f.Close()

	result, err := h.svc.PreviewImport(f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result, "Preview import jadwal")
}

// ListKegiatan godoc
//
//	@Summary	Daftar kegiatan (publik)
//	@Tags		kegiatan
//	@Produce	json
//	@Param		dari	query		string	false	"Tanggal awal, contoh 2026-11-01"
//	@Param		sampai	query		string	false	"Tanggal akhir, contoh 2026-11-30"
//	@Param		q		query		string	false	"Cari nama kegiatan"
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"tanggal | nama_kegiatan | created_at, awali - untuk DESC"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Router		/kegiatan [get]
func (h *Handler) ListKegiatan(c *gin.Context) {
	params, err := pagination.FromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter := KegiatanQuery{
		Params: params,
		Dari:   c.Query("dari"),
		Sampai: c.Query("sampai"),
		Q:      c.Query("q"),
	}

	hasil, err := h.svc.ListKegiatan(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := pagination.CekHalaman(params, hasil.Meta.Total); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// GetKegiatan godoc
//
//	@Summary	Detail kegiatan (publik)
//	@Tags		kegiatan
//	@Produce	json
//	@Param		id	path		int	true	"ID kegiatan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/kegiatan/{id} [get]
func (h *Handler) GetKegiatan(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	hasil, err := h.svc.GetKegiatan(c.Request.Context(), kegiatanID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// CreateKegiatan godoc
//
//	@Summary	Tambah kegiatan (Super Admin)
//	@Tags		kegiatan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		KegiatanRequest	true	"Data kegiatan"
//	@Success	201		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/kegiatan [post]
func (h *Handler) CreateKegiatan(c *gin.Context) {
	// isi req dari JSON body; kalau JSON rusak -> 400, kalau tidak sesuai tag binding di dto.go -> 422
	var req KegiatanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	hasil, err := h.svc.CreateKegiatan(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Kegiatan berhasil dibuat")
}

// UpdateKegiatan godoc
//
//	@Summary	Ubah kegiatan (Super Admin)
//	@Tags		kegiatan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID kegiatan"
//	@Param		body	body		KegiatanRequest	true	"Data kegiatan"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/kegiatan/{id} [put]
func (h *Handler) UpdateKegiatan(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req KegiatanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	hasil, err := h.svc.UpdateKegiatan(c.Request.Context(), kegiatanID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Kegiatan diperbarui")
}

// DeleteKegiatan godoc
//
//	@Summary	Hapus kegiatan (Super Admin)
//	@Tags		kegiatan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID kegiatan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/kegiatan/{id} [delete]
func (h *Handler) DeleteKegiatan(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteKegiatan(c.Request.Context(), kegiatanID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Kegiatan dihapus")
}

// ListPetugas godoc
//
//	@Summary	Daftar petugas satu kegiatan (Super Admin)
//	@Tags		kegiatan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID kegiatan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/kegiatan/{id}/petugas [get]
func (h *Handler) ListPetugas(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	hasil, err := h.svc.ListPetugas(c.Request.Context(), kegiatanID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// TugaskanPetugas godoc
//
//	@Summary		Tugaskan jemaat ke kegiatan (Super Admin)
//	@Description	Jemaat harus aktif dan punya keahlian itu (sudah diverifikasi). Bentrok waktu hanya jadi peringatan di field peringatan.
//	@Tags			kegiatan
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"ID kegiatan"
//	@Param			body	body		PenugasanRequest	true	"Jemaat dan keahlian"
//	@Success		201		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/kegiatan/{id}/petugas [post]
func (h *Handler) TugaskanPetugas(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req PenugasanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	hasil, err := h.svc.TugaskanPetugas(c.Request.Context(), kegiatanID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Petugas berhasil ditugaskan")
}

// HapusPetugas godoc
//
//	@Summary	Batalkan penugasan petugas (Super Admin)
//	@Tags		kegiatan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id				path		int	true	"ID kegiatan"
//	@Param		id_penjadwalan	path		int	true	"ID penugasan"
//	@Success	200				{object}	response.Body
//	@Failure	404				{object}	response.Body
//	@Router		/kegiatan/{id}/petugas/{id_penjadwalan} [delete]
func (h *Handler) HapusPetugas(c *gin.Context) {
	kegiatanID, ok := parseID(c, "id")
	if !ok {
		return
	}
	penjadwalanID, ok := parseID(c, "id_penjadwalan")
	if !ok {
		return
	}

	if err := h.svc.HapusPetugas(c.Request.Context(), kegiatanID, penjadwalanID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Penugasan dibatalkan")
}

// JadwalSaya godoc
//
//	@Summary	Tugas pelayanan milik saya
//	@Tags		jadwal
//	@Produce	json
//	@Security	BearerAuth
//	@Param		dari	query		string	false	"Tanggal awal, contoh 2026-11-01"
//	@Param		sampai	query		string	false	"Tanggal akhir, contoh 2026-11-30"
//	@Param		page	query		int		false	"Halaman (default 1)"
//	@Param		limit	query		int		false	"Jumlah per halaman (default 20, maks 100)"
//	@Param		sort	query		string	false	"tanggal, awali - untuk DESC"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Failure	401		{object}	response.Body
//	@Router		/jadwal/saya [get]
func (h *Handler) JadwalSaya(c *gin.Context) {
	user, ok := currentuser.From(c)
	if !ok {
		response.Error(c, apperror.Unauthorized("Silakan login terlebih dahulu"))
		return
	}

	params, err := pagination.FromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter := JadwalSayaQuery{
		Params: params,
		Dari:   c.Query("dari"),
		Sampai: c.Query("sampai"),
	}

	hasil, err := h.svc.JadwalSaya(c.Request.Context(), user.ID, filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := pagination.CekHalaman(params, hasil.Meta.Total); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// parseID ambil ID dari URL, contoh /kegiatan/5 -> 5.
// Kalau bukan angka positif, langsung kirim 400 dan kembalikan ok = false (handler tinggal return).
func parseID(c *gin.Context, namaParam string) (uint, bool) {
	angka, err := strconv.ParseUint(c.Param(namaParam), 10, 32)
	if err != nil || angka == 0 {
		response.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return uint(angka), true
}

// ImportSchedules godoc
//
//	@Summary	Simpan import jadwal kegiatan (Super Admin)
//	@Tags		schedule
//	@Accept		multipart/form-data
//	@Produce	json
//	@Security	BearerAuth
//	@Param		file	formData	file	true	"Template Excel (.xlsx), maks 5 MB"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Router		/schedules/import [post]
func (h *Handler) ImportSchedules(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("File wajib diunggah pada field \"file\""))
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Error(c, apperror.Internal(err))
		return
	}
	defer f.Close()

	result, err := h.svc.ImportSchedules(c.Request.Context(), f)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, result, "Jadwal berhasil diimport")
}

