package evaluation

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/forms/v1"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/response"
	"sim-g-backend/models"
)

// Tarik respon: kiriman Google Form disalin ke respon_evaluasi + jawaban_evaluasi.
// Setiap respon dicek lewat jawaban "Kode kebaktian":
//   - kode tidak cocok dengan kuesioner jenis ini -> INVALID (KODE_SALAH)
//   - dikirim sebelum dibuka / sesudah ditutup     -> INVALID (BELUM_DIBUKA / LEWAT_WAKTU)
//   - selain itu                                   -> VALID, masuk ke kuesioner sesuai kode
// Respon yang sudah pernah ditarik dilewati (google_response_id unik), jadi aman dipanggil berulang.

// TarikResponResponse: ringkasan hasil POST /kuesioner/tarik-respon.
type TarikResponResponse struct {
	JenisKuesioner string `json:"jenis_kuesioner"`
	TotalDiGoogle  int    `json:"total_di_google"`
	Baru           int    `json:"baru"`    // baru disimpan kali ini
	Valid          int    `json:"valid"`   // dari yang baru
	Invalid        int    `json:"invalid"` // dari yang baru
}

// ================= service =================

func (service *Service) TarikRespon(ctx context.Context, jenis string) (*TarikResponResponse, error) {
	jenis = strings.ToUpper(strings.TrimSpace(jenis))
	if !jenisValid(jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	if service.form == nil {
		return nil, apperror.New(http.StatusServiceUnavailable, "Google Form belum dihubungkan, isi GOOGLE_FORM_ID_* di .env")
	}
	// satu tarikan per waktu supaya respon yang sama tidak diproses dua kali bersamaan
	service.kunciSinkron.Lock()
	defer service.kunciSinkron.Unlock()

	form, err := service.form.Ambil(ctx, jenis)
	if err != nil {
		return nil, googleError(err)
	}
	daftarRespon, err := service.form.AmbilRespon(ctx, jenis)
	if err != nil {
		return nil, googleError(err)
	}
	hasil := &TarikResponResponse{JenisKuesioner: jenis, TotalDiGoogle: len(daftarRespon)}

	// pertanyaan dicari lewat ID Google-nya (termasuk yang sudah nonaktif, untuk respon lama)
	semuaPertanyaan, err := service.repo.ListPertanyaan(ctx, PertanyaanQuery{Jenis: jenis})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	pertanyaanByGoogleID := map[string]*models.PertanyaanEvaluasi{}
	for i := range semuaPertanyaan {
		if id := semuaPertanyaan[i].GoogleQuestionID; id != nil {
			pertanyaanByGoogleID[*id] = &semuaPertanyaan[i]
		}
	}

	var daftarID []string
	for _, respon := range daftarRespon {
		daftarID = append(daftarID, respon.ResponseId)
	}
	sudahTersimpan, err := service.repo.ResponTersimpan(ctx, daftarID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	cari := &pencariKuesioner{service: service, jenis: jenis, simpanan: map[string]*models.KuesionerEvaluasi{}}
	idPertanyaanKode := idPertanyaanKodeDariForm(form)
	for _, responGoogle := range daftarRespon {
		if sudahTersimpan[responGoogle.ResponseId] {
			continue
		}
		respon, err := service.susunRespon(ctx, responGoogle, idPertanyaanKode, pertanyaanByGoogleID, cari)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		err = service.repo.SimpanRespon(ctx, respon)
		if errors.Is(err, ErrDuplicate) {
			continue // sudah disimpan proses lain
		}
		if err != nil {
			return nil, apperror.Internal(err)
		}
		hasil.Baru++
		if respon.StatusRespon == models.ResponValid {
			hasil.Valid++
		} else {
			hasil.Invalid++
		}
	}
	return hasil, nil
}

// susunRespon mengubah satu kiriman Google jadi baris respon + jawabannya.
func (service *Service) susunRespon(ctx context.Context, responGoogle *forms.FormResponse, idPertanyaanKode string,
	pertanyaanByGoogleID map[string]*models.PertanyaanEvaluasi, cari *pencariKuesioner) (*models.ResponEvaluasi, error) {

	waktuSubmit := service.waktuDariGoogle(responGoogle)
	respon := &models.ResponEvaluasi{
		StatusRespon:     models.ResponValid,
		WaktuSubmit:      waktuSubmit,
		GoogleResponseID: responGoogle.ResponseId,
	}

	// 1. cek kode
	kode := strings.ToUpper(strings.TrimSpace(jawabanPertama(responGoogle, idPertanyaanKode)))
	if kode != "" {
		respon.KodeDimasukkan = &kode
	}
	kuesioner, err := cari.dariKode(ctx, kode)
	if err != nil {
		return nil, err
	}
	alasan := ""
	switch {
	case kuesioner == nil:
		alasan = models.AlasanKodeSalah
	case kuesioner.DibukaPada != nil && waktuSubmit.Before(*kuesioner.DibukaPada):
		alasan = models.AlasanBelumDibuka
	case kuesioner.DitutupPada != nil && !waktuSubmit.Before(*kuesioner.DitutupPada):
		alasan = models.AlasanLewatWaktu
	}
	if kuesioner != nil {
		respon.KuesionerID = &kuesioner.ID
	}
	if alasan != "" {
		respon.StatusRespon = models.ResponInvalid
		respon.AlasanInvalid = &alasan
	}

	// 2. salin jawaban tiap pertanyaan (respon INVALID juga disimpan, tapi tidak dihitung di laporan)
	for idGoogle, jawabanGoogle := range responGoogle.Answers {
		pertanyaan := pertanyaanByGoogleID[idGoogle]
		if pertanyaan == nil || jawabanGoogle.TextAnswers == nil {
			continue // item kode, atau pertanyaan yang sudah tidak ada di database
		}
		for _, teksJawaban := range jawabanGoogle.TextAnswers.Answers {
			respon.Jawaban = append(respon.Jawaban, susunJawaban(pertanyaan, teksJawaban.Value))
		}
	}
	return respon, nil
}

// susunJawaban: RATING -> angka, tipe pilihan -> id opsi, teks -> teks.
func susunJawaban(pertanyaan *models.PertanyaanEvaluasi, nilai string) models.JawabanEvaluasi {
	jawaban := models.JawabanEvaluasi{PertanyaanID: pertanyaan.ID}
	switch {
	case pertanyaan.TipeJawaban == models.TipeRating:
		if angka, err := strconv.Atoi(nilai); err == nil {
			jawaban.NilaiRating = &angka
			return jawaban
		}
	case tipePunyaOpsi(pertanyaan.TipeJawaban):
		for _, opsi := range pertanyaan.Opsi {
			if strings.EqualFold(opsi.Label, nilai) {
				idOpsi := opsi.ID
				jawaban.OpsiID = &idOpsi
				return jawaban
			}
		}
	}
	// teks biasa, atau label/angka yang tidak dikenali tetap disimpan sebagai teks
	jawaban.NilaiTeks = &nilai
	return jawaban
}

// jawabanPertama: isi jawaban untuk satu pertanyaan Google ("" kalau tidak dijawab).
func jawabanPertama(responGoogle *forms.FormResponse, idPertanyaan string) string {
	jawaban, ada := responGoogle.Answers[idPertanyaan]
	if !ada || jawaban.TextAnswers == nil || len(jawaban.TextAnswers.Answers) == 0 {
		return ""
	}
	return jawaban.TextAnswers.Answers[0].Value
}

// waktuDariGoogle: Google mengirim waktu UTC format RFC3339, diubah ke zona waktu aplikasi.
func (service *Service) waktuDariGoogle(responGoogle *forms.FormResponse) time.Time {
	teks := responGoogle.LastSubmittedTime
	if teks == "" {
		teks = responGoogle.CreateTime
	}
	waktu, err := time.Parse(time.RFC3339Nano, teks)
	if err != nil {
		return time.Now().In(service.loc)
	}
	return waktu.In(service.loc).Truncate(time.Second)
}

// pencariKuesioner: kode yang sama dicari ke database sekali saja.
type pencariKuesioner struct {
	service  *Service
	jenis    string
	simpanan map[string]*models.KuesionerEvaluasi // nil = kode tidak cocok
}

func (cari *pencariKuesioner) dariKode(ctx context.Context, kode string) (*models.KuesionerEvaluasi, error) {
	if kode == "" {
		return nil, nil
	}
	if kuesioner, sudahDicari := cari.simpanan[kode]; sudahDicari {
		return kuesioner, nil
	}
	kuesioner, err := cari.service.repo.FindKuesionerByKode(ctx, kode)
	if errors.Is(err, ErrNotFound) {
		kuesioner, err = nil, nil
	}
	if err != nil {
		return nil, err
	}
	if kuesioner != nil && kuesioner.JenisKuesioner != cari.jenis { // kode tahunan diketik di form mingguan
		kuesioner = nil
	}
	cari.simpanan[kode] = kuesioner
	return kuesioner, nil
}

// ================= handler =================

// TarikRespon godoc
//
//	@Summary		Tarik respon Google Form ke database (Super Admin)
//	@Description	Respon dicek lewat kode: kode salah / di luar jadwal = INVALID (tetap disimpan, tidak dihitung di laporan). Aman dipanggil berulang.
//	@Tags			kuesioner
//	@Produce		json
//	@Security		BearerAuth
//	@Param			jenis	query		string	true	"MINGGUAN | TAHUNAN"
//	@Success		200		{object}	response.Body
//	@Failure		400		{object}	response.Body
//	@Failure		502		{object}	response.Body
//	@Failure		503		{object}	response.Body
//	@Router			/kuesioner/tarik-respon [post]
func (handler *Handler) TarikRespon(c *gin.Context) {
	hasil, err := handler.service.TarikRespon(c.Request.Context(), c.Query("jenis"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, hasil, "Respon evaluasi berhasil ditarik")
}

// ================= repository =================

// ResponTersimpan: dari daftar ID respon Google, mana saja yang sudah ada di database.
func (repo *gormRepository) ResponTersimpan(ctx context.Context, daftarID []string) (map[string]bool, error) {
	hasil := map[string]bool{}
	if len(daftarID) == 0 {
		return hasil, nil
	}
	var tersimpan []string
	err := repo.db.WithContext(ctx).Unscoped().Model(&models.ResponEvaluasi{}).
		Where("google_response_id IN ?", daftarID).Pluck("google_response_id", &tersimpan).Error
	for _, id := range tersimpan {
		hasil[id] = true
	}
	return hasil, err
}

// SimpanRespon: respon dan jawabannya disimpan bersamaan (gagal satu = batal semua).
func (repo *gormRepository) SimpanRespon(ctx context.Context, respon *models.ResponEvaluasi) error {
	return mapErr(repo.db.WithContext(ctx).Create(respon).Error)
}
