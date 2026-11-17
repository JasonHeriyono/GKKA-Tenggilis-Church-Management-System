package evaluation

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sim-g-backend/models"
)

// Error umum dari repository; service mengubahnya jadi pesan HTTP (404 / 409).
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data duplikat")
)

// Repository: daftar fungsi query database sesuai kebutuhan service.
type Repository interface {
	ListPertanyaan(ctx context.Context, filter PertanyaanQuery) ([]models.PertanyaanEvaluasi, error)
	FindPertanyaan(ctx context.Context, pertanyaanID uint) (*models.PertanyaanEvaluasi, error)
	CreatePertanyaan(ctx context.Context, pertanyaan *models.PertanyaanEvaluasi) error
	UpdatePertanyaan(ctx context.Context, pertanyaan *models.PertanyaanEvaluasi, labelOpsi []string) error
	SetAktifPertanyaan(ctx context.Context, pertanyaanID uint, isAktif bool) error
	DeletePertanyaan(ctx context.Context, pertanyaanID uint) error
	SimpanIDGoogle(ctx context.Context, daftarPertanyaan []models.PertanyaanEvaluasi) error
	ImportPertanyaan(ctx context.Context, pertanyaanBaru, pertanyaanDipakaiUlang []models.PertanyaanEvaluasi, idDinonaktifkan []uint) error

	// jadwal kuesioner (kuesioner.go)
	ListKuesioner(ctx context.Context, jenis string) ([]models.KuesionerEvaluasi, error)
	FindKuesioner(ctx context.Context, kuesionerID uint) (*models.KuesionerEvaluasi, error)
	FindKegiatan(ctx context.Context, kegiatanID uint) (*models.Kegiatan, error)
	CreateKuesioner(ctx context.Context, kuesioner *models.KuesionerEvaluasi) error
	UpdateJadwalKuesioner(ctx context.Context, kuesioner *models.KuesionerEvaluasi) error
	KuesionerDibuka(ctx context.Context, jenis string, sekarang time.Time) (*models.KuesionerEvaluasi, error)
	KuesionerBerikutnya(ctx context.Context, jenis string, sekarang time.Time) (*models.KuesionerEvaluasi, error)
	KodeAksesDipakai(ctx context.Context, kode string) (bool, error)
	FindKuesionerByKode(ctx context.Context, kode string) (*models.KuesionerEvaluasi, error)

	// respon Google Form (tarik_respon.go)
	ResponTersimpan(ctx context.Context, daftarID []string) (map[string]bool, error)
	SimpanRespon(ctx context.Context, respon *models.ResponEvaluasi) error

	// pengaturan aplikasi (dipakai template header Google Form)
	AmbilPengaturan(ctx context.Context, daftarKunci ...string) (map[string]string, error)
	SimpanPengaturan(ctx context.Context, isi map[string]string) error
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// ListPertanyaan ambil semua pertanyaan + pilihan jawabannya (tidak dibagi per halaman karena datanya sedikit).
func (repo *gormRepository) ListPertanyaan(ctx context.Context, filter PertanyaanQuery) ([]models.PertanyaanEvaluasi, error) {
	query := repo.db.WithContext(ctx).Preload("Opsi", urutOpsi)
	if filter.Jenis != "" {
		query = query.Where("jenis_kuesioner = ?", filter.Jenis)
	}
	if filter.Aktif != nil {
		query = query.Where("is_aktif = ?", *filter.Aktif)
	}
	var daftarPertanyaan []models.PertanyaanEvaluasi
	err := query.Order("urutan ASC").Find(&daftarPertanyaan).Error
	return daftarPertanyaan, err
}

// urutOpsi mengurutkan pilihan jawaban dari urutan 1, 2, 3, dst.
func urutOpsi(db *gorm.DB) *gorm.DB { return db.Order("urutan ASC") }

// FindPertanyaan mengambil satu pertanyaan beserta opsinya, ErrNotFound jika tidak ada.
func (repo *gormRepository) FindPertanyaan(ctx context.Context, pertanyaanID uint) (*models.PertanyaanEvaluasi, error) {
	var pertanyaan models.PertanyaanEvaluasi
	if err := repo.db.WithContext(ctx).Preload("Opsi", urutOpsi).First(&pertanyaan, pertanyaanID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &pertanyaan, nil
}

// CreatePertanyaan simpan pertanyaan + pilihan jawabannya sekaligus (kalau salah satu gagal, semua dibatalkan).
func (repo *gormRepository) CreatePertanyaan(ctx context.Context, pertanyaan *models.PertanyaanEvaluasi) error {
	return mapErr(repo.db.WithContext(ctx).Create(pertanyaan).Error)
}

// UpdatePertanyaan ubah pertanyaan + pilihan jawabannya (kalau salah satu gagal, semua dibatalkan).
func (repo *gormRepository) UpdatePertanyaan(ctx context.Context, pertanyaan *models.PertanyaanEvaluasi, labelOpsi []string) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// pakai map supaya nilai false (wajib) tetap ikut di-update
		dataBaru := map[string]any{
			"jenis_kuesioner": pertanyaan.JenisKuesioner,
			"teks_pertanyaan": pertanyaan.Teks,
			"tipe_jawaban":    pertanyaan.TipeJawaban,
			"urutan":          pertanyaan.Urutan,
			"wajib":           pertanyaan.Wajib,
		}
		err := tx.Model(&models.PertanyaanEvaluasi{}).Where("id_pertanyaan = ?", pertanyaan.ID).Updates(dataBaru).Error
		if err != nil {
			return err
		}
		return syncOpsi(tx, pertanyaan.ID, labelOpsi)
	})
	return mapErr(err)
}

// syncOpsi hapus sementara semua pilihan lama, lalu pilihan yang masih dipakai dikembalikan
// dengan ID yang sama (supaya jawaban lama tidak hilang), sisanya dibuat baru.
func syncOpsi(tx *gorm.DB, pertanyaanID uint, labelOpsi []string) error {
	if err := tx.Where("id_pertanyaan = ?", pertanyaanID).Delete(&models.OpsiPertanyaan{}).Error; err != nil {
		return err
	}
	for i, label := range labelOpsi {
		urutan := i + 1 // urutan opsi mulai dari 1
		var opsiLama models.OpsiPertanyaan
		// cari juga pilihan yang sudah dihapus (Unscoped); huruf besar/kecil dianggap sama
		err := tx.Unscoped().Where("id_pertanyaan = ? AND label = ?", pertanyaanID, label).Take(&opsiLama).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&models.OpsiPertanyaan{PertanyaanID: pertanyaanID, Label: label, Urutan: urutan}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		dataOpsi := map[string]any{"label": label, "urutan": urutan, "deleted_at": nil} // deleted_at nil = batal dihapus
		err = tx.Unscoped().Model(&opsiLama).Updates(dataOpsi).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// SetAktifPertanyaan ubah status aktif (cek pertanyaannya ada atau tidak sudah dilakukan di service).
func (repo *gormRepository) SetAktifPertanyaan(ctx context.Context, pertanyaanID uint, isAktif bool) error {
	return repo.db.WithContext(ctx).Model(&models.PertanyaanEvaluasi{}).
		Where("id_pertanyaan = ?", pertanyaanID).Update("is_aktif", isAktif).Error
}

// DeletePertanyaan hapus sementara (isi deleted_at), data tetap ada di database.
func (repo *gormRepository) DeletePertanyaan(ctx context.Context, pertanyaanID uint) error {
	hasil := repo.db.WithContext(ctx).Delete(&models.PertanyaanEvaluasi{}, pertanyaanID)
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SimpanIDGoogle simpan ID item & ID pertanyaan Google hasil sinkron (kalau satu gagal, semua dibatalkan).
func (repo *gormRepository) SimpanIDGoogle(ctx context.Context, daftarPertanyaan []models.PertanyaanEvaluasi) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, pertanyaan := range daftarPertanyaan {
			dataGoogle := map[string]any{
				"google_item_id":     pertanyaan.GoogleItemID,
				"google_question_id": pertanyaan.GoogleQuestionID,
			}
			err := tx.Model(&models.PertanyaanEvaluasi{}).Where("id_pertanyaan = ?", pertanyaan.ID).Updates(dataGoogle).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// ImportPertanyaan simpan hasil import Excel dalam 1 transaksi (kalau satu langkah gagal, semua dibatalkan):
// 1. nonaktifkan pertanyaan lama yang tidak ada di Excel
// 2. perbarui urutan, wajib, dan opsi pertanyaan yang dipakai ulang
// 3. buat pertanyaan baru beserta opsinya
func (repo *gormRepository) ImportPertanyaan(ctx context.Context, pertanyaanBaru, pertanyaanDipakaiUlang []models.PertanyaanEvaluasi, idDinonaktifkan []uint) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(idDinonaktifkan) > 0 {
			err := tx.Model(&models.PertanyaanEvaluasi{}).Where("id_pertanyaan IN ?", idDinonaktifkan).Update("is_aktif", false).Error
			if err != nil {
				return err
			}
		}

		for _, pertanyaan := range pertanyaanDipakaiUlang {
			// pakai map supaya nilai false (wajib) tetap ikut di-update
			dataBaru := map[string]any{
				"teks_pertanyaan": pertanyaan.Teks,
				"urutan":          pertanyaan.Urutan,
				"wajib":           pertanyaan.Wajib,
			}
			err := tx.Model(&models.PertanyaanEvaluasi{}).Where("id_pertanyaan = ?", pertanyaan.ID).Updates(dataBaru).Error
			if err != nil {
				return err
			}
			if err := syncOpsi(tx, pertanyaan.ID, labelDariOpsi(pertanyaan.Opsi)); err != nil {
				return err
			}
		}

		if len(pertanyaanBaru) > 0 {
			// opsi di dalam tiap pertanyaan ikut tersimpan otomatis
			if err := tx.Create(&pertanyaanBaru).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return mapErr(err)
}

// labelDariOpsi ambil teks label dari daftar opsi, contoh [{Label: "Iya"}, {Label: "Tidak"}] -> ["Iya", "Tidak"].
func labelDariOpsi(daftarOpsi []models.OpsiPertanyaan) []string {
	daftarLabel := []string{}
	for _, opsi := range daftarOpsi {
		daftarLabel = append(daftarLabel, opsi.Label)
	}
	return daftarLabel
}

// AmbilPengaturan: kunci yang belum pernah disimpan tidak ada di map hasil.
func (repo *gormRepository) AmbilPengaturan(ctx context.Context, daftarKunci ...string) (map[string]string, error) {
	var daftar []models.Pengaturan
	err := repo.db.WithContext(ctx).Where("kunci IN ?", daftarKunci).Find(&daftar).Error
	if err != nil {
		return nil, err
	}
	hasil := map[string]string{}
	for _, pengaturan := range daftar {
		hasil[pengaturan.Kunci] = pengaturan.Nilai
	}
	return hasil, nil
}

// SimpanPengaturan: insert, atau update nilai kalau kuncinya sudah ada.
func (repo *gormRepository) SimpanPengaturan(ctx context.Context, isi map[string]string) error {
	var daftar []models.Pengaturan
	for kunci, nilai := range isi {
		daftar = append(daftar, models.Pengaturan{Kunci: kunci, Nilai: nilai})
	}
	return repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kunci"}},
		DoUpdates: clause.AssignmentColumns([]string{"nilai", "updated_at"}),
	}).Create(&daftar).Error
}

// mapErr ubah error database jadi "tidak ditemukan" (ErrNotFound) atau "data dobel" (ErrDuplicate).
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // 1062 = kode MySQL untuk data dobel
		return ErrDuplicate
	}
	return err
}
