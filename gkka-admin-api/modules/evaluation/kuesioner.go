package evaluation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
	"sim-g-backend/modules/pengaturan"
)

// Jadwal kuesioner: kapan form evaluasi boleh diisi lewat app.
// Satu Google Form dipakai ulang tiap minggu, yang berganti hanya jadwal buka/tutupnya.

// formatWaktu: format jam buka/tutup dari web admin, contoh "2026-11-01 10:30:00".
const formatWaktu = "2006-01-02 15:04:05"

// aturanJadwal: pengaturan admin untuk jadwal kuesioner (tabel pengaturan).
// Hanya dipakai saat kuesioner dibuat, kuesioner lama tidak ikut berubah.
type aturanJadwal struct {
	lamaHari    int    // ditutup H+lamaHari
	jamTutup    string // contoh "23:59"
	durasiMenit int    // lama kegiatan kalau jam selesai kosong
}

// ambilAturanJadwal membaca pengaturan admin, yang belum diisi memakai nilai bawaan.
func (service *Service) ambilAturanJadwal(ctx context.Context) (aturanJadwal, error) {
	isi, err := service.repo.AmbilPengaturan(ctx,
		pengaturan.KunciLamaBukaHari, pengaturan.KunciJamTutup, pengaturan.KunciDurasiKegiatan)
	if err != nil {
		return aturanJadwal{}, err
	}
	return aturanJadwal{
		lamaHari:    pengaturan.Angka(isi, pengaturan.KunciLamaBukaHari),
		jamTutup:    pengaturan.Nilai(isi, pengaturan.KunciJamTutup),
		durasiMenit: pengaturan.Angka(isi, pengaturan.KunciDurasiKegiatan),
	}, nil
}

// KuesionerRequest dipakai untuk membuat jadwal kuesioner.
// MINGGUAN wajib id_kegiatan (kegiatan yang perlu_evaluasi), TAHUNAN wajib tahun + jadwal.
type KuesionerRequest struct {
	JenisKuesioner string  `json:"jenis_kuesioner" binding:"required,oneof=MINGGUAN TAHUNAN"`
	KegiatanID     *uint   `json:"id_kegiatan" binding:"omitempty,gt=0"`
	Tahun          *int    `json:"tahun" binding:"omitempty,min=2000,max=2100"`
	DibukaPada     *string `json:"dibuka_pada"`  // kosong (MINGGUAN) = saat kegiatan selesai
	DitutupPada    *string `json:"ditutup_pada"` // kosong (MINGGUAN) = H+N pukul 23.59
}

// JadwalKuesionerRequest dipakai admin untuk mengubah jam buka/tutup.
// Isi status untuk tombol cepat: {"status": "BUKA"} atau {"status": "TUTUP"}.
// Atau isi jam: {"dibuka_pada": "2026-11-01 10:30:00", "ditutup_pada": "2026-11-03 23:59:00"}
// (field yang tidak dikirim tetap memakai jam lama).
type JadwalKuesionerRequest struct {
	Status      string  `json:"status" binding:"omitempty,oneof=BUKA TUTUP"`
	DibukaPada  *string `json:"dibuka_pada"`
	DitutupPada *string `json:"ditutup_pada"`
}

// Status kuesioner dihitung dari jam sekarang, tidak disimpan di database.
const (
	StatusBelumDibuka = "BELUM_DIBUKA"
	StatusDibuka      = "DIBUKA"
	StatusDitutup     = "DITUTUP"
)

type KuesionerResponse struct {
	ID             uint       `json:"id"`
	JenisKuesioner string     `json:"jenis_kuesioner"`
	KegiatanID     *uint      `json:"id_kegiatan"`
	Tahun          *int       `json:"tahun"`
	JudulForm      string     `json:"judul_form"`
	Tema           *string    `json:"tema"`
	Pembicara      *string    `json:"pembicara"`
	DibukaPada     *time.Time `json:"dibuka_pada"`
	DitutupPada    *time.Time `json:"ditutup_pada"`
	KodeAkses      string     `json:"kode_akses"` // diumumkan saat ibadah, diisi jemaat sebelum mengisi form
	Status         string     `json:"status"`
}

// ================= service =================

// ListKuesioner: jadwal kuesioner terbaru di atas. Filter jenis boleh kosong.
func (service *Service) ListKuesioner(ctx context.Context, jenis string) ([]KuesionerResponse, error) {
	jenis = strings.ToUpper(strings.TrimSpace(jenis))
	if jenis != "" && !jenisValid(jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	daftar, err := service.repo.ListKuesioner(ctx, jenis)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	sekarang := time.Now()
	hasil := make([]KuesionerResponse, 0, len(daftar))
	for i := range daftar {
		hasil = append(hasil, toKuesionerResponse(&daftar[i], sekarang))
	}
	return hasil, nil
}

// CreateKuesioner membuat jadwal kuesioner. MINGGUAN: judul, tema, dan pembicara disalin dari kegiatan,
// jadwal kosong diisi jadwal bawaan. Jam yang diisi admin menimpa jadwal bawaan.
func (service *Service) CreateKuesioner(ctx context.Context, req KuesionerRequest) (*KuesionerResponse, error) {
	errorPerField := map[string]string{}
	dibukaAdmin := service.bacaWaktu(req.DibukaPada, "dibuka_pada", errorPerField)
	ditutupAdmin := service.bacaWaktu(req.DitutupPada, "ditutup_pada", errorPerField)

	var kuesioner *models.KuesionerEvaluasi
	switch req.JenisKuesioner {
	case models.KuesionerMingguan:
		if req.KegiatanID == nil {
			errorPerField["id_kegiatan"] = "wajib diisi untuk MINGGUAN"
			return nil, apperror.Validation(errorPerField)
		}
		kegiatan, err := service.repo.FindKegiatan(ctx, *req.KegiatanID)
		if errors.Is(err, ErrNotFound) {
			errorPerField["id_kegiatan"] = "kegiatan tidak ditemukan"
			return nil, apperror.Validation(errorPerField)
		}
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if !kegiatan.PerluEvaluasi {
			errorPerField["id_kegiatan"] = "kegiatan ini tidak perlu evaluasi"
			return nil, apperror.Validation(errorPerField)
		}
		aturan, err := service.ambilAturanJadwal(ctx)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		kuesioner, err = kuesionerMingguan(kegiatan, aturan)
		if err != nil {
			return nil, apperror.Internal(err)
		}

	case models.KuesionerTahunan:
		if req.Tahun == nil {
			errorPerField["tahun"] = "wajib diisi untuk TAHUNAN"
		}
		if dibukaAdmin == nil && errorPerField["dibuka_pada"] == "" {
			errorPerField["dibuka_pada"] = "wajib diisi untuk TAHUNAN"
		}
		if ditutupAdmin == nil && errorPerField["ditutup_pada"] == "" {
			errorPerField["ditutup_pada"] = "wajib diisi untuk TAHUNAN"
		}
		if len(errorPerField) > 0 {
			return nil, apperror.Validation(errorPerField)
		}
		kuesioner = &models.KuesionerEvaluasi{
			JenisKuesioner: models.KuesionerTahunan,
			Tahun:          req.Tahun,
			JudulForm:      fmt.Sprintf("Evaluasi Tahunan %d", *req.Tahun),
		}
	}

	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}
	// jam dari admin menimpa jadwal bawaan
	if dibukaAdmin != nil {
		kuesioner.DibukaPada = dibukaAdmin
	}
	if ditutupAdmin != nil {
		kuesioner.DitutupPada = ditutupAdmin
	}
	if err := cekJadwal(*kuesioner.DibukaPada, *kuesioner.DitutupPada); err != nil {
		return nil, err
	}

	if err := service.simpanDenganKode(ctx, kuesioner); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, apperror.Conflict("Kuesioner untuk kegiatan / tahun ini sudah ada")
		}
		return nil, apperror.Internal(err)
	}
	hasil := toKuesionerResponse(kuesioner, time.Now())
	return &hasil, nil
}

// BuatKuesionerOtomatis dipanggil setelah import Excel kegiatan: setiap kegiatan yang perlu_evaluasi
// dibuatkan kuesioner MINGGUAN dengan jadwal bawaan. Yang sudah punya kuesioner dilewati.
// Mengembalikan jumlah kuesioner yang baru dibuat.
func (service *Service) BuatKuesionerOtomatis(ctx context.Context, daftarKegiatanID []uint) (int, error) {
	dibuat := 0
	aturan, err := service.ambilAturanJadwal(ctx)
	if err != nil {
		return dibuat, err
	}
	for _, kegiatanID := range daftarKegiatanID {
		kegiatan, err := service.repo.FindKegiatan(ctx, kegiatanID)
		if err != nil {
			return dibuat, err
		}
		if !kegiatan.PerluEvaluasi {
			continue
		}
		kuesioner, err := kuesionerMingguan(kegiatan, aturan)
		if err != nil {
			return dibuat, err
		}
		// jaga-jaga kegiatan lewat tengah malam: jangan simpan kuesioner yang langsung tertutup
		if !kuesioner.DitutupPada.After(*kuesioner.DibukaPada) {
			return dibuat, fmt.Errorf("jadwal kuesioner %q tidak valid: jam tutup %s sebelum jam buka %s",
				kegiatan.Nama, kuesioner.DitutupPada.Format(formatWaktu), kuesioner.DibukaPada.Format(formatWaktu))
		}
		err = service.simpanDenganKode(ctx, kuesioner)
		if errors.Is(err, ErrDuplicate) {
			continue // kegiatan ini sudah punya kuesioner
		}
		if err != nil {
			return dibuat, err
		}
		dibuat++
	}
	return dibuat, nil
}

// UbahJadwalKuesioner mengganti jam buka/tutup, atau buka/tutup sekarang lewat status.
func (service *Service) UbahJadwalKuesioner(ctx context.Context, kuesionerID uint, req JadwalKuesionerRequest) (*KuesionerResponse, error) {
	kuesioner, err := service.repo.FindKuesioner(ctx, kuesionerID)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.NotFound("Kuesioner tidak ditemukan")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	sekarang := time.Now().In(service.loc).Truncate(time.Second)
	switch req.Status {
	case "BUKA":
		// buka sekarang; jam tutup lama dipakai kalau masih di depan, kalau sudah lewat dihitung ulang H+N
		kuesioner.DibukaPada = &sekarang
		if kuesioner.DitutupPada == nil || !kuesioner.DitutupPada.After(sekarang) {
			aturan, err := service.ambilAturanJadwal(ctx)
			if err != nil {
				return nil, apperror.Internal(err)
			}
			ditutup := jamTutup(sekarang, aturan)
			kuesioner.DitutupPada = &ditutup
		}

	case "TUTUP":
		// tutup sekarang; jam buka tidak diubah, kecuali belum pernah dibuka
		kuesioner.DitutupPada = &sekarang
		if kuesioner.DibukaPada == nil || kuesioner.DibukaPada.After(sekarang) {
			kuesioner.DibukaPada = &sekarang
		}

	default:
		errorPerField := map[string]string{}
		dibuka := service.bacaWaktu(req.DibukaPada, "dibuka_pada", errorPerField)
		ditutup := service.bacaWaktu(req.DitutupPada, "ditutup_pada", errorPerField)
		if len(errorPerField) > 0 {
			return nil, apperror.Validation(errorPerField)
		}
		if dibuka == nil && ditutup == nil {
			return nil, apperror.BadRequest("Isi status, dibuka_pada, atau ditutup_pada")
		}
		// field yang tidak dikirim tetap memakai jam lama
		if dibuka != nil {
			kuesioner.DibukaPada = dibuka
		}
		if ditutup != nil {
			kuesioner.DitutupPada = ditutup
		}
		if kuesioner.DibukaPada == nil || kuesioner.DitutupPada == nil {
			return nil, apperror.Validation(map[string]string{"dibuka_pada": "jam buka dan tutup wajib diisi"})
		}
		if err := cekJadwal(*kuesioner.DibukaPada, *kuesioner.DitutupPada); err != nil {
			return nil, err
		}
	}

	if err := service.repo.UpdateJadwalKuesioner(ctx, kuesioner); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.NotFound("Kuesioner tidak ditemukan")
		}
		return nil, apperror.Internal(err)
	}
	hasil := toKuesionerResponse(kuesioner, sekarang)
	return &hasil, nil
}

// kuesionerMingguan menyusun kuesioner dari kegiatan: judul, tema, pembicara disalin, jadwal bawaan.
func kuesionerMingguan(kegiatan *models.Kegiatan, aturan aturanJadwal) (*models.KuesionerEvaluasi, error) {
	dibuka, ditutup, err := jadwalBawaan(kegiatan, aturan)
	if err != nil {
		return nil, err
	}
	return &models.KuesionerEvaluasi{
		JenisKuesioner: models.KuesionerMingguan,
		KegiatanID:     &kegiatan.ID,
		JudulForm:      fmt.Sprintf("Evaluasi %s, %s", kegiatan.Nama, kegiatan.Tanggal.Format("02-01-2006")),
		Tema:           kegiatan.Tema,
		Pembicara:      kegiatan.Pembicara,
		DibukaPada:     &dibuka,
		DitutupPada:    &ditutup,
	}, nil
}

// jadwalBawaan: dibuka saat kegiatan selesai. Kalau jam selesai kosong, dianggap selesai
// jam mulai + durasi bawaan. Ditutup H+lamaHari pada jam tutup (pengaturan admin).
func jadwalBawaan(kegiatan *models.Kegiatan, aturan aturanJadwal) (dibuka, ditutup time.Time, err error) {
	t := kegiatan.Tanggal // lokasi waktu = APP_TIMEZONE (diatur di koneksi database)

	if kegiatan.WaktuSelesai != nil && *kegiatan.WaktuSelesai != "" {
		pukul, err := time.Parse("15:04:05", *kegiatan.WaktuSelesai)
		if err != nil {
			return dibuka, ditutup, fmt.Errorf("jam selesai %q tidak valid: %w", *kegiatan.WaktuSelesai, err)
		}
		dibuka = time.Date(t.Year(), t.Month(), t.Day(), pukul.Hour(), pukul.Minute(), 0, 0, t.Location())
	} else {
		pukul, err := time.Parse("15:04:05", kegiatan.WaktuMulai)
		if err != nil {
			return dibuka, ditutup, fmt.Errorf("jam mulai %q tidak valid: %w", kegiatan.WaktuMulai, err)
		}
		mulai := time.Date(t.Year(), t.Month(), t.Day(), pukul.Hour(), pukul.Minute(), 0, 0, t.Location())
		dibuka = mulai.Add(time.Duration(aturan.durasiMenit) * time.Minute)
	}

	return dibuka, jamTutup(t, aturan), nil
}

// jamTutup: tanggal + lamaHari, pada jam tutup dari pengaturan (contoh 23:59).
func jamTutup(tanggal time.Time, aturan aturanJadwal) time.Time {
	pukul, err := time.Parse("15:04", aturan.jamTutup)
	if err != nil { // isi pengaturan rusak, pakai 23:59
		pukul = time.Date(0, 1, 1, 23, 59, 0, 0, time.UTC)
	}
	return time.Date(tanggal.Year(), tanggal.Month(), tanggal.Day()+aturan.lamaHari,
		pukul.Hour(), pukul.Minute(), 0, 0, tanggal.Location())
}

func cekJadwal(dibuka, ditutup time.Time) error {
	if !ditutup.After(dibuka) {
		return apperror.Validation(map[string]string{"ditutup_pada": "harus setelah dibuka_pada"})
	}
	return nil
}

// bacaWaktu: teks "2026-11-01 10:30:00" -> waktu. Kosong = nil (tidak diisi).
// Format salah dicatat di errorPerField.
func (service *Service) bacaWaktu(teks *string, namaField string, errorPerField map[string]string) *time.Time {
	if teks == nil || strings.TrimSpace(*teks) == "" {
		return nil
	}
	waktu, err := time.ParseInLocation(formatWaktu, strings.TrimSpace(*teks), service.loc)
	if err != nil {
		errorPerField[namaField] = "format harus YYYY-MM-DD HH:MM:SS"
		return nil
	}
	return &waktu
}

func statusKuesioner(kuesioner *models.KuesionerEvaluasi, sekarang time.Time) string {
	switch {
	case kuesioner.DibukaPada == nil || sekarang.Before(*kuesioner.DibukaPada):
		return StatusBelumDibuka
	case kuesioner.DitutupPada != nil && !sekarang.Before(*kuesioner.DitutupPada):
		return StatusDitutup
	}
	return StatusDibuka
}

func toKuesionerResponse(kuesioner *models.KuesionerEvaluasi, sekarang time.Time) KuesionerResponse {
	return KuesionerResponse{
		ID:             kuesioner.ID,
		JenisKuesioner: kuesioner.JenisKuesioner,
		KegiatanID:     kuesioner.KegiatanID,
		Tahun:          kuesioner.Tahun,
		JudulForm:      kuesioner.JudulForm,
		Tema:           kuesioner.Tema,
		Pembicara:      kuesioner.Pembicara,
		DibukaPada:     kuesioner.DibukaPada,
		DitutupPada:    kuesioner.DitutupPada,
		KodeAkses:      kuesioner.KodeAkses,
		Status:         statusKuesioner(kuesioner, sekarang),
	}
}

// ================= repository =================

func (repo *gormRepository) ListKuesioner(ctx context.Context, jenis string) ([]models.KuesionerEvaluasi, error) {
	query := repo.db.WithContext(ctx)
	if jenis != "" {
		query = query.Where("jenis_kuesioner = ?", jenis)
	}
	var daftar []models.KuesionerEvaluasi
	// dibatasi: kuesioner bertambah tiap minggu, admin cukup melihat yang terbaru
	err := query.Order("dibuka_pada DESC").Limit(100).Find(&daftar).Error
	return daftar, err
}

func (repo *gormRepository) FindKuesioner(ctx context.Context, kuesionerID uint) (*models.KuesionerEvaluasi, error) {
	var kuesioner models.KuesionerEvaluasi
	if err := repo.db.WithContext(ctx).First(&kuesioner, kuesionerID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &kuesioner, nil
}

func (repo *gormRepository) FindKegiatan(ctx context.Context, kegiatanID uint) (*models.Kegiatan, error) {
	var kegiatan models.Kegiatan
	if err := repo.db.WithContext(ctx).First(&kegiatan, kegiatanID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &kegiatan, nil
}

// CreateKuesioner: ErrKodeDipakai kalau kode bentrok, ErrDuplicate kalau kegiatan/tahun sudah punya kuesioner.
func (repo *gormRepository) CreateKuesioner(ctx context.Context, kuesioner *models.KuesionerEvaluasi) error {
	err := repo.db.WithContext(ctx).Create(kuesioner).Error
	if kodeBentrok(err) {
		return ErrKodeDipakai
	}
	return mapErr(err)
}

func (repo *gormRepository) UpdateJadwalKuesioner(ctx context.Context, kuesioner *models.KuesionerEvaluasi) error {
	result := repo.db.WithContext(ctx).Model(kuesioner).Updates(map[string]any{
		"dibuka_pada":  kuesioner.DibukaPada,
		"ditutup_pada": kuesioner.DitutupPada,
	})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// KuesionerDibuka: kuesioner yang sedang dibuka. Kalau beberapa kebaktian dibuka bersamaan
// (mis. Umum 1 dan Umum 2), ambil yang ditutup paling akhir.
func (repo *gormRepository) KuesionerDibuka(ctx context.Context, jenis string, sekarang time.Time) (*models.KuesionerEvaluasi, error) {
	var kuesioner models.KuesionerEvaluasi
	// Kegiatan + komunitasnya ikut dimuat untuk header Google Form (questionnaire_header.go)
	err := repo.db.WithContext(ctx).Preload("Kegiatan.Komunitas").
		Where("jenis_kuesioner = ? AND dibuka_pada <= ? AND ditutup_pada > ?", jenis, sekarang, sekarang).
		Order("ditutup_pada DESC").
		First(&kuesioner).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &kuesioner, nil
}

// KuesionerBerikutnya: kuesioner terdekat yang belum dibuka.
func (repo *gormRepository) KuesionerBerikutnya(ctx context.Context, jenis string, sekarang time.Time) (*models.KuesionerEvaluasi, error) {
	var kuesioner models.KuesionerEvaluasi
	err := repo.db.WithContext(ctx).
		Where("jenis_kuesioner = ? AND dibuka_pada > ?", jenis, sekarang).
		Order("dibuka_pada ASC").
		First(&kuesioner).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &kuesioner, nil
}
