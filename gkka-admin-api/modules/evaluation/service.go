package evaluation

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

// Service berisi aturan bisnis pertanyaan evaluasi.
// Dipanggil oleh handler.go, lalu memanggil repository.go untuk baca/tulis database.
type Service struct {
	repo         Repository
	form         GoogleForm      // nil = Google Form belum dihubungkan
	kunciSinkron sync.Mutex      // dipakai di SinkronGoogle (sinkron_google.go)
	cacheForm    cacheFormPublik // dipakai di FormPublik (form_publik.go)
	loc          *time.Location  // zona waktu aplikasi (APP_TIMEZONE), untuk membaca jam dari admin
}

// NewService: loc adalah zona waktu aplikasi (WIB).
func NewService(repo Repository, form GoogleForm, loc *time.Location) *Service {
	if loc == nil {
		loc = time.UTC
	}
	return &Service{repo: repo, form: form, loc: loc}
}

// ListPertanyaan ambil daftar pertanyaan. Filter jenis dan aktif boleh kosong (= ambil semua).
func (service *Service) ListPertanyaan(ctx context.Context, filter PertanyaanQuery) ([]PertanyaanResponse, error) {
	filter.Jenis = strings.ToUpper(strings.TrimSpace(filter.Jenis)) // "mingguan " -> "MINGGUAN"
	if filter.Jenis != "" && !jenisValid(filter.Jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}

	daftarPertanyaan, err := service.repo.ListPertanyaan(ctx, filter)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	// ubah setiap data database jadi bentuk JSON untuk client
	hasil := make([]PertanyaanResponse, 0, len(daftarPertanyaan))
	for i := range daftarPertanyaan {
		hasil = append(hasil, toPertanyaanResponse(&daftarPertanyaan[i]))
	}
	return hasil, nil
}

// GetPertanyaan ambil satu pertanyaan beserta pilihan jawabannya.
func (service *Service) GetPertanyaan(ctx context.Context, pertanyaanID uint) (*PertanyaanResponse, error) {
	pertanyaan, err := service.repo.FindPertanyaan(ctx, pertanyaanID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	hasil := toPertanyaanResponse(pertanyaan)
	return &hasil, nil
}

// CreatePertanyaan cek data -> susun pertanyaan + opsinya -> simpan. Pertanyaan baru langsung aktif.
func (service *Service) CreatePertanyaan(ctx context.Context, req PertanyaanRequest) (*PertanyaanResponse, error) {
	labelOpsi, err := validasiPertanyaan(req)
	if err != nil {
		return nil, err
	}

	pertanyaan := &models.PertanyaanEvaluasi{
		JenisKuesioner: req.JenisKuesioner,
		Teks:           strings.TrimSpace(req.Teks),
		TipeJawaban:    req.TipeJawaban,
		Urutan:         req.Urutan,
		Wajib:          req.Wajib,
		IsAktif:        true,
	}
	for i, label := range labelOpsi {
		opsi := models.OpsiPertanyaan{Label: label, Urutan: i + 1} // urutan opsi mulai dari 1
		pertanyaan.Opsi = append(pertanyaan.Opsi, opsi)
	}

	if err := service.repo.CreatePertanyaan(ctx, pertanyaan); err != nil {
		return nil, duplicateOrInternal(err)
	}
	hasil := toPertanyaanResponse(pertanyaan)
	return &hasil, nil
}

// UpdatePertanyaan cek data -> pastikan pertanyaannya ada -> timpa isinya -> simpan.
// Opsi disamakan dengan yang dikirim (caranya ada di syncOpsi, repository.go).
func (service *Service) UpdatePertanyaan(ctx context.Context, pertanyaanID uint, req PertanyaanRequest) (*PertanyaanResponse, error) {
	labelOpsi, err := validasiPertanyaan(req)
	if err != nil {
		return nil, err
	}

	pertanyaan, err := service.repo.FindPertanyaan(ctx, pertanyaanID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	pertanyaan.JenisKuesioner = req.JenisKuesioner
	pertanyaan.Teks = strings.TrimSpace(req.Teks)
	pertanyaan.TipeJawaban = req.TipeJawaban
	pertanyaan.Urutan = req.Urutan
	pertanyaan.Wajib = req.Wajib

	if err := service.repo.UpdatePertanyaan(ctx, pertanyaan, labelOpsi); err != nil {
		return nil, duplicateOrInternal(err)
	}
	return service.GetPertanyaan(ctx, pertanyaanID) // ambil ulang supaya opsi terbaru ikut terkirim
}

// SetAktifPertanyaan aktifkan / nonaktifkan pertanyaan. Pertanyaan nonaktif tidak dihapus, hanya tidak dipakai di form.
func (service *Service) SetAktifPertanyaan(ctx context.Context, pertanyaanID uint, isAktif bool) (*PertanyaanResponse, error) {
	// cek dulu pertanyaannya ada, supaya ID salah dapat 404
	if _, err := service.repo.FindPertanyaan(ctx, pertanyaanID); err != nil {
		return nil, notFoundOrInternal(err)
	}
	if err := service.repo.SetAktifPertanyaan(ctx, pertanyaanID, isAktif); err != nil {
		return nil, apperror.Internal(err)
	}
	return service.GetPertanyaan(ctx, pertanyaanID)
}

// DeletePertanyaan hapus pertanyaan. Datanya tidak benar-benar hilang (deleted_at diisi), jadi jawaban lama tetap aman.
func (service *Service) DeletePertanyaan(ctx context.Context, pertanyaanID uint) error {
	if err := service.repo.DeletePertanyaan(ctx, pertanyaanID); err != nil {
		return notFoundOrInternal(err)
	}
	return nil
}

// PreviewImportPertanyaan baca Excel lalu tunjukkan apa yang akan terjadi. Belum ada data yang disimpan.
func (service *Service) PreviewImportPertanyaan(ctx context.Context, file io.Reader) (*ImportPreviewResponse, error) {
	rencana, err := service.siapkanImport(ctx, file)
	if err != nil {
		return nil, err
	}
	return &rencana.preview, nil
}

// ImportPertanyaan simpan isi Excel sebagai set pertanyaan (ganti set):
// pertanyaan yang sama dipakai ulang, yang baru dibuat, yang tidak ada di Excel dinonaktifkan.
// File dicek ulang di sini, jadi hasilnya sama dengan preview.
func (service *Service) ImportPertanyaan(ctx context.Context, file io.Reader) (*ImportSimpanResponse, error) {
	rencana, err := service.siapkanImport(ctx, file)
	if err != nil {
		return nil, err
	}
	if !rencana.preview.BisaDisimpan {
		return nil, apperror.Validation(map[string]string{"file": "masih ada baris yang error atau file kosong, cek dulu lewat preview"})
	}

	err = service.repo.ImportPertanyaan(ctx, rencana.pertanyaanBaru, rencana.pertanyaanDipakaiUlang, rencana.idDinonaktifkan)
	if err != nil {
		return nil, duplicateOrInternal(err)
	}
	return &ImportSimpanResponse{
		Ditambah:      len(rencana.pertanyaanBaru),
		Diperbarui:    len(rencana.pertanyaanDipakaiUlang),
		Dinonaktifkan: len(rencana.idDinonaktifkan),
	}, nil
}

// rencanaImport: hasil siapkanImport, dipakai oleh preview maupun simpan.
type rencanaImport struct {
	preview                ImportPreviewResponse
	pertanyaanBaru         []models.PertanyaanEvaluasi
	pertanyaanDipakaiUlang []models.PertanyaanEvaluasi // sudah berisi urutan, wajib, dan opsi dari Excel
	idDinonaktifkan        []uint
}

// siapkanImport baca + cek Excel, lalu cocokkan dengan pertanyaan aktif di database.
// Pertanyaan dianggap SAMA kalau jenis, teks (huruf besar/kecil diabaikan), dan tipe jawabannya sama.
func (service *Service) siapkanImport(ctx context.Context, file io.Reader) (*rencanaImport, error) {
	rows, err := bacaSheetPertanyaan(file)
	if err != nil {
		if errors.Is(err, ErrSheetTidakAda) {
			return nil, apperror.BadRequest("Sheet \"Pertanyaan\" tidak ditemukan, gunakan template pertanyaan dari sistem")
		}
		appErr := apperror.BadRequest("File harus berupa Excel (.xlsx) yang valid")
		appErr.Err = err
		return nil, appErr
	}

	daftarBaris, kolomHilang, err := parseBarisPertanyaan(rows)
	if err != nil {
		return nil, apperror.BadRequest(err.Error()) // contoh: lebih dari 400 pertanyaan
	}
	if len(kolomHilang) > 0 {
		errorPerKolom := map[string]string{}
		for _, nama := range kolomHilang {
			errorPerKolom[nama] = "kolom wajib tidak ditemukan di baris header"
		}
		return nil, apperror.Validation(errorPerKolom)
	}

	// ambil pertanyaan yang sedang aktif, lalu kelompokkan per kunci jenis|teks|tipe
	aktif := true
	pertanyaanAktif, err := service.repo.ListPertanyaan(ctx, PertanyaanQuery{Aktif: &aktif})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	pertanyaanLama := map[string]*models.PertanyaanEvaluasi{}
	for i := range pertanyaanAktif {
		lama := &pertanyaanAktif[i]
		pertanyaanLama[kunciPertanyaan(lama.JenisKuesioner, lama.Teks, lama.TipeJawaban)] = lama
	}

	rencana := &rencanaImport{}
	rencana.preview.Baris = daftarBaris
	rencana.preview.AkanDinonaktifkan = []PertanyaanResponse{} // array kosong, bukan null
	jenisDiFile := map[string]bool{}
	dipakaiUlang := map[uint]bool{}

	for i := range rencana.preview.Baris {
		baris := &rencana.preview.Baris[i]
		if baris.Errors != nil {
			rencana.preview.BarisError++
			continue
		}
		rencana.preview.BarisValid++
		jenisDiFile[baris.JenisKuesioner] = true

		lama, ada := pertanyaanLama[kunciPertanyaan(baris.JenisKuesioner, baris.Teks, baris.TipeJawaban)]
		if ada && !dipakaiUlang[lama.ID] {
			// pertanyaan lama dipakai ulang: ID (dan ID Google-nya) tetap, isinya ikut Excel
			baris.Status = StatusImportSama
			dipakaiUlang[lama.ID] = true
			lama.Teks = baris.Teks // huruf besar/kecil ikut versi Excel
			lama.Urutan = baris.Urutan
			lama.Wajib = baris.Wajib
			lama.Opsi = buatOpsi(baris.Opsi)
			rencana.pertanyaanDipakaiUlang = append(rencana.pertanyaanDipakaiUlang, *lama)
			continue
		}

		baris.Status = StatusImportBaru
		rencana.pertanyaanBaru = append(rencana.pertanyaanBaru, models.PertanyaanEvaluasi{
			JenisKuesioner: baris.JenisKuesioner,
			Teks:           baris.Teks,
			TipeJawaban:    baris.TipeJawaban,
			Urutan:         baris.Urutan,
			Wajib:          baris.Wajib,
			IsAktif:        true,
			Opsi:           buatOpsi(baris.Opsi),
		})
	}

	// pertanyaan aktif yang jenisnya ada di Excel tapi tidak dipakai ulang -> dinonaktifkan (tidak dihapus)
	for i := range pertanyaanAktif {
		lama := &pertanyaanAktif[i]
		if jenisDiFile[lama.JenisKuesioner] && !dipakaiUlang[lama.ID] {
			rencana.idDinonaktifkan = append(rencana.idDinonaktifkan, lama.ID)
			rencana.preview.AkanDinonaktifkan = append(rencana.preview.AkanDinonaktifkan, toPertanyaanResponse(lama))
		}
	}

	rencana.preview.TotalBaris = len(daftarBaris)
	rencana.preview.BisaDisimpan = rencana.preview.TotalBaris > 0 && rencana.preview.BarisError == 0
	return rencana, nil
}

// kunciPertanyaan: kunci untuk mencari pertanyaan yang sama, contoh "MINGGUAN|apakah ...|RATING".
func kunciPertanyaan(jenis, teks, tipeJawaban string) string {
	return jenis + "|" + strings.ToLower(strings.TrimSpace(teks)) + "|" + tipeJawaban
}

// buatOpsi ubah daftar label jadi data opsi; urutan opsi mulai dari 1.
func buatOpsi(daftarLabel []string) []models.OpsiPertanyaan {
	daftarOpsi := []models.OpsiPertanyaan{}
	for i, label := range daftarLabel {
		daftarOpsi = append(daftarOpsi, models.OpsiPertanyaan{Label: label, Urutan: i + 1})
	}
	return daftarOpsi
}

// validasiPertanyaan cek aturan tambahan yang tidak bisa ditulis lewat tag binding di dto.go.
// Kalau ada yang salah -> error 422 berisi pesan per field. Kalau lolos -> label opsi yang sudah dirapikan.
func validasiPertanyaan(req PertanyaanRequest) ([]string, error) {
	errorPerField := map[string]string{}

	if strings.TrimSpace(req.Teks) == "" {
		errorPerField["teks_pertanyaan"] = "wajib diisi"
	}

	labelOpsi, pesanOpsi := cekOpsi(req.TipeJawaban, req.Opsi)
	if pesanOpsi != "" {
		errorPerField["opsi"] = pesanOpsi
	}

	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}
	return labelOpsi, nil
}

// cekOpsi rapikan label opsi lalu cek aturannya. Hasil pesan kosong ("") artinya opsi lolos.
// Aturan: tipe pilihan wajib punya minimal 1 opsi, tipe lain tidak boleh punya opsi, label tidak boleh kosong/dobel.
func cekOpsi(tipeJawaban string, daftarLabel []string) ([]string, string) {
	// tipe RATING / TEKS_SINGKAT / PARAGRAF tidak boleh dikirim dengan opsi
	if !tipePunyaOpsi(tipeJawaban) {
		if len(daftarLabel) > 0 {
			return nil, "tipe " + tipeJawaban + " tidak memakai opsi"
		}
		return nil, ""
	}

	labelRapi := make([]string, 0, len(daftarLabel))
	sudahAda := map[string]bool{}
	for _, label := range daftarLabel {
		label = strings.TrimSpace(label) // buang spasi di depan/belakang
		if label == "" {
			return nil, "label opsi tidak boleh kosong"
		}
		kunci := strings.ToLower(label) // "Baik" dan "baik" dianggap sama, seperti di database
		if sudahAda[kunci] {
			return nil, "label opsi \"" + label + "\" dobel"
		}
		sudahAda[kunci] = true
		labelRapi = append(labelRapi, label)
	}

	if len(labelRapi) == 0 {
		return nil, "minimal 1 opsi untuk tipe " + tipeJawaban
	}
	return labelRapi, ""
}

// tipePunyaOpsi: hanya pilihan ganda, centang, dan dropdown yang punya pilihan jawaban.
func tipePunyaOpsi(tipeJawaban string) bool {
	return tipeJawaban == models.TipePilihanGanda ||
		tipeJawaban == models.TipeCentang ||
		tipeJawaban == models.TipeDropdown
}

// jenisValid: jenis kuesioner hanya MINGGUAN atau TAHUNAN.
func jenisValid(jenis string) bool {
	return jenis == models.KuesionerMingguan || jenis == models.KuesionerTahunan
}

// notFoundOrInternal: pertanyaan tidak ditemukan -> 404, error lain -> 500.
func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("Pertanyaan tidak ditemukan")
	}
	return apperror.Internal(err)
}

// duplicateOrInternal: label opsi dobel di database -> 409, error lain -> 500.
func duplicateOrInternal(err error) error {
	if errors.Is(err, ErrDuplicate) {
		return apperror.Conflict("Label opsi sudah dipakai di pertanyaan ini")
	}
	return apperror.Internal(err)
}

// toPertanyaanResponse ubah data dari database (models) jadi bentuk JSON untuk client (dto).
func toPertanyaanResponse(pertanyaan *models.PertanyaanEvaluasi) PertanyaanResponse {
	hasil := PertanyaanResponse{
		ID:             pertanyaan.ID,
		JenisKuesioner: pertanyaan.JenisKuesioner,
		Teks:           pertanyaan.Teks,
		TipeJawaban:    pertanyaan.TipeJawaban,
		Urutan:         pertanyaan.Urutan,
		Wajib:          pertanyaan.Wajib,
		IsAktif:        pertanyaan.IsAktif,
		Opsi:           []OpsiResponse{}, // array kosong, bukan null, kalau tidak ada opsi
	}
	for _, opsi := range pertanyaan.Opsi {
		hasil.Opsi = append(hasil.Opsi, OpsiResponse{ID: opsi.ID, Label: opsi.Label, Urutan: opsi.Urutan})
	}
	return hasil
}

// UpdateHeader menyimpan template header, lalu langsung memasangnya ke kuesioner yang sedang dibuka.
func (service *Service) UpdateHeader(ctx context.Context, req UpdateHeaderRequest) error {
	jenis := strings.ToUpper(strings.TrimSpace(req.Jenis))
	if !jenisValid(jenis) {
		return apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	judul := strings.TrimSpace(req.Judul)
	if judul == "" {
		return apperror.BadRequest("Judul form wajib diisi")
	}

	kunciJudul, kunciDeskripsi := kunciHeader(jenis)
	err := service.repo.SimpanPengaturan(ctx, map[string]string{
		kunciJudul:     judul,
		kunciDeskripsi: strings.TrimSpace(req.Deskripsi),
	})
	if err != nil {
		return apperror.Internal(err)
	}

	// belum ada kuesioner yang dibuka: template dipakai nanti saat kuesioner dibuka
	if service.form == nil {
		return nil
	}
	kuesioner, err := service.repo.KuesionerDibuka(ctx, jenis, time.Now())
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return apperror.Internal(err)
	}
	if err := service.pasangHeader(ctx, jenis, kuesioner); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// AmbilHeader: template yang tersimpan untuk web admin (judul kosong = belum diatur).
func (service *Service) AmbilHeader(ctx context.Context, jenis string) (*UpdateHeaderRequest, error) {
	jenis = strings.ToUpper(strings.TrimSpace(jenis))
	if !jenisValid(jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	template, err := service.ambilTemplateHeader(ctx, jenis)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := &UpdateHeaderRequest{Jenis: jenis}
	if template != nil {
		hasil.Judul, hasil.Deskripsi = template.Judul, template.Deskripsi
	}
	return hasil, nil
}
