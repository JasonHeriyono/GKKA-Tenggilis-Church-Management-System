package evaluation

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"google.golang.org/api/forms/v1"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/response"
	"sim-g-backend/models"
)

// Kode akses: setiap kuesioner punya kode pendek (contoh "K7MP2X") yang diumumkan saat ibadah.
// Jemaat mengisi kode di app -> backend cek -> app membuka Google Form dengan kode sudah terisi.
// Google Forms API tidak bisa memvalidasi jawaban, jadi kode dicek ulang saat respon ditarik.

// hurufKode: huruf A-Z (tanpa I) dan angka 1-9 (tanpa 0).
const hurufKode = "ABCDEFGHJKLMNOPQRSTUVWXYZ123456789"

// panjangKode: 6 karakter (lebih dari 1,5 miliar kombinasi), dibuat otomatis tiap kuesioner baru.
const panjangKode = 6

// ErrKodeDipakai: kode yang baru dibuat ternyata sudah dipakai kuesioner lain (dua kuesioner disimpan bersamaan).
var ErrKodeDipakai = errors.New("kode akses sudah dipakai")

// judulItemKode: pertanyaan pertama (item 0) di Google Form, dibuat otomatis saat sinkron.
const judulItemKode = "Kode kebaktian"

// CekKodeRequest, contoh: {"kode": "K7MP2X"}
type CekKodeRequest struct {
	Kode string `json:"kode" binding:"required,max=8"`
}

// CekKodeResponse: link_form sudah berisi kode, app tinggal membukanya.
type CekKodeResponse struct {
	IDKuesioner    uint       `json:"id_kuesioner"`
	JenisKuesioner string     `json:"jenis_kuesioner"`
	Judul          string     `json:"judul"`
	LinkForm       string     `json:"link_form"`
	DitutupPada    *time.Time `json:"ditutup_pada"`
}

// ================= service =================

// buatKodeAkses membuat kode acak yang belum pernah dipakai kuesioner lain.
func (service *Service) buatKodeAkses(ctx context.Context) (string, error) {
	for percobaan := 0; percobaan < 10; percobaan++ {
		kode := ""
		for i := 0; i < panjangKode; i++ {
			angka, err := rand.Int(rand.Reader, big.NewInt(int64(len(hurufKode))))
			if err != nil {
				return "", err
			}
			kode += string(hurufKode[angka.Int64()])
		}
		dipakai, err := service.repo.KodeAksesDipakai(ctx, kode)
		if err != nil {
			return "", err
		}
		if !dipakai {
			return kode, nil
		}
	}
	return "", errors.New("gagal membuat kode akses yang unik, coba lagi")
}

// simpanDenganKode: buat kode lalu simpan kuesioner. Kalau kode bentrok dengan kuesioner yang
// disimpan di waktu yang sama, buat kode baru dan simpan ulang (maksimal 3 kali).
func (service *Service) simpanDenganKode(ctx context.Context, kuesioner *models.KuesionerEvaluasi) error {
	for percobaan := 0; percobaan < 3; percobaan++ {
		kode, err := service.buatKodeAkses(ctx)
		if err != nil {
			return err
		}
		kuesioner.KodeAkses = kode
		err = service.repo.CreateKuesioner(ctx, kuesioner)
		if !errors.Is(err, ErrKodeDipakai) {
			return err // berhasil (nil), atau error lain seperti ErrDuplicate
		}
	}
	return errors.New("gagal menyimpan kuesioner: kode akses terus bentrok, coba lagi")
}

// kodeBentrok: true kalau database menolak karena UNIQUE kode_akses (uk_kuesioner_kode).
func kodeBentrok(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 &&
		strings.Contains(mysqlErr.Message, "uk_kuesioner_kode")
}

// CekKode: kode benar dan form sedang dibuka -> link Google Form dengan kode sudah terisi.
func (service *Service) CekKode(ctx context.Context, req CekKodeRequest) (*CekKodeResponse, error) {
	kode := strings.ToUpper(strings.TrimSpace(req.Kode))
	kuesioner, err := service.repo.FindKuesionerByKode(ctx, kode)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.Validation(map[string]string{"kode": "kode tidak ditemukan, cek lagi kode yang diumumkan"})
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	switch statusKuesioner(kuesioner, time.Now()) {
	case StatusBelumDibuka:
		return nil, apperror.Validation(map[string]string{"kode": "form evaluasi untuk kode ini belum dibuka"})
	case StatusDitutup:
		return nil, apperror.Validation(map[string]string{"kode": "form evaluasi untuk kode ini sudah ditutup"})
	}

	if service.form == nil {
		return nil, apperror.New(http.StatusServiceUnavailable, "Form evaluasi belum tersedia")
	}
	info, err := service.infoGoogle(ctx, kuesioner.JenisKuesioner)
	if err != nil {
		return nil, err
	}
	if !info.MenerimaJawaban {
		return nil, apperror.New(http.StatusServiceUnavailable, "Form evaluasi sedang tidak menerima jawaban")
	}

	return &CekKodeResponse{
		IDKuesioner:    kuesioner.ID,
		JenisKuesioner: kuesioner.JenisKuesioner,
		Judul:          kuesioner.JudulForm,
		LinkForm:       linkDenganKode(info.LinkForm, service.idPertanyaanKode(kuesioner.JenisKuesioner), kode),
		DitutupPada:    kuesioner.DitutupPada,
	}, nil
}

// idPertanyaanKode: ID pertanyaan "Kode kebaktian" dari cache infoGoogle ("" = belum ada di form).
func (service *Service) idPertanyaanKode(jenis string) string {
	service.cacheForm.kunci.Lock()
	defer service.cacheForm.kunci.Unlock()
	return service.cacheForm.isi[jenis].idPertanyaanKode
}

// linkDenganKode: link form dengan jawaban kode sudah terisi (link prefill).
// ID pertanyaan dari API berupa heksadesimal, link prefill memakai angka desimal: entry.<angka>=<kode>.
func linkDenganKode(link, idPertanyaanKode, kode string) string {
	angka, err := strconv.ParseInt(idPertanyaanKode, 16, 64)
	if err != nil { // item kode belum ada di form (belum sinkron), jemaat mengetik sendiri
		return link
	}
	alamat, err := url.Parse(link)
	if err != nil {
		return link
	}
	query := alamat.Query()
	query.Set("usp", "pp_url")
	query.Set("entry."+strconv.FormatInt(angka, 10), kode)
	alamat.RawQuery = query.Encode()
	return alamat.String()
}

// ================= Google Form =================

// buatItemKode: pertanyaan teks singkat wajib di paling atas form.
func buatItemKode() *forms.Item {
	return &forms.Item{
		Title:       judulItemKode,
		Description: "Kode diumumkan saat ibadah. Kalau form dibuka dari aplikasi, kode sudah terisi.",
		QuestionItem: &forms.QuestionItem{
			Question: &forms.Question{Required: true, TextQuestion: &forms.TextQuestion{}},
		},
	}
}

// cariItemKode: posisi item kode di form, -1 kalau belum ada.
func cariItemKode(daftarItem []*forms.Item) int {
	for i, item := range daftarItem {
		if item.Title == judulItemKode && tipeDariGoogle(item) == models.TipeTeksSingkat {
			return i
		}
	}
	return -1
}

// pastikanItemKode: item kode harus ada di posisi 0, dibuat atau dipindah kalau belum.
func (service *Service) pastikanItemKode(ctx context.Context, jenis string, form *forms.Form) (*forms.Form, error) {
	posisiKode := cariItemKode(form.Items)
	if posisiKode == 0 {
		return form, nil
	}
	perubahan := perintahPindah(posisiKode, 0)
	if posisiKode == -1 {
		perubahan = &forms.Request{CreateItem: &forms.CreateItemRequest{Item: buatItemKode(), Location: posisi(0)}}
	}
	return service.form.Ubah(ctx, jenis, []*forms.Request{perubahan})
}

// idPertanyaanKodeDariForm: dipakai infoGoogle untuk mengisi cache.
func idPertanyaanKodeDariForm(form *forms.Form) string {
	posisiKode := cariItemKode(form.Items)
	if posisiKode == -1 {
		return ""
	}
	return form.Items[posisiKode].QuestionItem.Question.QuestionId
}

// ================= handler =================

// CekKode godoc
//
//	@Summary		Cek kode akses form evaluasi (publik)
//	@Description	Kode benar dan form sedang dibuka -> link Google Form dengan kode sudah terisi.
//	@Tags			kuesioner
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CekKodeRequest	true	"Kode dari pengumuman ibadah"
//	@Success		200		{object}	response.Body
//	@Failure		422		{object}	response.Body
//	@Failure		503		{object}	response.Body
//	@Router			/evaluasi/kode [post]
func (handler *Handler) CekKode(c *gin.Context) {
	var req CekKodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	hasil, err := handler.service.CekKode(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Kode benar, silakan isi form evaluasi")
}

// ================= repository =================

// KodeAksesDipakai: kuesioner yang sudah dihapus ikut dicek karena kodenya masih tersimpan (UNIQUE).
func (repo *gormRepository) KodeAksesDipakai(ctx context.Context, kode string) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Unscoped().Model(&models.KuesionerEvaluasi{}).
		Where("kode_akses = ?", kode).Count(&jumlah).Error
	return jumlah > 0, err
}

// FindKuesionerByKode: huruf kecil/besar dianggap sama (collation database).
func (repo *gormRepository) FindKuesionerByKode(ctx context.Context, kode string) (*models.KuesionerEvaluasi, error) {
	var kuesioner models.KuesionerEvaluasi
	err := repo.db.WithContext(ctx).Where("kode_akses = ?", kode).First(&kuesioner).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &kuesioner, nil
}
