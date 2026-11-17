package group

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

// Repository: daftar query ke database untuk komunitas dan anggotanya.
// Service hanya kenal interface ini, jadi bisa diganti repo palsu saat unit test.
type Repository interface {
	List(ctx context.Context, filter ListQuery) ([]models.Komunitas, int64, error)
	FindByID(ctx context.Context, komunitasID uint) (*models.Komunitas, error)
	FindDeletedByNama(ctx context.Context, nama string) (*models.Komunitas, error)
	Restore(ctx context.Context, komunitas *models.Komunitas) error
	Create(ctx context.Context, komunitas *models.Komunitas) error
	Update(ctx context.Context, komunitas *models.Komunitas) error
	Delete(ctx context.Context, komunitasID uint) error
	UserExists(ctx context.Context, userID uint) (bool, error)
	ListAnggota(ctx context.Context, komunitasID uint) ([]models.UserKomunitas, error)
	AddAnggota(ctx context.Context, anggota *models.UserKomunitas) error
	SetKetua(ctx context.Context, komunitasID, userID uint, isKetua bool) error
	RemoveAnggota(ctx context.Context, komunitasID, userID uint) error
}

// gormRepository: implementasi Repository memakai GORM (MySQL).
type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

// List mengambil komunitas per halaman + total data (untuk meta pagination).
func (repo *gormRepository) List(ctx context.Context, filter ListQuery) ([]models.Komunitas, int64, error) {
	query := repo.db.WithContext(ctx).Model(&models.Komunitas{})
	if filter.Q != "" { // pencarian ?q= berdasarkan nama
		query = query.Where("nama_komunitas LIKE ?", "%"+escapeLike(filter.Q)+"%")
	}
	query = query.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var daftarKomunitas []models.Komunitas
	err := query.
		Order(filter.Order([]string{"nama_komunitas", "created_at"}, "nama_komunitas ASC")). // hanya kolom ini yang boleh di ?sort=
		Order("id_komunitas ASC").                                                            // nilai sama -> urut ID, supaya halaman stabil
		Offset(filter.Offset).Limit(filter.Limit).
		Find(&daftarKomunitas).Error
	return daftarKomunitas, total, err
}

// FindByID mengambil satu komunitas; ErrNotFound jika tidak ada atau sudah dihapus.
func (repo *gormRepository) FindByID(ctx context.Context, komunitasID uint) (*models.Komunitas, error) {
	var komunitas models.Komunitas
	if err := repo.db.WithContext(ctx).First(&komunitas, komunitasID).Error; err != nil {
		return nil, mapErr(err)
	}
	return &komunitas, nil
}

// FindDeletedByNama cari komunitas yang sudah dihapus (deleted_at terisi) dengan nama yang sama.
// Unscoped supaya data yang sudah dihapus ikut dicari.
func (repo *gormRepository) FindDeletedByNama(ctx context.Context, nama string) (*models.Komunitas, error) {
	var komunitas models.Komunitas
	err := repo.db.WithContext(ctx).Unscoped().
		Where("nama_komunitas = ? AND deleted_at IS NOT NULL", nama).
		First(&komunitas).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &komunitas, nil
}

// Restore memulihkan komunitas yang sudah dihapus (deleted_at = NULL) sekaligus memperbarui nama dan deskripsinya.
func (repo *gormRepository) Restore(ctx context.Context, komunitas *models.Komunitas) error {
	dataBaru := map[string]any{
		"nama_komunitas": komunitas.Nama,
		"deskripsi":      komunitas.Deskripsi,
		"deleted_at":     nil,
	}
	return repo.db.WithContext(ctx).Unscoped().Model(&models.Komunitas{}).
		Where("id_komunitas = ?", komunitas.ID).Updates(dataBaru).Error
}

// Create menyimpan komunitas baru; ID hasil insert otomatis terisi ke komunitas.ID.
func (repo *gormRepository) Create(ctx context.Context, komunitas *models.Komunitas) error {
	return mapErr(repo.db.WithContext(ctx).Create(komunitas).Error)
}

// Update hanya mengubah kolom nama dan deskripsi.
func (repo *gormRepository) Update(ctx context.Context, komunitas *models.Komunitas) error {
	return mapErr(repo.db.WithContext(ctx).Model(komunitas).Select("nama_komunitas", "deskripsi").Updates(komunitas).Error)
}

// Delete = soft delete (isi deleted_at), data lama tetap ada di database.
func (repo *gormRepository) Delete(ctx context.Context, komunitasID uint) error {
	hasil := repo.db.WithContext(ctx).Delete(&models.Komunitas{}, komunitasID)
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 { // tidak ada baris yang terhapus = ID tidak ada
		return ErrNotFound
	}
	return nil
}

// UserExists mengecek user ada dan belum dihapus.
func (repo *gormRepository) UserExists(ctx context.Context, userID uint) (bool, error) {
	var jumlah int64
	err := repo.db.WithContext(ctx).Model(&models.User{}).Where("id_user = ?", userID).Count(&jumlah).Error
	return jumlah > 0, err
}

// ListAnggota mengambil anggota beserta data user-nya, ketua di urutan atas.
// INNER JOIN supaya user yang sudah dihapus tidak ikut.
func (repo *gormRepository) ListAnggota(ctx context.Context, komunitasID uint) ([]models.UserKomunitas, error) {
	var daftarAnggota []models.UserKomunitas
	err := repo.db.WithContext(ctx).
		InnerJoins("User").
		Where("user_komunitas.id_komunitas = ?", komunitasID).
		Order("user_komunitas.is_ketua DESC, User.nama ASC").
		Find(&daftarAnggota).Error
	return daftarAnggota, err
}

// AddAnggota memasukkan user ke komunitas; ErrDuplicate jika sudah jadi anggota.
func (repo *gormRepository) AddAnggota(ctx context.Context, anggota *models.UserKomunitas) error {
	return mapErr(repo.db.WithContext(ctx).Omit("User").Create(anggota).Error) // Omit: abaikan field User, jangan ikut menulis tabel user
}

// SetKetua mengubah status ketua; ErrNotFound jika user bukan anggota komunitas ini.
func (repo *gormRepository) SetKetua(ctx context.Context, komunitasID, userID uint, isKetua bool) error {
	query := repo.db.WithContext(ctx).Model(&models.UserKomunitas{}).
		Where("id_komunitas = ? AND id_user = ?", komunitasID, userID).
		Session(&gorm.Session{})
	var jumlah int64 // jumlah baris keanggotaan yang cocok (0 atau 1)
	if err := query.Count(&jumlah).Error; err != nil {
		return err
	}
	if jumlah == 0 {
		return ErrNotFound
	}
	return query.Update("is_ketua", isKetua).Error
}

// RemoveAnggota menghapus permanen (Unscoped) supaya user bisa ditambahkan lagi
// tanpa bentrok unique key (id_user, id_komunitas).
func (repo *gormRepository) RemoveAnggota(ctx context.Context, komunitasID, userID uint) error {
	hasil := repo.db.WithContext(ctx).Unscoped().
		Where("id_komunitas = ? AND id_user = ?", komunitasID, userID).
		Delete(&models.UserKomunitas{})
	if hasil.Error != nil {
		return hasil.Error
	}
	if hasil.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
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
