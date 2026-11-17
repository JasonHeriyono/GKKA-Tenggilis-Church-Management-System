package pendaftaran

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
	// ErrStateChanged: data sudah diubah pihak lain sejak dibaca (mis. pengajuan sudah diproses admin lain).
	ErrStateChanged = errors.New("data sudah berubah")
)

// Repository: query database untuk signup dan verifikasi. Service hanya kenal interface ini.
type Repository interface {
	// --- user ---
	FindRoleByName(ctx context.Context, name string) (*models.Role, error)
	// KontakDipakai: email / no. telepon sudah dipakai user lain (termasuk yang dihapus, karena UNIQUE tetap berlaku).
	// kecualiUserID = user yang sedang mengedit dirinya sendiri (0 = tidak ada).
	KontakDipakai(ctx context.Context, email, noTelepon string, kecualiUserID uint) (emailDipakai, noTeleponDipakai bool, err error)
	CreateUser(ctx context.Context, u *models.User) error // Identitas ikut tersimpan
	// HapusUserPermanen: batalkan akun yang baru dibuat saat signup gagal di tengah jalan.
	HapusUserPermanen(ctx context.Context, userID uint) error
	FindUserByID(ctx context.Context, id uint) (*models.User, error)
	UpdateProfil(ctx context.Context, u *models.User, data models.IdentitasUser) error
	// UbahStatusUser hanya berhasil kalau status sekarang = dari. false = status sudah berubah.
	UbahStatusUser(ctx context.Context, userID uint, dari, ke string) (bool, error)
	// SetujuiUser: status jadi TERVERIFIKASI dan kode jemaat diisi.
	SetujuiUser(ctx context.Context, userID uint, kodeJemaat string) error

	// --- pendaftaran (pengajuan verifikasi) ---
	CreatePendaftaran(ctx context.Context, p *models.Pendaftaran) error
	FindPendaftaran(ctx context.Context, id uint) (*models.Pendaftaran, error)
	PendaftaranTerakhir(ctx context.Context, userID uint) (*models.Pendaftaran, error)
	ListPendaftaran(ctx context.Context, q ListQuery) ([]models.Pendaftaran, int64, error)
	// ClaimPendaftaran mengubah status dari MENUNGGU ke status baru. false = sudah diproses pihak lain.
	ClaimPendaftaran(ctx context.Context, id uint, status string, actorID uint, catatan string, at time.Time) (bool, error)

	// Tx menjalankan fn dalam satu transaksi; error dari fn = rollback semua.
	Tx(ctx context.Context, fn func(Repository) error) error
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Tx(ctx context.Context, fn func(Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&gormRepository{db: tx})
	})
}

// ================= user =================

func (r *gormRepository) FindRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Where("nama_role = ?", name).First(&role).Error; err != nil {
		return nil, mapErr(err)
	}
	return &role, nil
}

func (r *gormRepository) KontakDipakai(ctx context.Context, email, noTelepon string, kecualiUserID uint) (bool, bool, error) {
	dipakai := func(kolom, nilai string) (bool, error) {
		if nilai == "" {
			return false, nil
		}
		var jumlah int64
		err := r.db.WithContext(ctx).Unscoped().Model(&models.User{}).
			Where(kolom+" = ? AND id_user <> ?", nilai, kecualiUserID).Count(&jumlah).Error
		return jumlah > 0, err
	}
	emailDipakai, err := dipakai("email", email)
	if err != nil {
		return false, false, err
	}
	noTeleponDipakai, err := dipakai("no_telepon", noTelepon)
	return emailDipakai, noTeleponDipakai, err
}

func (r *gormRepository) CreateUser(ctx context.Context, u *models.User) error {
	// Omit("Role"): jangan ikut menulis baris role. Identitas (has-one) ikut tersimpan otomatis.
	return mapErr(r.db.WithContext(ctx).Omit("Role").Create(u).Error)
}

// HapusUserPermanen benar-benar menghapus baris user (identitas dan token ikut terhapus lewat ON DELETE CASCADE).
func (r *gormRepository) HapusUserPermanen(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Unscoped().Delete(&models.User{}, userID).Error
}

func (r *gormRepository) FindUserByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	if err := r.db.WithContext(ctx).Preload("Role").Preload("Identitas").First(&u, id).Error; err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *gormRepository) UpdateProfil(ctx context.Context, u *models.User, data models.IdentitasUser) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dataUser := map[string]any{"nama": u.Nama, "email": u.Email, "no_telepon": u.NoTelepon}
		if err := tx.Model(&models.User{}).Where("id_user = ?", u.ID).Updates(dataUser).Error; err != nil {
			return err
		}
		var jumlah int64
		if err := tx.Model(&models.IdentitasUser{}).Where("id_user = ?", u.ID).Count(&jumlah).Error; err != nil {
			return err
		}
		if jumlah == 0 {
			return tx.Create(&data).Error
		}
		// Select("*") supaya kolom yang dikosongkan jemaat ikut tersimpan kosong
		return tx.Model(&models.IdentitasUser{}).Where("id_user = ?", u.ID).
			Select("*").Omit("id_user", "created_at", "deleted_at").Updates(&data).Error
	})
	return mapErr(err)
}

func (r *gormRepository) UbahStatusUser(ctx context.Context, userID uint, dari, ke string) (bool, error) {
	res := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id_user = ? AND status_verifikasi = ?", userID, dari).
		Update("status_verifikasi", ke)
	return res.RowsAffected > 0, res.Error
}

func (r *gormRepository) SetujuiUser(ctx context.Context, userID uint, kodeJemaat string) error {
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("id_user = ?", userID).
		Updates(map[string]any{"status_verifikasi": models.Terverifikasi, "kode_jemaat": kodeJemaat}).Error
	return mapErr(err)
}

// ================= pendaftaran =================

func (r *gormRepository) CreatePendaftaran(ctx context.Context, p *models.Pendaftaran) error {
	return r.db.WithContext(ctx).Omit("User").Create(p).Error
}

func (r *gormRepository) FindPendaftaran(ctx context.Context, id uint) (*models.Pendaftaran, error) {
	var p models.Pendaftaran
	if err := r.db.WithContext(ctx).Preload("User").First(&p, id).Error; err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

func (r *gormRepository) PendaftaranTerakhir(ctx context.Context, userID uint) (*models.Pendaftaran, error) {
	var p models.Pendaftaran
	err := r.db.WithContext(ctx).Where("id_user = ?", userID).Order("id_pendaftaran DESC").First(&p).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &p, nil
}

func (r *gormRepository) ListPendaftaran(ctx context.Context, q ListQuery) ([]models.Pendaftaran, int64, error) {
	base := r.db.WithContext(ctx).Model(&models.Pendaftaran{})
	if q.Status != "" {
		base = base.Where("status = ?", q.Status)
	}
	if q.Q != "" {
		like := "%" + escapeLike(q.Q) + "%"
		cariUser := r.db.WithContext(ctx).Model(&models.User{}).Select("id_user").
			Where("nama LIKE ? OR email LIKE ? OR no_telepon LIKE ?", like, like, like)
		base = base.Where("id_user IN (?)", cariUser)
	}
	base = base.Session(&gorm.Session{}) // aman dipakai ulang untuk Count lalu Find

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var daftar []models.Pendaftaran
	err := base.Preload("User").
		Order(q.Order([]string{"created_at", "status"}, "created_at DESC")).
		Order("id_pendaftaran DESC").
		Offset(q.Offset).Limit(q.Limit).
		Find(&daftar).Error
	return daftar, total, err
}

func (r *gormRepository) ClaimPendaftaran(ctx context.Context, id uint, status string, actorID uint, catatan string, at time.Time) (bool, error) {
	var nilaiCatatan any // nil = kolom dikosongkan
	if catatan != "" {
		nilaiCatatan = catatan
	}
	res := r.db.WithContext(ctx).Model(&models.Pendaftaran{}).
		Where("id_pendaftaran = ? AND status = ?", id, models.PendaftaranMenunggu).
		Updates(map[string]any{
			"status":         status,
			"diperiksa_oleh": actorID,
			"diperiksa_pada": at,
			"catatan_admin":  nilaiCatatan,
		})
	return res.RowsAffected > 0, res.Error
}

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

// escapeLike supaya % dan _ dari user dicari apa adanya, bukan dianggap wildcard.
func escapeLike(teks string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(teks)
}
