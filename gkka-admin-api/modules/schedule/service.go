package schedule

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
)

// PembuatKuesioner: modul evaluasi, dipakai untuk membuat kuesioner otomatis setelah import Excel.
type PembuatKuesioner interface {
	BuatKuesionerOtomatis(ctx context.Context, daftarKegiatanID []uint) (int, error)
}

// Service berisi logika bisnis jadwal dan kegiatan. Tidak tahu soal HTTP.
type Service struct {
	repo      Repository
	loc       *time.Location
	kuesioner PembuatKuesioner // nil = kuesioner tidak dibuat otomatis
}

// NewService: loc adalah zona waktu aplikasi (WIB) untuk membaca tanggal dan jam.
func NewService(repo Repository, loc *time.Location, kuesioner PembuatKuesioner) *Service {
	if loc == nil {
		loc = time.UTC
	}
	return &Service{repo: repo, loc: loc, kuesioner: kuesioner}
}

// PreviewImport membaca template Excel, memvalidasi per baris, dan mencari bentrok waktu
// di dalam file. Tidak menyimpan apa pun ke database.
//
// Belum dicakup: bentrok dengan jadwal yang sudah ada di DB dan bentrok skill
// (butuh models dan repository setelah ERD final).
func (s *Service) PreviewImport(r io.Reader) (*PreviewResponse, error) {
	rows, err := readSheet(r)
	if err != nil {
		if errors.Is(err, ErrNoSheet) {
			return nil, apperror.BadRequest("File Excel tidak memiliki sheet Kegiatan")
		}
		e := apperror.BadRequest("File harus berupa Excel (.xlsx) yang valid")
		e.Err = err
		return nil, e
	}
	return s.previewRows(rows)
}

func (s *Service) previewRows(rows [][]string) (*PreviewResponse, error) {
	parsed, err := parseRows(rows, s.loc)
	if err != nil {
		if errors.Is(err, ErrTooManyRows) {
			return nil, apperror.BadRequest(err.Error())
		}
		return nil, apperror.Internal(err)
	}
	if len(parsed.Missing) > 0 {
		fields := make(map[string]string, len(parsed.Missing))
		for _, k := range parsed.Missing {
			fields[k] = "kolom wajib tidak ditemukan di baris header"
		}
		return nil, apperror.Validation(fields)
	}

	resp := &PreviewResponse{
		TotalRows: len(parsed.Rows),
		Rows:      parsed.Rows,
		Clashes:   detectImportClashes(parsed.Rows),
	}
	for _, r := range parsed.Rows {
		if r.Valid() {
			resp.ValidRows++
		}
	}
	resp.InvalidRows = resp.TotalRows - resp.ValidRows
	resp.CanCommit = resp.ValidRows > 0 && resp.InvalidRows == 0
	return resp, nil
}

// detectImportClashes mencari petugas yang sama di dua kegiatan dengan waktu tumpang tindih.
// Petugas dikenali dari teks kolom petugas (tanpa membedakan huruf besar/kecil).
func detectImportClashes(rows []ImportRow) []ClashInfo {
	clashes := []ClashInfo{}

	var (
		slots []Slot
		lines []int
		names []string
	)
	ids := map[string]uint{}
	for _, r := range rows {
		if !r.Valid() || r.Assignee == "" || r.Start == nil || r.End == nil {
			continue
		}
		key := strings.ToLower(r.Assignee)
		id, ok := ids[key]
		if !ok {
			id = uint(len(ids) + 1)
			ids[key] = id
		}
		slots = append(slots, Slot{Title: r.Activity, Start: *r.Start, End: *r.End, UserID: id})
		lines = append(lines, r.Line)
		names = append(names, r.Assignee)
	}

	for _, c := range DetectTimeClashes(slots) {
		clashes = append(clashes, ClashInfo{
			Type:  c.Type,
			Lines: []int{lines[c.A], lines[c.B]},
			Message: fmt.Sprintf("Baris %d dan %d: %s ditugaskan di waktu yang bertabrakan",
				lines[c.A], lines[c.B], names[c.A]),
		})
	}
	return clashes
}

// ListKegiatan ambil daftar kegiatan per halaman. Filter tanggal dan nama boleh kosong.
func (s *Service) ListKegiatan(ctx context.Context, filter KegiatanQuery) (*KegiatanListResponse, error) {
	// ?dari= dan ?sampai= harus format 2026-11-01 supaya bisa dibandingkan di database
	if !formatTanggalBenar(filter.Dari, s.loc) {
		return nil, apperror.BadRequest("Parameter dari harus format tanggal 2026-11-01")
	}
	if !formatTanggalBenar(filter.Sampai, s.loc) {
		return nil, apperror.BadRequest("Parameter sampai harus format tanggal 2026-11-01")
	}

	daftarKegiatan, total, err := s.repo.ListKegiatan(ctx, filter)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	hasil := &KegiatanListResponse{
		Items: make([]KegiatanResponse, 0, len(daftarKegiatan)), // array kosong, bukan null
		Meta:  pagination.NewMeta(filter.Params, total),
	}
	for i := range daftarKegiatan {
		hasil.Items = append(hasil.Items, toKegiatanResponse(&daftarKegiatan[i]))
	}
	return hasil, nil
}

// GetKegiatan ambil satu kegiatan.
func (s *Service) GetKegiatan(ctx context.Context, kegiatanID uint) (*KegiatanResponse, error) {
	kegiatan, err := s.repo.FindKegiatan(ctx, kegiatanID)
	if err != nil {
		return nil, kegiatanNotFoundOrInternal(err)
	}
	hasil := toKegiatanResponse(kegiatan)
	return &hasil, nil
}

// CreateKegiatan cek data -> simpan -> ambil ulang supaya nama komunitas ikut terkirim.
func (s *Service) CreateKegiatan(ctx context.Context, req KegiatanRequest) (*KegiatanResponse, error) {
	kegiatan, err := s.kegiatanDariRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateKegiatan(ctx, kegiatan); err != nil {
		return nil, kegiatanDuplicateOrInternal(err)
	}
	return s.GetKegiatan(ctx, kegiatan.ID)
}

// UpdateKegiatan cek data -> pastikan kegiatannya ada -> timpa isinya.
func (s *Service) UpdateKegiatan(ctx context.Context, kegiatanID uint, req KegiatanRequest) (*KegiatanResponse, error) {
	kegiatan, err := s.kegiatanDariRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	// cek dulu kegiatannya ada, supaya ID salah dapat 404
	if _, err := s.repo.FindKegiatan(ctx, kegiatanID); err != nil {
		return nil, kegiatanNotFoundOrInternal(err)
	}

	kegiatan.ID = kegiatanID
	if err := s.repo.UpdateKegiatan(ctx, kegiatan); err != nil {
		return nil, kegiatanDuplicateOrInternal(err)
	}
	return s.GetKegiatan(ctx, kegiatanID)
}

// DeleteKegiatan hapus kegiatan. Datanya tidak benar-benar hilang (deleted_at diisi).
func (s *Service) DeleteKegiatan(ctx context.Context, kegiatanID uint) error {
	if err := s.repo.DeleteKegiatan(ctx, kegiatanID); err != nil {
		return kegiatanNotFoundOrInternal(err)
	}
	return nil
}

// kegiatanDariRequest cek isi request satu per satu lalu ubah jadi data kegiatan.
// Kalau ada yang salah -> error 422 berisi pesan per field.
func (s *Service) kegiatanDariRequest(ctx context.Context, req KegiatanRequest) (*models.Kegiatan, error) {
	errorPerField := map[string]string{}

	nama := strings.TrimSpace(req.Nama)
	if nama == "" {
		errorPerField["nama_kegiatan"] = "wajib diisi"
	}

	// tanggal: parseDate ada di excel_importer.go, menerima 2026-11-01 atau 01/11/2026
	tanggal, err := parseDate(strings.TrimSpace(req.Tanggal), s.loc)
	if err != nil {
		errorPerField["tanggal"] = err.Error()
	}

	// jam: parseClock ada di excel_importer.go, menerima 07:00 atau 07.00
	jamMulai, menitMulai, errMulai := parseClock(strings.TrimSpace(req.WaktuMulai))
	if errMulai != nil {
		errorPerField["waktu_mulai"] = errMulai.Error()
	}

	// waktu_selesai boleh kosong; kalau diisi harus setelah waktu_mulai
	var waktuSelesai *string
	if teksSelesai := strings.TrimSpace(req.WaktuSelesai); teksSelesai != "" {
		jamSelesai, menitSelesai, err := parseClock(teksSelesai)
		if err != nil {
			errorPerField["waktu_selesai"] = err.Error()
		} else if errMulai == nil && jamSelesai*60+menitSelesai <= jamMulai*60+menitMulai {
			errorPerField["waktu_selesai"] = "harus setelah waktu_mulai"
		} else {
			jam := formatJam(jamSelesai, menitSelesai)
			waktuSelesai = &jam
		}
	}

	// komunitas boleh kosong, tapi kalau diisi harus ada di database
	if req.KomunitasID != nil {
		ada, err := s.repo.KomunitasAda(ctx, *req.KomunitasID)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if !ada {
			errorPerField["id_komunitas"] = "komunitas tidak ditemukan"
		}
	}

	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}

	return &models.Kegiatan{
		Nama:          nama,
		Tanggal:       tanggal,
		WaktuMulai:    formatJam(jamMulai, menitMulai),
		WaktuSelesai:  waktuSelesai,
		Tema:          teksAtauNil(req.Tema),
		NatsAlkitab:   teksAtauNil(req.NatsAlkitab),
		Pembicara:     teksAtauNil(req.Pembicara),
		Lokasi:        teksAtauNil(req.Lokasi),
		Deskripsi:     teksAtauNil(req.Deskripsi),
		KomunitasID:   req.KomunitasID,
		PerluEvaluasi: req.PerluEvaluasi,
	}, nil
}

// formatTanggalBenar: kosong dianggap benar (tidak difilter), selain itu harus format 2026-11-01.
func formatTanggalBenar(teks string, loc *time.Location) bool {
	if teks == "" {
		return true
	}
	_, err := time.ParseInLocation("2006-01-02", teks, loc)
	return err == nil
}

// formatJam: 7 jam 0 menit -> "07:00:00" (format kolom TIME di MySQL).
func formatJam(jam, menit int) string {
	return fmt.Sprintf("%02d:%02d:00", jam, menit)
}

// tampilJam: "07:00:00" dari database -> "07:00" untuk client.
func tampilJam(waktu string) string {
	if len(waktu) >= 5 {
		return waktu[:5]
	}
	return waktu
}

// teksAtauNil: teks kosong disimpan sebagai NULL, bukan string kosong.
func teksAtauNil(teks string) *string {
	teks = strings.TrimSpace(teks)
	if teks == "" {
		return nil
	}
	return &teks
}

// kegiatanNotFoundOrInternal: kegiatan tidak ditemukan -> 404, error lain -> 500.
func kegiatanNotFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("Kegiatan tidak ditemukan")
	}
	return apperror.Internal(err)
}

// kegiatanDuplicateOrInternal: nama + tanggal + jam mulai sudah dipakai -> 409, error lain -> 500.
func kegiatanDuplicateOrInternal(err error) error {
	if errors.Is(err, ErrDuplicate) {
		return apperror.Conflict("Kegiatan dengan nama, tanggal, dan jam mulai yang sama sudah ada")
	}
	return apperror.Internal(err)
}

// toKegiatanResponse ubah data dari database (models) jadi bentuk JSON untuk client (dto).
func toKegiatanResponse(kegiatan *models.Kegiatan) KegiatanResponse {
	hasil := KegiatanResponse{
		ID:            kegiatan.ID,
		Nama:          kegiatan.Nama,
		Tanggal:       kegiatan.Tanggal.Format("2006-01-02"),
		WaktuMulai:    tampilJam(kegiatan.WaktuMulai),
		Tema:          kegiatan.Tema,
		NatsAlkitab:   kegiatan.NatsAlkitab,
		Pembicara:     kegiatan.Pembicara,
		Lokasi:        kegiatan.Lokasi,
		Deskripsi:     kegiatan.Deskripsi,
		KomunitasID:   kegiatan.KomunitasID,
		PerluEvaluasi: kegiatan.PerluEvaluasi,
	}
	if kegiatan.WaktuSelesai != nil {
		jam := tampilJam(*kegiatan.WaktuSelesai)
		hasil.WaktuSelesai = &jam
	}
	if kegiatan.Komunitas != nil {
		hasil.NamaKomunitas = &kegiatan.Komunitas.Nama
	}
	return hasil
}

// TugaskanPetugas menugaskan satu jemaat ke kegiatan dengan satu keahlian.
// Syarat (error 422 per field): jemaat ada, aktif, bukan Guest, dan punya keahlian itu yang sudah
// diverifikasi admin. Kombinasi yang sama dua kali -> 409. Bentrok waktu dengan kegiatan lain di hari
// yang sama hanya jadi peringatan di response; penugasan tetap disimpan.
func (s *Service) TugaskanPetugas(ctx context.Context, kegiatanID uint, req PenugasanRequest) (*PenugasanResponse, error) {
	kegiatan, err := s.repo.FindKegiatan(ctx, kegiatanID)
	if err != nil {
		return nil, kegiatanNotFoundOrInternal(err)
	}

	errorPerField := map[string]string{}

	user, err := s.repo.FindUser(ctx, req.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		errorPerField["id_user"] = "jemaat tidak ditemukan"
	case err != nil:
		return nil, apperror.Internal(err)
	case !user.IsAktif:
		errorPerField["id_user"] = "akun jemaat tidak aktif"
	case user.Role.Nama == models.RoleGuest:
		errorPerField["id_user"] = "belum terdaftar sebagai jemaat"
	}

	if _, err := s.repo.FindKeahlian(ctx, req.KeahlianID); errors.Is(err, ErrNotFound) {
		errorPerField["id_keahlian"] = "keahlian tidak ditemukan"
	} else if err != nil {
		return nil, apperror.Internal(err)
	}

	// keahlian dicek hanya kalau jemaat dan keahliannya valid
	if len(errorPerField) == 0 {
		punya, err := s.repo.UserPunyaKeahlian(ctx, req.UserID, req.KeahlianID)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if !punya {
			errorPerField["id_keahlian"] = "jemaat tidak memiliki keahlian ini"
		}
	}
	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}

	// bentrok waktu: hanya peringatan, jadi dicari sebelum simpan
	tanggal := kegiatan.Tanggal.Format("2006-01-02")
	lainnya, err := s.repo.ListPenugasanUserPadaTanggal(ctx, req.UserID, tanggal, kegiatanID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	peringatan := s.cariBentrokWaktu(kegiatan, req.UserID, lainnya)

	penugasan := &models.JadwalPelayanan{
		KegiatanID:       kegiatanID,
		UserID:           req.UserID,
		KeahlianID:       req.KeahlianID,
		StatusKonfirmasi: models.PenugasanTerkonfirmasi, // ditugaskan admin = langsung terkonfirmasi (default ERD)
	}
	if err := s.repo.CreatePenugasan(ctx, penugasan); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, apperror.Conflict("Jemaat ini sudah ditugaskan dengan keahlian yang sama di kegiatan ini")
		}
		return nil, apperror.Internal(err)
	}

	// ambil ulang supaya nama jemaat dan keahlian ikut terkirim
	tersimpan, err := s.repo.FindPenugasan(ctx, kegiatanID, penugasan.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := toPenugasanResponse(tersimpan)
	hasil.Peringatan = peringatan
	return &hasil, nil
}

// ListPetugas ambil semua petugas satu kegiatan.
func (s *Service) ListPetugas(ctx context.Context, kegiatanID uint) (*PetugasListResponse, error) {
	if _, err := s.repo.FindKegiatan(ctx, kegiatanID); err != nil {
		return nil, kegiatanNotFoundOrInternal(err)
	}
	daftar, err := s.repo.ListPetugasKegiatan(ctx, kegiatanID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := &PetugasListResponse{Items: make([]PenugasanResponse, 0, len(daftar))} // array kosong, bukan null
	for i := range daftar {
		hasil.Items = append(hasil.Items, toPenugasanResponse(&daftar[i]))
	}
	return hasil, nil
}

// HapusPetugas membatalkan satu penugasan. Datanya tidak benar-benar hilang (deleted_at diisi).
func (s *Service) HapusPetugas(ctx context.Context, kegiatanID, penjadwalanID uint) error {
	if err := s.repo.DeletePenugasan(ctx, kegiatanID, penjadwalanID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperror.NotFound("Penugasan tidak ditemukan")
		}
		return apperror.Internal(err)
	}
	return nil
}

// JadwalSaya ambil tugas pelayanan milik jemaat yang sedang login. Filter tanggal boleh kosong.
func (s *Service) JadwalSaya(ctx context.Context, userID uint, filter JadwalSayaQuery) (*JadwalSayaListResponse, error) {
	if !formatTanggalBenar(filter.Dari, s.loc) {
		return nil, apperror.BadRequest("Parameter dari harus format tanggal 2026-11-01")
	}
	if !formatTanggalBenar(filter.Sampai, s.loc) {
		return nil, apperror.BadRequest("Parameter sampai harus format tanggal 2026-11-01")
	}

	daftar, total, err := s.repo.ListJadwalUser(ctx, userID, filter)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := &JadwalSayaListResponse{
		Items: make([]JadwalSayaResponse, 0, len(daftar)),
		Meta:  pagination.NewMeta(filter.Params, total),
	}
	for i := range daftar {
		hasil.Items = append(hasil.Items, toJadwalSayaResponse(&daftar[i]))
	}
	return hasil, nil
}

// rentangKegiatan mengubah tanggal + jam kegiatan jadi waktu mulai dan selesai.
// Kegiatan tanpa jam selesai dianggap berdurasi 1 menit: bentrok hanya kalau jam mulainya sama,
// atau jatuh di dalam kegiatan lain yang punya jam selesai. ok = false kalau jam di database tidak terbaca.
func (s *Service) rentangKegiatan(kegiatan *models.Kegiatan) (mulai, selesai time.Time, ok bool) {
	jamMulai, err := time.Parse("15:04:05", kegiatan.WaktuMulai)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	tahun, bulan, hari := kegiatan.Tanggal.Date()
	mulai = time.Date(tahun, bulan, hari, jamMulai.Hour(), jamMulai.Minute(), 0, 0, s.loc)
	selesai = mulai.Add(time.Minute)

	if kegiatan.WaktuSelesai != nil {
		if jamSelesai, err := time.Parse("15:04:05", *kegiatan.WaktuSelesai); err == nil {
			akhir := time.Date(tahun, bulan, hari, jamSelesai.Hour(), jamSelesai.Minute(), 0, 0, s.loc)
			if akhir.After(mulai) {
				selesai = akhir
			}
		}
	}
	return mulai, selesai, true
}

// cariBentrokWaktu memakai DetectTimeClashes (clash_detector.go): kegiatan baru jadi slot pertama,
// lalu hanya bentrok yang melibatkan slot itu yang dilaporkan.
func (s *Service) cariBentrokWaktu(baru *models.Kegiatan, userID uint, lainnya []models.JadwalPelayanan) []string {
	peringatan := []string{} // array kosong, bukan null
	mulai, selesai, ok := s.rentangKegiatan(baru)
	if !ok {
		return peringatan
	}

	slots := []Slot{{Title: baru.Nama, Start: mulai, End: selesai, UserID: userID}}
	for i := range lainnya {
		mulaiLain, selesaiLain, ok := s.rentangKegiatan(&lainnya[i].Kegiatan)
		if !ok {
			continue
		}
		slots = append(slots, Slot{Title: lainnya[i].Kegiatan.Nama, Start: mulaiLain, End: selesaiLain, UserID: userID})
	}

	for _, bentrok := range DetectTimeClashes(slots) {
		if bentrok.A != 0 { // pasangan yang tidak melibatkan kegiatan baru dilewati
			continue
		}
		lain := slots[bentrok.B]
		peringatan = append(peringatan, fmt.Sprintf("Bentrok waktu dengan kegiatan %q (mulai %s)", lain.Title, lain.Start.Format("15:04")))
	}
	return peringatan
}

// toPenugasanResponse ubah data dari database (models) jadi bentuk JSON untuk client (dto).
func toPenugasanResponse(penugasan *models.JadwalPelayanan) PenugasanResponse {
	return PenugasanResponse{
		ID:           penugasan.ID,
		KegiatanID:   penugasan.KegiatanID,
		UserID:       penugasan.UserID,
		KodeJemaat:   penugasan.User.KodeJemaat,
		Nama:         penugasan.User.Nama,
		KeahlianID:   penugasan.KeahlianID,
		NamaKeahlian: penugasan.Keahlian.Nama,
		Status:       penugasan.StatusKonfirmasi,
	}
}

func toJadwalSayaResponse(penugasan *models.JadwalPelayanan) JadwalSayaResponse {
	hasil := JadwalSayaResponse{
		ID:           penugasan.ID,
		KegiatanID:   penugasan.KegiatanID,
		NamaKegiatan: penugasan.Kegiatan.Nama,
		Tanggal:      penugasan.Kegiatan.Tanggal.Format("2006-01-02"),
		WaktuMulai:   tampilJam(penugasan.Kegiatan.WaktuMulai),
		Lokasi:       penugasan.Kegiatan.Lokasi,
		KeahlianID:   penugasan.KeahlianID,
		NamaKeahlian: penugasan.Keahlian.Nama,
		Status:       penugasan.StatusKonfirmasi,
	}
	if penugasan.Kegiatan.WaktuSelesai != nil {
		jam := tampilJam(*penugasan.Kegiatan.WaktuSelesai)
		hasil.WaktuSelesai = &jam
	}
	return hasil
}

// ImportSchedules menyimpan isi sheet "Kegiatan". Kegiatan yang sudah ada (nama + tanggal + jam mulai sama)
// diperbarui, sisanya dibuat baru. Ditolak kalau masih ada baris error (cek dulu lewat preview).
// Setelah disimpan, KEBAKTIAN yang perlu_evaluasi otomatis dibuatkan kuesioner evaluasi.
func (s *Service) ImportSchedules(ctx context.Context, r io.Reader) (*ImportSimpanResponse, error) {
	rows, err := readSheet(r)
	if err != nil {
		if errors.Is(err, ErrNoSheet) {
			return nil, apperror.BadRequest("File Excel tidak memiliki sheet Kegiatan")
		}
		e := apperror.BadRequest("File harus berupa Excel (.xlsx) yang valid")
		e.Err = err
		return nil, e
	}

	parsed, err := parseRows(rows, s.loc)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}
	if len(parsed.Missing) > 0 {
		return nil, apperror.BadRequest("Kolom wajib tidak ditemukan di baris header Excel")
	}
	for _, row := range parsed.Rows {
		if !row.Valid() {
			return nil, apperror.Validation(map[string]string{"file": "Perbaiki error di baris Excel sebelum menyimpan"})
		}
	}

	var kegiatanBaru, kegiatanDiperbarui []models.Kegiatan
	var kuesionerBaru, kuesionerDiperbarui []bool // true = kegiatan ini dibuatkan kuesioner otomatis
	komunitasCache := map[string]*uint{}          // nama komunitas (huruf kecil) -> id, supaya tidak query berulang

	for _, row := range parsed.Rows {
		// komunitas yang tidak ditemukan dibiarkan kosong (dianggap Umum)
		var idKomunitas *uint
		namaKomunitas := strings.ToLower(strings.TrimSpace(row.Komunitas))
		if namaKomunitas != "" {
			id, ada := komunitasCache[namaKomunitas]
			if !ada {
				komunitas, err := s.repo.FindKomunitasByNama(ctx, row.Komunitas)
				if err != nil && !errors.Is(err, ErrNotFound) {
					return nil, apperror.Internal(err)
				}
				if komunitas != nil {
					id = &komunitas.ID
				}
				komunitasCache[namaKomunitas] = id
			}
			idKomunitas = id
		}

		y, mo, d := row.Start.Date()
		kegiatan := models.Kegiatan{
			Nama:          row.Activity,
			Tanggal:       time.Date(y, mo, d, 0, 0, 0, 0, s.loc),
			WaktuMulai:    row.WaktuMulai + ":00",
			Tema:          teksAtauNil(row.Tema),
			NatsAlkitab:   teksAtauNil(row.NatsAlkitab),
			Pembicara:     teksAtauNil(row.Pembicara),
			Lokasi:        teksAtauNil(row.Lokasi),
			Deskripsi:     teksAtauNil(row.Deskripsi),
			KomunitasID:   idKomunitas,
			PerluEvaluasi: row.PerluEvaluasi,
		}
		if row.WaktuSelesai != "" {
			jam := row.WaktuSelesai + ":00"
			kegiatan.WaktuSelesai = &jam
		}

		// untuk sekarang hanya KEBAKTIAN yang dapat kuesioner (jenis kosong dianggap KEBAKTIAN)
		perluKuesioner := row.PerluEvaluasi && (row.JenisKegiatan == "" || row.JenisKegiatan == JenisKebaktian)

		lama, err := s.repo.FindKegiatanByUnik(ctx, kegiatan.Nama, row.Date, kegiatan.WaktuMulai)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, apperror.Internal(err)
		}
		if lama != nil {
			kegiatan.ID = lama.ID
			kegiatanDiperbarui = append(kegiatanDiperbarui, kegiatan)
			kuesionerDiperbarui = append(kuesionerDiperbarui, perluKuesioner)
		} else {
			kegiatanBaru = append(kegiatanBaru, kegiatan)
			kuesionerBaru = append(kuesionerBaru, perluKuesioner)
		}
	}

	if err := s.repo.ImportKegiatan(ctx, kegiatanBaru, kegiatanDiperbarui); err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := &ImportSimpanResponse{Ditambah: len(kegiatanBaru), Diperbarui: len(kegiatanDiperbarui)}

	// ID kegiatan baru sudah terisi setelah ImportKegiatan
	var daftarID []uint
	for i := range kegiatanBaru {
		if kuesionerBaru[i] {
			daftarID = append(daftarID, kegiatanBaru[i].ID)
		}
	}
	for i := range kegiatanDiperbarui {
		if kuesionerDiperbarui[i] {
			daftarID = append(daftarID, kegiatanDiperbarui[i].ID)
		}
	}
	if s.kuesioner != nil && len(daftarID) > 0 {
		dibuat, err := s.kuesioner.BuatKuesionerOtomatis(ctx, daftarID)
		if err != nil {
			// kegiatan sudah tersimpan; kuesioner yang gagal bisa dibuat manual lewat POST /kuesioner
			return nil, apperror.Internal(fmt.Errorf("kegiatan tersimpan, tapi kuesioner gagal dibuat: %w", err))
		}
		hasil.KuesionerDibuat = dibuat
	}
	return hasil, nil
}
