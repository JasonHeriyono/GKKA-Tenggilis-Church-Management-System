package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"sim-g-backend/models"
)

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data duplikat")
)

type Repository interface {
	List(ctx context.Context, q ListQuery) ([]models.User, int64, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindRoleByName(ctx context.Context, name string) (*models.Role, error)
	Create(ctx context.Context, u *models.User) error
	// Update mengganti data utama user dan seluruh identitasnya (satu transaksi).
	Update(ctx context.Context, u *models.User, data models.IdentitasUser) error
	// Delete: hapus sementara, email/no HP dikosongkan supaya bisa dipakai orang lain, semua sesi dicabut,
	// dan pengajuan verifikasi yang masih MENUNGGU otomatis ditolak (satu transaksi).
	Delete(ctx context.Context, id, adminID uint) error
	UpdateRole(ctx context.Context, id, roleID uint) error
	// SetActive menonaktifkan/mengaktifkan akun. Saat dinonaktifkan, semua refresh token dicabut (satu transaksi).
	SetActive(ctx context.Context, id uint, aktif bool) error
	// SetPassword mengganti password dan mencabut semua refresh token user (satu transaksi).
	SetPassword(ctx context.Context, id uint, hash string) error
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) List(ctx context.Context, q ListQuery) ([]models.User, int64, error) {
	base := r.db.WithContext(ctx).Model(&models.User{})
	if q.Q != "" {
		like := "%" + escapeLike(q.Q) + "%"
		base = base.Where("nama LIKE ? OR kode_jemaat LIKE ? OR email LIKE ? OR no_telepon LIKE ?", like, like, like, like)
	}
	if q.Role != "" {
		base = base.Where("id_role = (SELECT id_role FROM role WHERE nama_role = ? AND deleted_at IS NULL)", q.Role)
	}
	if q.StatusVerifikasi != "" {
		base = base.Where("status_verifikasi = ?", q.StatusVerifikasi)
	}
	base = base.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []models.User
	err := base.Preload("Role").Preload("Identitas").
		Order(q.Order([]string{"nama", "kode_jemaat", "created_at"}, "nama ASC")).
		Order("id_user ASC"). // nama sama -> urut ID, supaya data tidak dobel/hilang antar halaman
		Offset(q.Offset).Limit(q.Limit).
		Find(&users).Error
	return users, total, err
}

func (r *gormRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).Preload("Role").Preload("Identitas").First(&u, id).Error; err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *gormRepository) FindRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Where("nama_role = ?", name).First(&role).Error; err != nil {
		return nil, mapErr(err)
	}
	return &role, nil
}

func (r *gormRepository) Create(ctx context.Context, u *models.User) error {
	// Omit("Role"): jangan ikut menulis baris role. Identitas (has-one) ikut tersimpan otomatis.
	return mapErr(r.db.WithContext(ctx).Omit("Role").Create(u).Error)
}

func (r *gormRepository) Update(ctx context.Context, u *models.User, data models.IdentitasUser) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// pakai map supaya nilai kosong (nil) ikut tersimpan
		dataUser := map[string]any{
			"kode_jemaat": u.KodeJemaat,
			"nama":        u.Nama,
			"email":       u.Email,
			"no_telepon":  u.NoTelepon,
		}
		if err := tx.Model(&models.User{}).Where("id_user = ?", u.ID).Updates(dataUser).Error; err != nil {
			return err
		}
		return simpanIdentitas(tx, data)
	})
	return mapErr(err)
}

// simpanIdentitas: buat baris baru kalau belum ada, kalau sudah ada timpa semua kolomnya.
func simpanIdentitas(tx *gorm.DB, data models.IdentitasUser) error {
	var jumlah int64
	if err := tx.Model(&models.IdentitasUser{}).Where("id_user = ?", data.UserID).Count(&jumlah).Error; err != nil {
		return err
	}
	if jumlah == 0 {
		return tx.Create(&data).Error
	}
	// Select("*") supaya kolom yang dikosongkan admin ikut tersimpan kosong
	return tx.Model(&models.IdentitasUser{}).Where("id_user = ?", data.UserID).
		Select("*").Omit("id_user", "created_at", "deleted_at").Updates(&data).Error
}

func (r *gormRepository) Delete(ctx context.Context, id, adminID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dataKosong := map[string]any{"email": nil, "no_telepon": nil, "is_aktif": false}
		if err := updateColumns(tx, id, dataKosong); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&models.User{}, id).Error; err != nil {
			return err
		}
		// pengajuan verifikasi yang belum diperiksa tidak boleh tertinggal di antrean
		dataTolak := map[string]any{
			"status":         models.PendaftaranDitolak,
			"diperiksa_oleh": adminID,
			"diperiksa_pada": time.Now(),
			"catatan_admin":  "Akun dihapus admin",
		}
		err := tx.Model(&models.Pendaftaran{}).
			Where("id_user = ? AND status = ?", id, models.PendaftaranMenunggu).
			Updates(dataTolak).Error
		if err != nil {
			return err
		}
		return revokeAll(tx, id)
	})
}

func (r *gormRepository) UpdateRole(ctx context.Context, id, roleID uint) error {
	return updateColumns(r.db.WithContext(ctx), id, map[string]any{"id_role": roleID})
}

func (r *gormRepository) SetActive(ctx context.Context, id uint, aktif bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateColumns(tx, id, map[string]any{"is_aktif": aktif}); err != nil {
			return err
		}
		if aktif {
			return nil
		}
		return revokeAll(tx, id)
	})
}

func (r *gormRepository) SetPassword(ctx context.Context, id uint, hash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := updateColumns(tx, id, map[string]any{"password_hash": hash}); err != nil {
			return err
		}
		return revokeAll(tx, id)
	})
}

func updateColumns(tx *gorm.DB, id uint, values map[string]any) error {
	res := tx.Model(&models.User{}).Where("id_user = ?", id).Updates(values)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// bisa juga nilainya sama dengan sebelumnya (MySQL tidak menghitungnya); cek keberadaan user
		var n int64
		if err := tx.Model(&models.User{}).Where("id_user = ?", id).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func revokeAll(tx *gorm.DB, userID uint) error {
	return tx.Model(&models.RefreshToken{}).
		Where("id_user = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == 1062 { // duplicate entry
		return ErrDuplicate
	}
	return err
}

func escapeLike(s string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(s)
}
