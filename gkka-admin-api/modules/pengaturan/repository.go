package pengaturan

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"sim-g-backend/models"
)

type Repository interface {
	List(ctx context.Context) ([]models.Pengaturan, error)
	Simpan(ctx context.Context, daftar []models.Pengaturan) error
}

type gormRepository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (repo *gormRepository) List(ctx context.Context) ([]models.Pengaturan, error) {
	var daftar []models.Pengaturan
	err := repo.db.WithContext(ctx).Find(&daftar).Error
	return daftar, err
}

// Simpan: insert, atau update kalau kuncinya sudah ada.
func (repo *gormRepository) Simpan(ctx context.Context, daftar []models.Pengaturan) error {
	return repo.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kunci"}},
		DoUpdates: clause.AssignmentColumns([]string{"nilai", "deskripsi", "diubah_oleh", "updated_at"}),
	}).Create(&daftar).Error
}
