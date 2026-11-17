package schedule

import (
	"context"
	"errors"

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

// Repository: daftar fungsi query database yang dibutuhkan service kegiatan.
type Repository interface {
	ListKegiatan(ctx context.Context, filter KegiatanQuery) ([]models.Kegiatan, int64, error)
	FindKegiatan(ctx context.Context, kegiatanID uint) (*models.Kegiatan, error)
	CreateKegiatan(ctx context.Context, kegiatan *models.Kegiatan) error
	UpdateKegiatan(ctx context.Context, kegiatan *models.Kegiatan) error
	DeleteKegiatan(ctx context.Context, kegiatanID uint) error
	KomunitasAda(ctx context.Context, komunitasID uint) (bool, error)

	// Penugasan petugas (tabel jadwal_pelayanan).
	FindUser(ctx context.Context, userID uint) (*models.User, error)
	FindKeahlian(ctx context.Context, keahlianID uint) (*models.Keahlian, error)
	// UserPunyaKeahlian: true kalau jemaat punya keahlian itu dan sudah diverifikasi admin.
	UserPunyaKeahlian(ctx context.Context, userID, keahlianID uint) (bool, error)
	CreatePenugasan(ctx context.Context, penugasan *models.JadwalPelayanan) error
	FindPenugasan(ctx context.Context, kegiatanID, penjadwalanID uint) (*models.JadwalPelayanan, error)
	ListPetugasKegiatan(ctx context.Context, kegiatanID uint) ([]models.JadwalPelayanan, error)
	DeletePenugasan(ctx context.Context, kegiatanID, penjadwalanID uint) error
	// ListPenugasanUserPadaTanggal: tugas seorang jemaat di satu tanggal (untuk cek bentrok waktu).
	ListPenugasanUserPadaTanggal(ctx context.Context, userID uint, tanggal string, kecualiKegiatanID uint) ([]models.JadwalPelayanan, error)
	ListJadwalUser(ctx context.Context, userID uint, filter JadwalSayaQuery) ([]models.JadwalPelayanan, int64, error)

	// import Excel kegiatan (ImportSchedules)
	FindKomunitasByNama(ctx context.Context, nama string) (*models.Komunitas, error)
	ImportKegiatan(ctx context.Context, kegiatanBaru []models.Kegiatan, kegiatanDiperbarui []models.Kegiatan) error
	FindKegiatanByUnik(ctx context.Context, nama, tanggal, waktuMulai string) (*models.Kegiatan, error)
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// ListKegiatan ambil kegiatan per halaman. Filter tanggal dan nama boleh kosong.
func (repo *gormRepository) ListKegiatan(ctx context.Context, filter KegiatanQuery) ([]models.Kegiatan, int64, error) {
	query := repo.db.WithContext(ctx).Model(&models.Kegiatan{})
	if filter.Dari != "" {
		query = query.Where("tanggal >= ?", filter.Dari)
	}
	if filter.Sampai != "" {
		query = query.Where("tanggal <= ?", filter.Sampai)
	}
	if filter.Q != "" {
		query = query.Where("nama_kegiatan LIKE ?", "%"+filter.Q+"%")
	}
	query = query.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var daftarKegiatan []models.Kegiatan
	err := query.Preload("Komunitas").
		Order(filter.Order([]string{"tanggal", "nama_kegiatan", "created_at"}, "tanggal ASC")). // hanya kolom ini yang boleh di ?sort=
		Order("waktu_mulai ASC").                                                               // tanggal sama -> urut jam mulai
		Order("id_kegiatan ASC").                                                               // supaya urutan halaman stabil
		Offset(filter.Offset).Limit(filter.Limit).
		Find(&daftarKegiatan).Error
	return daftarKegiatan, total, err
}

// FindKegiatan ambil satu kegiatan beserta komunitasnya, ErrNotFound kalau tidak ada.
func (repo *gormRepository) FindKegiatan(ctx context.Context, kegiatanID uint) (*models.Kegiatan, error) {
	var kegiatan models.Kegiatan
	err := repo.db.WithContext(ctx).Preload("Komunitas").First(&kegiatan, kegiatanID).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &kegiatan, nil
}

// CreateKegiatan simpan kegiatan baru. Kalau kegiatan yang sama (nama + tanggal + jam mulai)
// pernah dihapus, data lama itu dipulihkan dan ditimpa, karena UNIQUE KEY tetap menghitung data yang dihapus.
func (repo *gormRepository) CreateKegiatan(ctx context.Context, kegiatan *models.Kegiatan) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var kegiatanLama models.Kegiatan
		err := tx.Unscoped().
			Where("nama_kegiatan = ? AND tanggal = ? AND waktu_mulai = ?", kegiatan.Nama, kegiatan.Tanggal.Format("2006-01-02"), kegiatan.WaktuMulai).
			Where("deleted_at IS NOT NULL").
			Take(&kegiatanLama).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(kegiatan).Error // tidak ada data lama -> buat baru
		}
		if err != nil {
			return err
		}

		kegiatan.ID = kegiatanLama.ID
		dataBaru := dataKegiatan(kegiatan)
		dataBaru["deleted_at"] = nil // nil = batal dihapus
		return tx.Unscoped().Model(&models.Kegiatan{}).Where("id_kegiatan = ?", kegiatan.ID).Updates(dataBaru).Error
	})
	return mapErr(err)
}

// UpdateKegiatan timpa semua isi kegiatan (cek kegiatannya ada sudah dilakukan di service).
func (repo *gormRepository) UpdateKegiatan(ctx context.Context, kegiatan *models.Kegiatan) error {
	err := repo.db.WithContext(ctx).Model(&models.Kegiatan{}).
		Where("id_kegiatan = ?", kegiatan.ID).Updates(dataKegiatan(kegiatan)).Error
	return mapErr(err)
}

// dataKegiatan ubah kegiatan jadi map kolom -> nilai.
// Pakai map supaya nilai kosong (nil / false) tetap ikut di-update, tidak dilewati GORM.
func dataKegiatan(kegiatan *models.Kegiatan) map[string]any {
	return map[string]any{
		"nama_kegiatan":  kegiatan.Nama,
		"tanggal":        kegiatan.Tanggal.Format("2006-01-02"),
		"waktu_mulai":    kegiatan.WaktuMulai,
		"waktu_selesai":  kegiatan.WaktuSelesai,
		"tema":           kegiatan.Tema,
		"nats_alkitab":   kegiatan.NatsAlkitab,
		"pembicara":      kegiatan.Pembicara,
		"lokasi":         kegiatan.Lokasi,
		"deskripsi":      kegiatan.Deskripsi,
		"id_komunitas":   kegiatan.KomunitasID,
		"perlu_evaluasi": kegiatan.PerluEvaluasi,
	}
}

// DeleteKegiatan hapus sementara (isi deleted_at), data absensi dan evaluasi lama tetap aman.
func (repo *gormRepository) DeleteKegiatan(ctx context.Context, kegiatanID uint) error {
	hasil := repo.db.WithContext(ctx).Delete(&models.Kegiatan{}, kegiatanID)
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// KomunitasAda: true kalau komunitas dengan ID ini ada (dan belum dihapus).
func (repo *gormRepository) KomunitasAda(ctx context.Context, komunitasID uint) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Model(&models.Komunitas{}).Where("id_komunitas = ?", komunitasID).Count(&jumlah).Error
	return jumlah > 0, err
}

// FindUser ambil jemaat beserta role-nya, ErrNotFound kalau tidak ada.
func (repo *gormRepository) FindUser(ctx context.Context, userID uint) (*models.User, error) {
	var user models.User
	if err := repo.db.WithContext(ctx).Preload("Role").First(&user, userID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &user, nil
}

// FindKeahlian ambil satu keahlian, ErrNotFound kalau tidak ada.
func (repo *gormRepository) FindKeahlian(ctx context.Context, keahlianID uint) (*models.Keahlian, error) {
	var keahlian models.Keahlian
	if err := repo.db.WithContext(ctx).First(&keahlian, keahlianID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &keahlian, nil
}

// UserPunyaKeahlian: true kalau jemaat punya keahlian ini (dari admin atau pelatihan).
func (repo *gormRepository) UserPunyaKeahlian(ctx context.Context, userID, keahlianID uint) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Model(&models.UserKeahlian{}).
		Where("id_user = ? AND id_keahlian = ?", userID, keahlianID).
		Count(&jumlah).Error
	return jumlah > 0, err
}

// CreatePenugasan simpan penugasan baru. Kalau penugasan yang sama (kegiatan + user + keahlian)
// pernah dibatalkan, data lama itu dipulihkan, karena UNIQUE KEY tetap menghitung data yang dihapus.
// Kalau masih aktif -> ErrDuplicate.
func (repo *gormRepository) CreatePenugasan(ctx context.Context, penugasan *models.JadwalPelayanan) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lama models.JadwalPelayanan
		err := tx.Unscoped().
			Where("id_kegiatan = ? AND id_user = ? AND id_keahlian = ?", penugasan.KegiatanID, penugasan.UserID, penugasan.KeahlianID).
			Where("deleted_at IS NOT NULL").
			Take(&lama).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Omit(clause.Associations).Create(penugasan).Error // tidak ada data lama -> buat baru
		}
		if err != nil {
			return err
		}

		penugasan.ID = lama.ID
		return tx.Unscoped().Model(&models.JadwalPelayanan{}).Where("id_penjadwalan = ?", penugasan.ID).
			Updates(map[string]any{"status_konfirmasi": penugasan.StatusKonfirmasi, "deleted_at": nil}).Error // nil = batal dihapus
	})
	return mapErr(err)
}

// FindPenugasan ambil satu penugasan beserta jemaat dan keahliannya.
// kegiatanID ikut dicek supaya /kegiatan/1/petugas/9 tidak bisa membuka penugasan milik kegiatan lain.
func (repo *gormRepository) FindPenugasan(ctx context.Context, kegiatanID, penjadwalanID uint) (*models.JadwalPelayanan, error) {
	var penugasan models.JadwalPelayanan
	err := repo.db.WithContext(ctx).Preload("User").Preload("Keahlian").
		Where("id_kegiatan = ? AND id_penjadwalan = ?", kegiatanID, penjadwalanID).
		First(&penugasan).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &penugasan, nil
}

// ListPetugasKegiatan ambil semua petugas satu kegiatan.
func (repo *gormRepository) ListPetugasKegiatan(ctx context.Context, kegiatanID uint) ([]models.JadwalPelayanan, error) {
	var daftar []models.JadwalPelayanan
	err := repo.db.WithContext(ctx).Preload("User").Preload("Keahlian").
		Where("id_kegiatan = ?", kegiatanID).
		Order("id_penjadwalan ASC").
		Find(&daftar).Error
	return daftar, err
}

// DeletePenugasan batalkan penugasan (isi deleted_at).
func (repo *gormRepository) DeletePenugasan(ctx context.Context, kegiatanID, penjadwalanID uint) error {
	hasil := repo.db.WithContext(ctx).
		Where("id_kegiatan = ? AND id_penjadwalan = ?", kegiatanID, penjadwalanID).
		Delete(&models.JadwalPelayanan{})
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListPenugasanUserPadaTanggal ambil tugas seorang jemaat di satu tanggal beserta kegiatannya.
// Penugasan yang DITOLAK tidak dihitung karena jemaat tidak jadi bertugas.
func (repo *gormRepository) ListPenugasanUserPadaTanggal(ctx context.Context, userID uint, tanggal string, kecualiKegiatanID uint) ([]models.JadwalPelayanan, error) {
	var daftar []models.JadwalPelayanan
	err := repo.db.WithContext(ctx).
		Joins("JOIN kegiatan ON kegiatan.id_kegiatan = jadwal_pelayanan.id_kegiatan AND kegiatan.deleted_at IS NULL").
		Where("jadwal_pelayanan.id_user = ? AND kegiatan.tanggal = ? AND jadwal_pelayanan.id_kegiatan <> ?", userID, tanggal, kecualiKegiatanID).
		Where("jadwal_pelayanan.status_konfirmasi <> ?", models.PenugasanDitolak).
		Preload("Kegiatan").
		Find(&daftar).Error
	return daftar, err
}

// ListJadwalUser ambil tugas pelayanan seorang jemaat per halaman. Filter tanggal boleh kosong.
func (repo *gormRepository) ListJadwalUser(ctx context.Context, userID uint, filter JadwalSayaQuery) ([]models.JadwalPelayanan, int64, error) {
	query := repo.db.WithContext(ctx).Model(&models.JadwalPelayanan{}).
		Joins("JOIN kegiatan ON kegiatan.id_kegiatan = jadwal_pelayanan.id_kegiatan AND kegiatan.deleted_at IS NULL").
		Where("jadwal_pelayanan.id_user = ?", userID)
	if filter.Dari != "" {
		query = query.Where("kegiatan.tanggal >= ?", filter.Dari)
	}
	if filter.Sampai != "" {
		query = query.Where("kegiatan.tanggal <= ?", filter.Sampai)
	}
	query = query.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var daftar []models.JadwalPelayanan
	err := query.Preload("Kegiatan").Preload("Keahlian").
		Order(filter.Order([]string{"tanggal"}, "tanggal ASC")). // kolom tanggal hanya ada di tabel kegiatan
		Order("kegiatan.waktu_mulai ASC").
		Order("jadwal_pelayanan.id_penjadwalan ASC"). // supaya urutan halaman stabil
		Offset(filter.Offset).Limit(filter.Limit).
		Find(&daftar).Error
	return daftar, total, err
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

// FindKomunitasByNama: komunitas dari nama di kolom Excel, ErrNotFound kalau tidak ada.
func (repo *gormRepository) FindKomunitasByNama(ctx context.Context, nama string) (*models.Komunitas, error) {
	var komunitas models.Komunitas
	err := repo.db.WithContext(ctx).Where("nama_komunitas = ?", nama).First(&komunitas).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &komunitas, nil
}

// ImportKegiatan menyimpan hasil import dalam 1 transaksi. ID kegiatan baru terisi setelah disimpan.
func (repo *gormRepository) ImportKegiatan(ctx context.Context, kegiatanBaru []models.Kegiatan, kegiatanDiperbarui []models.Kegiatan) error {
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update yang sudah ada
		for _, k := range kegiatanDiperbarui {
			dataBaru := dataKegiatan(&k)
			dataBaru["deleted_at"] = nil // Batalkan soft-delete jika ada
			if err := tx.Unscoped().Model(&models.Kegiatan{}).Where("id_kegiatan = ?", k.ID).Updates(dataBaru).Error; err != nil {
				return err
			}
		}
		// Buat yang baru
		if len(kegiatanBaru) > 0 {
			if err := tx.Create(&kegiatanBaru).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return mapErr(err)
}

// FindKegiatanByUnik: kegiatan dengan nama + tanggal + jam mulai yang sama (termasuk yang sudah dihapus).
func (repo *gormRepository) FindKegiatanByUnik(ctx context.Context, nama, tanggal, waktuMulai string) (*models.Kegiatan, error) {
	var k models.Kegiatan
	err := repo.db.WithContext(ctx).Unscoped().
		Where("nama_kegiatan = ? AND tanggal = ? AND waktu_mulai = ?", nama, tanggal, waktuMulai).
		First(&k).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &k, nil
}
