package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"sim-g-backend/models"
)

// ErrNotFound dikembalikan repository jika data tidak ada.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository dibuat sebagai interface supaya service mudah diuji dengan fake.
type Repository interface {
	// FindUserByIdentifier mencari user (beserta role) berdasarkan email, kode jemaat, atau no. telepon.
	FindUserByIdentifier(ctx context.Context, identifier string) (*models.User, error)
	FindUserByID(ctx context.Context, id uint) (*models.User, error)
	// ListKeahlianUser: keahlian milik user, urut nama.
	ListKeahlianUser(ctx context.Context, userID uint) ([]models.Keahlian, error)

	CreateRefreshToken(ctx context.Context, t *models.RefreshToken) error
	FindRefreshToken(ctx context.Context, hash string) (*models.RefreshToken, error)
	// RotateRefreshToken mencabut token lama dan menyimpan token baru dalam satu transaksi.
	// rotated=false jika token lama ternyata sudah dicabut oleh request lain.
	RotateRefreshToken(ctx context.Context, oldID uint, next *models.RefreshToken) (rotated bool, err error)
	RevokeRefreshToken(ctx context.Context, hash string) error
	RevokeAllUserTokens(ctx context.Context, userID uint) error
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) FindUserByIdentifier(ctx context.Context, identifier string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Role").
		Where("email = ? OR kode_jemaat = ? OR no_telepon = ?", identifier, identifier, identifier).
		First(&u).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *gormRepository) FindUserByID(ctx context.Context, id uint) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Preload("Role").First(&u, id).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *gormRepository) ListKeahlianUser(ctx context.Context, userID uint) ([]models.Keahlian, error) {
	idKeahlianUser := r.db.WithContext(ctx).Model(&models.UserKeahlian{}).
		Select("id_keahlian").Where("id_user = ?", userID)

	var daftar []models.Keahlian
	err := r.db.WithContext(ctx).
		Where("id_keahlian IN (?)", idKeahlianUser).
		Order("nama_keahlian ASC").
		Find(&daftar).Error
	return daftar, err
}

func (r *gormRepository) CreateRefreshToken(ctx context.Context, t *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *gormRepository) FindRefreshToken(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if err != nil {
		return nil, mapErr(err)
	}
	return &t, nil
}

func (r *gormRepository) RotateRefreshToken(ctx context.Context, oldID uint, next *models.RefreshToken) (bool, error) {
	rotated := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.RefreshToken{}).
			Where("id_token = ? AND revoked_at IS NULL", oldID).
			Update("revoked_at", time.Now())
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // sudah dicabut request lain
		}
		rotated = true
		return tx.Create(next).Error
	})
	return rotated && err == nil, err
}

func (r *gormRepository) RevokeRefreshToken(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", time.Now()).Error
}

func (r *gormRepository) RevokeAllUserTokens(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("id_user = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// mapErr menerjemahkan "record not found" GORM menjadi ErrNotFound.
func mapErr(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
