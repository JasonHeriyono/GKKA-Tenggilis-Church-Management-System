package pengaturan

import (
	"context"
	"strings"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List: semua pengaturan yang boleh diubah admin, yang belum disimpan memakai nilai bawaan.
func (service *Service) List(ctx context.Context) ([]PengaturanResponse, error) {
	tersimpan, err := service.repo.List(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	byKunci := map[string]models.Pengaturan{}
	for _, baris := range tersimpan {
		byKunci[baris.Kunci] = baris
	}

	hasil := make([]PengaturanResponse, 0, len(daftarAturan))
	for _, aturan := range daftarAturan {
		item := PengaturanResponse{Kunci: aturan.Kunci, Nilai: aturan.Bawaan, Bawaan: aturan.Bawaan, Deskripsi: aturan.Deskripsi}
		if baris, ada := byKunci[aturan.Kunci]; ada {
			item.Nilai = baris.Nilai
			item.DiubahOleh = baris.DiubahOleh
			item.UpdatedAt = &baris.UpdatedAt
		}
		hasil = append(hasil, item)
	}
	return hasil, nil
}

// Simpan: semua nilai dicek dulu, kalau ada satu yang salah tidak ada yang disimpan.
func (service *Service) Simpan(ctx context.Context, req UpdatePengaturanRequest, adminID uint) ([]PengaturanResponse, error) {
	if len(req.Pengaturan) == 0 {
		return nil, apperror.BadRequest("Tidak ada pengaturan yang dikirim")
	}

	errorPerField := map[string]string{}
	var daftar []models.Pengaturan
	for kunci, nilai := range req.Pengaturan {
		nilai = strings.TrimSpace(nilai)
		aturan := cariAturan(kunci)
		if aturan == nil {
			errorPerField[kunci] = "pengaturan tidak dikenal"
			continue
		}
		if pesan := aturan.Cek(nilai); pesan != "" {
			errorPerField[kunci] = pesan
			continue
		}
		deskripsi := aturan.Deskripsi
		daftar = append(daftar, models.Pengaturan{Kunci: kunci, Nilai: nilai, Deskripsi: &deskripsi, DiubahOleh: &adminID})
	}
	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}

	if err := service.repo.Simpan(ctx, daftar); err != nil {
		return nil, apperror.Internal(err)
	}
	return service.List(ctx)
}
