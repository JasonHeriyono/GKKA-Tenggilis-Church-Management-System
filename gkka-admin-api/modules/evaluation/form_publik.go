package evaluation

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/forms/v1"

	"sim-g-backend/helpers/apperror"
)

// lamaCacheForm: isi form jarang berubah, cache supaya tiap jemaat yang membuka app
// tidak memanggil Google API (kuota Forms API terbatas per menit).
const lamaCacheForm = 5 * time.Minute

// FormPublikResponse: info form evaluasi untuk app jemaat / tamu.
// Di luar jadwal link_form dikosongkan, dibuka_pada/ditutup_pada berisi jadwal berikutnya (null kalau belum ada).
type FormPublikResponse struct {
	JenisKuesioner  string     `json:"jenis_kuesioner"`
	Judul           string     `json:"judul"`
	LinkForm        string     `json:"link_form"`
	MenerimaJawaban bool       `json:"menerima_jawaban"` // true = sedang dalam jadwal dan form Google terbuka
	DibukaPada      *time.Time `json:"dibuka_pada"`
	DitutupPada     *time.Time `json:"ditutup_pada"`
}

type cacheForm struct {
	isi        FormPublikResponse
	kedaluarsa time.Time
	// header yang sekarang ada di Google Form, dipakai pasangHeader (questionnaire_header.go)
	judulGoogle, deskripsiGoogle string
	// ID pertanyaan "Kode kebaktian", dipakai untuk link prefill (kode_akses.go)
	idPertanyaanKode string
}

// cacheFormPublik: hasil FormPublik per jenis kuesioner.
type cacheFormPublik struct {
	kunci sync.Mutex
	isi   map[string]cacheForm
}

// FormPublik mengembalikan link Google Form yang dipakai jemaat untuk mengisi evaluasi,
// hanya kalau sekarang masuk jadwal kuesioner yang diatur admin.
func (service *Service) FormPublik(ctx context.Context, jenis string) (*FormPublikResponse, error) {
	jenis = strings.ToUpper(strings.TrimSpace(jenis))
	if !jenisValid(jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	if service.form == nil {
		return nil, apperror.New(http.StatusServiceUnavailable, "Form evaluasi belum tersedia")
	}

	hasil, err := service.infoGoogle(ctx, jenis)
	if err != nil {
		return nil, err
	}

	sekarang := time.Now()
	dibuka, err := service.repo.KuesionerDibuka(ctx, jenis, sekarang)
	if err == nil {
		// header gagal diganti jangan sampai jemaat tidak bisa mengisi, cukup dicatat
		if err := service.pasangHeader(ctx, jenis, dibuka); err != nil {
			slog.WarnContext(ctx, "header Google Form gagal diganti", "jenis", jenis, "id_kuesioner", dibuka.ID, "error", err)
		}
		hasil.Judul = dibuka.JudulForm
		hasil.DibukaPada, hasil.DitutupPada = dibuka.DibukaPada, dibuka.DitutupPada
		return hasil, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}

	// di luar jadwal: link tidak dibagikan, beri tahu kapan dibuka lagi
	hasil.LinkForm = ""
	hasil.MenerimaJawaban = false
	berikutnya, err := service.repo.KuesionerBerikutnya(ctx, jenis, sekarang)
	if err == nil {
		hasil.Judul = berikutnya.JudulForm
		hasil.DibukaPada, hasil.DitutupPada = berikutnya.DibukaPada, berikutnya.DitutupPada
	} else if !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}
	return hasil, nil
}

// infoGoogle: link dan status form dari Google, di-cache per jenis.
func (service *Service) infoGoogle(ctx context.Context, jenis string) (*FormPublikResponse, error) {
	// dikunci sampai selesai supaya request barengan saat cache habis cukup sekali ke Google
	service.cacheForm.kunci.Lock()
	defer service.cacheForm.kunci.Unlock()

	sekarang := time.Now()
	if simpanan, ada := service.cacheForm.isi[jenis]; ada && sekarang.Before(simpanan.kedaluarsa) {
		hasil := simpanan.isi
		return &hasil, nil
	}

	form, err := service.form.Ambil(ctx, jenis)
	if err != nil {
		return nil, googleError(err)
	}
	hasil := toFormPublik(jenis, form)
	if service.cacheForm.isi == nil {
		service.cacheForm.isi = map[string]cacheForm{}
	}
	simpanan := cacheForm{isi: hasil, kedaluarsa: sekarang.Add(lamaCacheForm), idPertanyaanKode: idPertanyaanKodeDariForm(form)}
	if form.Info != nil {
		simpanan.judulGoogle, simpanan.deskripsiGoogle = form.Info.Title, form.Info.Description
	}
	service.cacheForm.isi[jenis] = simpanan
	return &hasil, nil
}

func toFormPublik(jenis string, form *forms.Form) FormPublikResponse {
	hasil := FormPublikResponse{
		JenisKuesioner: jenis,
		LinkForm:       form.ResponderUri,
		// form lama (dibuat sebelum ada fitur publish) tidak punya publishSettings, selalu terbuka
		MenerimaJawaban: true,
	}
	if form.Info != nil {
		hasil.Judul = form.Info.Title
	}
	if form.PublishSettings != nil {
		state := form.PublishSettings.PublishState
		hasil.MenerimaJawaban = state != nil && state.IsPublished && state.IsAcceptingResponses
	}
	return hasil
}
