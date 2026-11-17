package group

import (
	"context"
	"errors"
	"strings"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
)

type Service struct{
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List mengembalikan komunitas per halaman beserta meta pagination.
func (service *Service) List(ctx context.Context, filter ListQuery) (*ListResponse, error) {
        filter.Q = strings.TrimSpace(filter.Q)

        daftarKomunitas, total, err := service.repo.List(ctx, filter)
        if err != nil {
                return nil, apperror.Internal(err)
        }
        if err := pagination.CekHalaman(filter.Params, total); err != nil {
                return nil, err
        }

        items := make([]Response, 0, len(daftarKomunitas))
        for i := range daftarKomunitas {
                items = append(items, toResponse(&daftarKomunitas[i]))
        }
        return &ListResponse{
                Items: items,
                Meta:  pagination.NewMeta(filter.Params, total),
        }, nil
}
// Get mengembalikan detail komunitas beserta anggotanya (ketua di urutan atas).
func (service *Service) Get(ctx context.Context, komunitasID uint) (*DetailResponse, error) {
	komunitas, err := service.repo.FindByID(ctx, komunitasID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	
	daftarAnggota, err := service.repo.ListAnggota(ctx, komunitasID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	detail := &DetailResponse{
		Response: toResponse(komunitas),
		Anggota:  make([]AnggotaResponse, 0, len(daftarAnggota)),
	}

	for _, anggota := range daftarAnggota {
		detail.Anggota = append(detail.Anggota, AnggotaResponse{
				UserID:     anggota.UserID,
				KodeJemaat: anggota.User.KodeJemaat,
				Nama:       anggota.User.Nama,
				IsKetua:    anggota.IsKetua,
		})
	}
	return detail, nil
}
// Create membuat komunitas baru; 409 jika nama sudah dipakai komunitas lain.
// Kalau namanya sama dengan komunitas yang pernah dihapus, komunitas lama itu dipulihkan (beserta anggotanya).
func (service *Service) Create(ctx context.Context, req SaveRequest) (*Response, error) {
	nama := strings.TrimSpace(req.Nama)
	if nama == "" { // binding "required" masih meloloskan nama yang isinya cuma spasi
		return nil, apperror.Validation(map[string]string{"nama": "wajib diisi"})
	}
	deskripsi := nilIfEmpty(strings.TrimSpace(req.Deskripsi))

	komunitasLama, err := service.repo.FindDeletedByNama(ctx, nama)
	if err == nil {
		komunitasLama.Nama = nama
		komunitasLama.Deskripsi = deskripsi
		if err := service.repo.Restore(ctx, komunitasLama); err != nil {
			return nil, apperror.Internal(err)
		}
		res := toResponse(komunitasLama)
		return &res, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}

	komunitas := &models.Komunitas{Nama: nama, Deskripsi: deskripsi}
	if err := service.repo.Create(ctx, komunitas); err != nil {
		return nil, duplicateOrInternal(err)
	}

	res := toResponse(komunitas)
	return &res, nil
}

//Update komunitas 
func (service *Service) Update(ctx context.Context, komunitasID uint, req SaveRequest) (*Response, error) {
	komunitas, err := service.repo.FindByID(ctx, komunitasID)
	if err != nil {
			return nil, notFoundOrInternal(err)
	}
	nama := strings.TrimSpace(req.Nama)
	if nama == "" { // binding "required" masih meloloskan nama yang isinya cuma spasi
		return nil, apperror.Validation(map[string]string{"nama": "wajib diisi"})
	}
	komunitas.Nama = nama
	komunitas.Deskripsi = nilIfEmpty(strings.TrimSpace(req.Deskripsi))
	if err := service.repo.Update(ctx, komunitas); err != nil {
		return nil, duplicateOrInternal(err)
	}
	res := toResponse(komunitas)
	return &res, nil
}

// Soft delete komunitas, data lama tetap ada di database. 
func (service *Service) Delete(ctx context.Context, komunitasID uint) error {
	if err := service.repo.Delete(ctx, komunitasID); err != nil {
		return notFoundOrInternal(err)
	}
	return nil
}

// AddAnggota memasukkan user ke komunitas lalu mengembalikan detail terbaru.
func (service *Service) AddAnggota(ctx context.Context, komunitasID uint, req AnggotaRequest) (*DetailResponse, error) {
	if _, err := service.repo.FindByID(ctx, komunitasID); err != nil {
			return nil, notFoundOrInternal(err)
	}
	userAda, err := service.repo.UserExists(ctx, req.UserID)
	if err != nil {
			return nil, apperror.Internal(err)
	}
	if !userAda {
			return nil, apperror.Validation(
				map[string]string{
					"id_user": "user tidak ditemukan",
				})
	}
	anggota := &models.UserKomunitas{UserID: req.UserID, KomunitasID: komunitasID, IsKetua: req.IsKetua}
	if err := service.repo.AddAnggota(ctx, anggota); err != nil {
			if errors.Is(err, ErrDuplicate) {
					return nil, apperror.Conflict("User sudah menjadi anggota komunitas ini")
			}
			return nil, apperror.Internal(err)
	}
	return service.Get(ctx, komunitasID)
}

// SetKetua menjadikan / membatalkan ketua lalu mengembalikan detail terbaru.
func (service *Service) SetKetua(ctx context.Context, komunitasID, userID uint, isKetua bool) (*DetailResponse, error) {
	if err := service.repo.SetKetua(ctx, komunitasID, userID, isKetua); err != nil {
			return nil, anggotaNotFoundOrInternal(err)
	}
	return service.Get(ctx, komunitasID)
}
// RemoveAnggota mengeluarkan user dari komunitas.
func (service *Service) RemoveAnggota(ctx context.Context, komunitasID, userID uint) error {
	if err := service.repo.RemoveAnggota(ctx, komunitasID, userID); err != nil {
			return anggotaNotFoundOrInternal(err)
	}
	return nil
}

// notFoundOrInternal: ErrNotFound -> 404, error lain -> 500.
func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
			return apperror.NotFound("Komunitas tidak ditemukan")
	}
	return apperror.Internal(err)
}

// anggotaNotFoundOrInternal: sama, tapi pesannya untuk keanggotaan.
func anggotaNotFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
			return apperror.NotFound("User bukan anggota komunitas ini")
	}
	return apperror.Internal(err)
}

// duplicateOrInternal: ErrDuplicate -> 409. Unique key nama juga berlaku untuk baris yang sudah di-soft delete.
func duplicateOrInternal(err error) error {
	if errors.Is(err, ErrDuplicate) {
			return apperror.Conflict("Nama komunitas sudah dipakai")
	}
	return apperror.Internal(err)
}

// nilIfEmpty: string kosong disimpan sebagai NULL di database.
func nilIfEmpty(teks string) *string {
	if teks == "" {
			return nil
	}
	return &teks
}

// toResponse mengubah model database jadi bentuk JSON untuk client.
func toResponse(komunitas *models.Komunitas) Response {
	return Response{ID: komunitas.ID, Nama: komunitas.Nama, Deskripsi: komunitas.Deskripsi}
}