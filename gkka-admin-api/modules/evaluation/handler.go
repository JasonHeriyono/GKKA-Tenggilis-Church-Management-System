package evaluation

import (
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/response"
	"sim-g-backend/middleware"
)

// Handler menerima request HTTP, memanggil service, lalu mengirim response.
// Aturan bisnis tidak ditulis di sini, semuanya ada di service.go.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mendaftarkan URL endpoint pertanyaan beserta fungsi yang menanganinya.
// Pengecekan sebelum masuk handler dikirim dari routes.go:
// requireLogin = harus login, bolehBaca = Majelis atau Super Admin, superAdmin = khusus Super Admin.
func RegisterRoutes(router *gin.RouterGroup, handler *Handler, requireLogin, bolehBaca, superAdmin gin.HandlerFunc) {
	// publik (tanpa login): app jemaat dan tamu butuh link form evaluasi
	router.GET("/evaluasi/form", handler.FormPublik)
	// dibatasi per IP supaya kode tidak ditebak terus-menerus. Longgar karena jemaat di WiFi gereja
	// memakai IP yang sama: maks 100 percobaan sekaligus, lalu 5 per detik.
	router.POST("/evaluasi/kode", middleware.RateLimit(rate.Every(200*time.Millisecond), 100), handler.CekKode)

	// lihat pertanyaan, header form, dan jadwal kuesioner: Majelis (hanya baca) dan Super Admin
	router.GET("/pertanyaan", requireLogin, bolehBaca, handler.ListPertanyaan)
	router.GET("/pertanyaan/:id", requireLogin, bolehBaca, handler.GetPertanyaan)
	router.GET("/pertanyaan/header", requireLogin, bolehBaca, handler.AmbilHeader)
	router.GET("/kuesioner", requireLogin, bolehBaca, handler.ListKuesioner)

	// ubah pertanyaan dan sinkron Google Form: khusus Super Admin
	pertanyaan := router.Group("/pertanyaan", requireLogin, superAdmin)
	{
		pertanyaan.POST("", handler.CreatePertanyaan)
		pertanyaan.PUT("/:id", handler.UpdatePertanyaan)
		pertanyaan.PATCH("/:id/aktif", handler.SetAktifPertanyaan)
		pertanyaan.DELETE("/:id", handler.DeletePertanyaan)
		pertanyaan.POST("/gform", handler.SinkronGoogle)
		pertanyaan.POST("/import/preview", handler.PreviewImportPertanyaan)
		pertanyaan.POST("/import", handler.ImportPertanyaan)
		pertanyaan.PUT("/header", handler.UpdateHeader)
	}

	// ubah jadwal buka/tutup form evaluasi: khusus Super Admin (lihat daftarnya boleh Majelis, di atas)
	kuesioner := router.Group("/kuesioner", requireLogin, superAdmin)
	{
		kuesioner.POST("", handler.CreateKuesioner)
		kuesioner.POST("/tarik-respon", handler.TarikRespon)
		kuesioner.PUT("/:id/jadwal", handler.UbahJadwalKuesioner)
	}
}

// Catatan: komentar berawalan @ di atas tiap fungsi dipakai `go tool swag init`
// untuk membuat dokumentasi Swagger. Jangan dihapus.

// ListPertanyaan godoc
//
//	@Summary	Daftar pertanyaan evaluasi (Majelis, Super Admin)
//	@Tags		pertanyaan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		jenis	query		string	false	"MINGGUAN | TAHUNAN"
//	@Param		aktif	query		bool	false	"true = hanya yang aktif, false = hanya yang nonaktif"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Router		/pertanyaan [get]
func (handler *Handler) ListPertanyaan(c *gin.Context) {
	// contoh URL: /pertanyaan?jenis=MINGGUAN&aktif=true (keduanya boleh dikosongkan)
	aktif, ok := bacaFilterAktif(c)
	if !ok {
		return
	}
	filter := PertanyaanQuery{Jenis: c.Query("jenis"), Aktif: aktif}

	hasil, err := handler.service.ListPertanyaan(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// GetPertanyaan godoc
//
//	@Summary	Detail pertanyaan beserta opsi (Majelis, Super Admin)
//	@Tags		pertanyaan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID pertanyaan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/pertanyaan/{id} [get]
func (handler *Handler) GetPertanyaan(c *gin.Context) {
	pertanyaanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	hasil, err := handler.service.GetPertanyaan(c.Request.Context(), pertanyaanID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// CreatePertanyaan godoc
//
//	@Summary	Tambah pertanyaan evaluasi (Super Admin)
//	@Tags		pertanyaan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		PertanyaanRequest	true	"Data pertanyaan"
//	@Success	201		{object}	response.Body
//	@Failure	409		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/pertanyaan [post]
func (handler *Handler) CreatePertanyaan(c *gin.Context) {
	// isi req dari JSON body; kalau JSON rusak -> 400, kalau tidak sesuai tag binding di dto.go -> 422
	var req PertanyaanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	hasil, err := handler.service.CreatePertanyaan(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Pertanyaan berhasil dibuat")
}

// UpdatePertanyaan godoc
//
//	@Summary	Ubah pertanyaan evaluasi (Super Admin)
//	@Tags		pertanyaan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int					true	"ID pertanyaan"
//	@Param		body	body		PertanyaanRequest	true	"Data pertanyaan"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/pertanyaan/{id} [put]
func (handler *Handler) UpdatePertanyaan(c *gin.Context) {
	pertanyaanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req PertanyaanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	hasil, err := handler.service.UpdatePertanyaan(c.Request.Context(), pertanyaanID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Pertanyaan diperbarui")
}

// SetAktifPertanyaan godoc
//
//	@Summary	Aktifkan / nonaktifkan pertanyaan (Super Admin)
//	@Tags		pertanyaan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int				true	"ID pertanyaan"
//	@Param		body	body		AktifRequest	true	"Status aktif"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Router		/pertanyaan/{id}/aktif [patch]
func (handler *Handler) SetAktifPertanyaan(c *gin.Context) {
	pertanyaanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req AktifRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	// IsAktif bertipe *bool (pointer), jadi pakai *req.IsAktif untuk mengambil nilai true/false-nya
	hasil, err := handler.service.SetAktifPertanyaan(c.Request.Context(), pertanyaanID, *req.IsAktif)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Status pertanyaan diperbarui")
}

// DeletePertanyaan godoc
//
//	@Summary	Hapus pertanyaan evaluasi (Super Admin)
//	@Tags		pertanyaan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id	path		int	true	"ID pertanyaan"
//	@Success	200	{object}	response.Body
//	@Failure	404	{object}	response.Body
//	@Router		/pertanyaan/{id} [delete]
func (handler *Handler) DeletePertanyaan(c *gin.Context) {
	pertanyaanID, ok := parseID(c, "id")
	if !ok {
		return
	}

	if err := handler.service.DeletePertanyaan(c.Request.Context(), pertanyaanID); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Pertanyaan dihapus")
}

// FormPublik godoc
//
//	@Summary	Link Google Form evaluasi untuk jemaat dan tamu (publik)
//	@Tags		evaluasi
//	@Produce	json
//	@Param		jenis	query		string	true	"MINGGUAN | TAHUNAN"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Failure	502		{object}	response.Body
//	@Failure	503		{object}	response.Body
//	@Router		/evaluasi/form [get]
func (handler *Handler) FormPublik(c *gin.Context) {
	// contoh URL: GET /evaluasi/form?jenis=MINGGUAN
	hasil, err := handler.service.FormPublik(c.Request.Context(), c.Query("jenis"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// ListKuesioner godoc
//
//	@Summary	Daftar jadwal kuesioner evaluasi, terbaru di atas (Super Admin)
//	@Tags		kuesioner
//	@Produce	json
//	@Security	BearerAuth
//	@Param		jenis	query		string	false	"MINGGUAN | TAHUNAN"
//	@Success	200		{object}	response.Body
//	@Router		/kuesioner [get]
func (handler *Handler) ListKuesioner(c *gin.Context) {
	hasil, err := handler.service.ListKuesioner(c.Request.Context(), c.Query("jenis"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "")
}

// CreateKuesioner godoc
//
//	@Summary		Buat jadwal kuesioner evaluasi (Super Admin)
//	@Description	MINGGUAN: id_kegiatan wajib; jadwal kosong = dibuka saat kegiatan selesai (jam mulai + durasi bawaan kalau jam selesai kosong), ditutup H+N pada jam tutup (lihat GET /pengaturan). Kode akses dibuat otomatis. TAHUNAN: tahun dan jadwal wajib. Format jam: "2026-11-01 10:30:00".
//	@Tags			kuesioner
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		KuesionerRequest	true	"Data kuesioner"
//	@Success		201		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/kuesioner [post]
func (handler *Handler) CreateKuesioner(c *gin.Context) {
	var req KuesionerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.CreateKuesioner(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, hasil, "Jadwal kuesioner dibuat")
}

// UbahJadwalKuesioner godoc
//
//	@Summary		Ubah jam buka/tutup kuesioner evaluasi (Super Admin)
//	@Description	status BUKA = buka sekarang, TUTUP = tutup sekarang. Atau isi dibuka_pada / ditutup_pada ("2026-11-01 10:30:00"); yang tidak dikirim tetap jam lama.
//	@Tags			kuesioner
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		id		path		int						true	"ID kuesioner"
//	@Param		body	body		JadwalKuesionerRequest	true	"Jadwal baru"
//	@Success	200		{object}	response.Body
//	@Failure	404		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/kuesioner/{id}/jadwal [put]
func (handler *Handler) UbahJadwalKuesioner(c *gin.Context) {
	kuesionerID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req JadwalKuesionerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.UbahJadwalKuesioner(c.Request.Context(), kuesionerID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Jadwal kuesioner diperbarui")
}

// SinkronGoogle godoc
//
//	@Summary	Samakan pertanyaan aktif ke Google Form (Super Admin)
//	@Tags		pertanyaan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		jenis	query		string	true	"MINGGUAN | TAHUNAN"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Failure	502		{object}	response.Body
//	@Failure	503		{object}	response.Body
//	@Router		/pertanyaan/gform [post]
func (handler *Handler) SinkronGoogle(c *gin.Context) {
	// contoh URL: POST /pertanyaan/gform?jenis=MINGGUAN
	hasil, err := handler.service.SinkronGoogle(c.Request.Context(), c.Query("jenis"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Google Form sudah disamakan")
}

// PreviewImportPertanyaan godoc
//
//	@Summary		Preview import pertanyaan dari template Excel (Super Admin)
//	@Description	Cek isi sheet "Pertanyaan" per baris dan tunjukkan pertanyaan yang BARU, SAMA (dipakai ulang), dan yang akan dinonaktifkan. Tidak menyimpan data.
//	@Tags			pertanyaan
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Template pertanyaan (.xlsx), maks 5 MB"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		413		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/pertanyaan/import/preview [post]
func (handler *Handler) PreviewImportPertanyaan(c *gin.Context) {
	file, ok := bukaFileExcel(c)
	if !ok {
		return
	}
	defer file.Close()

	hasil, err := handler.service.PreviewImportPertanyaan(c.Request.Context(), file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Preview import pertanyaan")
}

// ImportPertanyaan godoc
//
//	@Summary		Simpan import pertanyaan dari template Excel (Super Admin)
//	@Description	Ganti set pertanyaan untuk jenis yang ada di file: pertanyaan sama dipakai ulang, yang baru dibuat, yang tidak ada di file dinonaktifkan. Semua atau tidak sama sekali.
//	@Tags			pertanyaan
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Template pertanyaan (.xlsx), maks 5 MB"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		413		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Router			/pertanyaan/import [post]
func (handler *Handler) ImportPertanyaan(c *gin.Context) {
	file, ok := bukaFileExcel(c)
	if !ok {
		return
	}
	defer file.Close()

	hasil, err := handler.service.ImportPertanyaan(c.Request.Context(), file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Pertanyaan berhasil diimport")
}

// maksUkuranUpload: batas ukuran file Excel yang diupload (5 MB).
const maksUkuranUpload = 5 << 20

// bukaFileExcel ambil file dari field "file" lalu cek ukuran dan formatnya.
// Kalau ada yang salah, langsung kirim error dan kembalikan ok = false (handler tinggal return).
func bukaFileExcel(c *gin.Context) (multipart.File, bool) {
	// batasi besar body sebelum dibaca; sisakan 1 MB untuk data tambahan multipart
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maksUkuranUpload+(1<<20))

	dataFile, err := c.FormFile("file")
	if err != nil {
		response.Error(c, apperror.BadRequest("File wajib diunggah pada field \"file\" (maks 5 MB)"))
		return nil, false
	}
	if dataFile.Size > maksUkuranUpload {
		response.Error(c, apperror.New(http.StatusRequestEntityTooLarge, "Ukuran file maksimal 5 MB"))
		return nil, false
	}
	if strings.ToLower(filepath.Ext(dataFile.Filename)) != ".xlsx" {
		response.Error(c, apperror.BadRequest("Format file harus .xlsx"))
		return nil, false
	}

	file, err := dataFile.Open()
	if err != nil {
		response.Error(c, apperror.Internal(err))
		return nil, false
	}
	return file, true
}

// parseID ambil ID dari URL, contoh /pertanyaan/5 -> 5.
// Kalau bukan angka positif, langsung kirim 400 dan kembalikan ok = false (handler tinggal return).
func parseID(c *gin.Context, namaParam string) (uint, bool) {
	angka, err := strconv.ParseUint(c.Param(namaParam), 10, 32)
	if err != nil || angka == 0 {
		response.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return uint(angka), true
}

// bacaFilterAktif baca ?aktif= dari URL.
// Kosong -> nil (tidak difilter). Bukan true/false -> kirim 400 dan kembalikan ok = false.
func bacaFilterAktif(c *gin.Context) (*bool, bool) {
	teks := c.Query("aktif")
	if teks == "" {
		return nil, true
	}

	aktif, err := strconv.ParseBool(teks)
	if err != nil {
		response.Error(c, apperror.BadRequest("Parameter aktif harus true atau false"))
		return nil, false
	}
	return &aktif, true
}

// UpdateHeader godoc
//
//	@Summary	Ubah header form evaluasi (Super Admin)
//	@Tags		pertanyaan
//	@Accept		json
//	@Produce	json
//	@Security	BearerAuth
//	@Param		body	body		UpdateHeaderRequest	true	"Data header form"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Failure	422		{object}	response.Body
//	@Router		/pertanyaan/header [put]
func (handler *Handler) UpdateHeader(c *gin.Context) {
	var req UpdateHeaderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}

	err := handler.service.UpdateHeader(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, nil, "Header form evaluasi berhasil diperbarui")
}

// AmbilHeader godoc
//
//	@Summary	Template header form evaluasi (Super Admin)
//	@Tags		pertanyaan
//	@Produce	json
//	@Security	BearerAuth
//	@Param		jenis	query		string	true	"MINGGUAN | TAHUNAN"
//	@Success	200		{object}	response.Body
//	@Failure	400		{object}	response.Body
//	@Router		/pertanyaan/header [get]
func (handler *Handler) AmbilHeader(c *gin.Context) {
	hasil, err := handler.service.AmbilHeader(c.Request.Context(), c.Query("jenis"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Template header form evaluasi")
}
