package skill

import (
	"context"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"sim-g-backend/models"
)

// Error umum dari repository; service mengubahnya jadi pesan HTTP (404 / 409).
var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data duplikat")
)

// Repository: daftar query ke database untuk keahlian dan pemiliknya.
// Service hanya kenal interface ini, jadi bisa diganti repo palsu saat unit test.
type Repository interface {
	List(ctx context.Context, filter ListQuery) ([]models.Keahlian, int64, error)
	FindByID(ctx context.Context, keahlianID uint) (*models.Keahlian, error)
	FindDeletedByNama(ctx context.Context, nama string) (*models.Keahlian, error)
	Restore(ctx context.Context, keahlian *models.Keahlian) error
	Create(ctx context.Context, keahlian *models.Keahlian) error
	Update(ctx context.Context, keahlian *models.Keahlian) error
	Delete(ctx context.Context, keahlianID uint) error
	// InUseBySchedule: keahlian masih dipakai di jadwal pelayanan yang aktif.
	InUseBySchedule(ctx context.Context, keahlianID uint) (bool, error)

	UserExists(ctx context.Context, userID uint) (bool, error)
	ListJemaat(ctx context.Context, keahlianID uint) ([]models.UserKeahlian, error)
	AddJemaat(ctx context.Context, userKeahlian *models.UserKeahlian) error
	RemoveJemaat(ctx context.Context, keahlianID, userID uint) error

	ListByUser(ctx context.Context, userID uint) ([]models.Keahlian, error)
	// CountExisting: berapa dari ids yang benar-benar ada (belum dihapus).
	CountExisting(ctx context.Context, ids []uint) (int64, error)
	// ReplaceUserKeahlian menjadikan ids sebagai daftar keahlian user, dalam satu transaksi.
	ReplaceUserKeahlian(ctx context.Context, userID uint, ids []uint) error
}

// gormRepository: implementasi Repository memakai GORM (MySQL).
type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

// List mengambil keahlian per halaman + total data (untuk meta pagination).
func (repo *gormRepository) List(ctx context.Context, filter ListQuery) ([]models.Keahlian, int64, error) {
	query := repo.db.WithContext(ctx).Model(&models.Keahlian{})
	if filter.Q != "" { // pencarian ?q= berdasarkan nama
		query = query.Where("nama_keahlian LIKE ?", "%"+escapeLike(filter.Q)+"%")
	}
	query = query.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var daftarKeahlian []models.Keahlian
	err := query.
		Order(filter.Order([]string{"nama_keahlian", "created_at"}, "nama_keahlian ASC")). // hanya kolom ini yang boleh di ?sort=
		Order("id_keahlian ASC").                                                          // nilai sama -> urut ID, supaya halaman stabil
		Offset(filter.Offset).Limit(filter.Limit).
		Find(&daftarKeahlian).Error
	return daftarKeahlian, total, err
}

// FindByID mengambil satu keahlian; ErrNotFound jika tidak ada atau sudah dihapus.
func (repo *gormRepository) FindByID(ctx context.Context, keahlianID uint) (*models.Keahlian, error) {
	var keahlian models.Keahlian
	if err := repo.db.WithContext(ctx).First(&keahlian, keahlianID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &keahlian, nil
}

// FindDeletedByNama cari keahlian yang sudah dihapus dengan nama yang sama (Unscoped = ikut cari yang terhapus).
func (repo *gormRepository) FindDeletedByNama(ctx context.Context, nama string) (*models.Keahlian, error) {
	var keahlian models.Keahlian
	err := repo.db.WithContext(ctx).Unscoped().
		Where("nama_keahlian = ? AND deleted_at IS NOT NULL", nama).
		First(&keahlian).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &keahlian, nil
}

// Restore memulihkan keahlian yang sudah dihapus (deleted_at = NULL) sekaligus memperbarui namanya.
func (repo *gormRepository) Restore(ctx context.Context, keahlian *models.Keahlian) error {
	dataBaru := map[string]any{"nama_keahlian": keahlian.Nama, "deleted_at": nil}
	return repo.db.WithContext(ctx).Unscoped().Model(&models.Keahlian{}).
		Where("id_keahlian = ?", keahlian.ID).Updates(dataBaru).Error
}

// Create menyimpan keahlian baru; ID hasil insert otomatis terisi ke keahlian.ID.
func (repo *gormRepository) Create(ctx context.Context, keahlian *models.Keahlian) error {
	return mapErr(repo.db.WithContext(ctx).Create(keahlian).Error)
}

// Update hanya mengubah kolom nama (is_sistem tidak pernah ikut berubah).
func (repo *gormRepository) Update(ctx context.Context, keahlian *models.Keahlian) error {
	return mapErr(repo.db.WithContext(ctx).Model(keahlian).Select("nama_keahlian").Updates(keahlian).Error)
}

// Delete = soft delete (isi deleted_at), data lama tetap ada di database.
func (repo *gormRepository) Delete(ctx context.Context, keahlianID uint) error {
	hasil := repo.db.WithContext(ctx).Delete(&models.Keahlian{}, keahlianID)
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 { // tidak ada baris yang terhapus = ID tidak ada
		return ErrNotFound
	}
	return nil
}

func (repo *gormRepository) InUseBySchedule(ctx context.Context, keahlianID uint) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Table("jadwal_pelayanan").
		Where("id_keahlian = ? AND deleted_at IS NULL", keahlianID).
		Count(&jumlah).Error
	return jumlah > 0, err
}

// UserExists mengecek user ada dan belum dihapus.
func (repo *gormRepository) UserExists(ctx context.Context, userID uint) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Model(&models.User{}).Where("id_user = ?", userID).Count(&jumlah).Error
	return jumlah > 0, err
}

// ListJemaat mengambil jemaat pemilik keahlian beserta data user-nya, urut nama.
// INNER JOIN supaya user yang sudah dihapus tidak ikut.
func (repo *gormRepository) ListJemaat(ctx context.Context, keahlianID uint) ([]models.UserKeahlian, error) {
	var daftar []models.UserKeahlian
	err := repo.db.WithContext(ctx).
		InnerJoins("User").
		Where("user_keahlian.id_keahlian = ?", keahlianID).
		Order("User.nama ASC").
		Find(&daftar).Error
	return daftar, err
}

// AddJemaat memberi keahlian ke user; ErrDuplicate jika user sudah punya keahlian itu.
func (repo *gormRepository) AddJemaat(ctx context.Context, userKeahlian *models.UserKeahlian) error {
	return mapErr(repo.db.WithContext(ctx).Omit("User").Create(userKeahlian).Error) // Omit: jangan ikut menulis tabel user
}

// RemoveJemaat menghapus permanen (Unscoped) supaya keahlian bisa diberikan lagi
// tanpa bentrok unique key (id_user, id_keahlian).
func (repo *gormRepository) RemoveJemaat(ctx context.Context, keahlianID, userID uint) error {
	hasil := repo.db.WithContext(ctx).Unscoped().
		Where("id_keahlian = ? AND id_user = ?", keahlianID, userID).
		Delete(&models.UserKeahlian{})
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListByUser mengambil keahlian milik satu user (keahlian yang sudah dihapus tidak ikut), urut nama.
func (repo *gormRepository) ListByUser(ctx context.Context, userID uint) ([]models.Keahlian, error) {
	idKeahlianUser := repo.db.WithContext(ctx).Model(&models.UserKeahlian{}).
		Select("id_keahlian").Where("id_user = ?", userID)

	var daftar []models.Keahlian
	err := repo.db.WithContext(ctx).
		Where("id_keahlian IN (?)", idKeahlianUser).
		Order("nama_keahlian ASC").
		Find(&daftar).Error
	return daftar, err
}

func (repo *gormRepository) CountExisting(ctx context.Context, ids []uint) (int64, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Model(&models.Keahlian{}).Where("id_keahlian IN ?", ids).Count(&jumlah).Error
	return jumlah, err
}

// ReplaceUserKeahlian: lepas keahlian yang tidak ada di ids, tambahkan yang belum ada.
// Semua dalam satu transaksi, jadi tidak pernah setengah jalan.
func (repo *gormRepository) ReplaceUserKeahlian(ctx context.Context, userID uint, ids []uint) error {
	return repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lepas := tx.Unscoped().Where("id_user = ?", userID)
		if len(ids) > 0 {
			lepas = lepas.Where("id_keahlian NOT IN ?", ids)
		}
		if err := lepas.Delete(&models.UserKeahlian{}).Error; err != nil {
			return err
		}
		for _, keahlianID := range ids {
			baris := models.UserKeahlian{UserID: userID, KeahlianID: keahlianID}
			err := tx.Where("id_user = ? AND id_keahlian = ?", userID, keahlianID).FirstOrCreate(&baris).Error
			if err != nil {
				return mapErr(err)
			}
		}
		return nil
	})
}

// mapErr menerjemahkan error GORM/MySQL ke ErrNotFound / ErrDuplicate.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 { // duplicate entry
		return ErrDuplicate
	}
	return err
}

// escapeLike supaya karakter % dan _ dari user dicari apa adanya, bukan dianggap wildcard.
func escapeLike(teks string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(teks)
}
