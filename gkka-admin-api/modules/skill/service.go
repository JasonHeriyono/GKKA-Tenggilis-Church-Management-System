package skill

import (
	"context"
	"errors"
	"strings"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List mengembalikan keahlian per halaman beserta meta pagination.
func (service *Service) List(ctx context.Context, filter ListQuery) (*ListResponse, error) {
	filter.Q = strings.TrimSpace(filter.Q)

	daftarKeahlian, total, err := service.repo.List(ctx, filter)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := pagination.CekHalaman(filter.Params, total); err != nil {
		return nil, err
	}
	return &ListResponse{
		Items: toResponses(daftarKeahlian),
		Meta:  pagination.NewMeta(filter.Params, total),
	}, nil
}

// Get mengembalikan detail keahlian beserta jemaat yang memilikinya.
func (service *Service) Get(ctx context.Context, keahlianID uint) (*DetailResponse, error) {
	keahlian, err := service.repo.FindByID(ctx, keahlianID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	daftarJemaat, err := service.repo.ListJemaat(ctx, keahlianID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	detail := &DetailResponse{
		Response: toResponse(keahlian),
		Jemaat:   make([]JemaatResponse, 0, len(daftarJemaat)),
	}
	for _, baris := range daftarJemaat {
		detail.Jemaat = append(detail.Jemaat, JemaatResponse{
			UserID:     baris.UserID,
			KodeJemaat: baris.User.KodeJemaat,
			Nama:       baris.User.Nama,
		})
	}
	return detail, nil
}

// Create membuat keahlian baru; 409 jika nama sudah dipakai. is_sistem selalu false dari API.
// Kalau namanya sama dengan keahlian yang pernah dihapus, keahlian lama itu dipulihkan (beserta jemaatnya).
func (service *Service) Create(ctx context.Context, req SaveRequest) (*Response, error) {
	nama := strings.TrimSpace(req.Nama)
	if nama == "" { // binding "required" masih meloloskan nama yang isinya cuma spasi
		return nil, apperror.Validation(map[string]string{"nama": "wajib diisi"})
	}

	keahlianLama, err := service.repo.FindDeletedByNama(ctx, nama)
	if err == nil {
		keahlianLama.Nama = nama
		if err := service.repo.Restore(ctx, keahlianLama); err != nil {
			return nil, apperror.Internal(err)
		}
		res := toResponse(keahlianLama)
		return &res, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}

	keahlian := &models.Keahlian{Nama: nama}
	if err := service.repo.Create(ctx, keahlian); err != nil {
		return nil, duplicateOrInternal(err)
	}
	res := toResponse(keahlian)
	return &res, nil
}

// Update mengganti nama keahlian. Keahlian sistem (Usher) tidak boleh diubah.
func (service *Service) Update(ctx context.Context, keahlianID uint, req SaveRequest) (*Response, error) {
	nama := strings.TrimSpace(req.Nama)
	if nama == "" {
		return nil, apperror.Validation(map[string]string{"nama": "wajib diisi"})
	}
	keahlian, err := service.repo.FindByID(ctx, keahlianID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if keahlian.IsSistem {
		return nil, apperror.Forbidden("Keahlian sistem tidak dapat diubah")
	}
	keahlian.Nama = nama
	if err := service.repo.Update(ctx, keahlian); err != nil {
		return nil, duplicateOrInternal(err)
	}
	res := toResponse(keahlian)
	return &res, nil
}

// Delete = soft delete. Ditolak untuk keahlian sistem dan keahlian yang masih dipakai di jadwal pelayanan.
func (service *Service) Delete(ctx context.Context, keahlianID uint) error {
	keahlian, err := service.repo.FindByID(ctx, keahlianID)
	if err != nil {
		return notFoundOrInternal(err)
	}
	if keahlian.IsSistem {
		return apperror.Forbidden("Keahlian sistem tidak dapat dihapus")
	}
	dipakai, err := service.repo.InUseBySchedule(ctx, keahlianID)
	if err != nil {
		return apperror.Internal(err)
	}
	if dipakai {
		return apperror.Conflict("Keahlian masih dipakai di jadwal pelayanan")
	}
	if err := service.repo.Delete(ctx, keahlianID); err != nil {
		return notFoundOrInternal(err)
	}
	return nil
}

// AddJemaat memberi keahlian ke satu jemaat lalu mengembalikan detail terbaru.
func (service *Service) AddJemaat(ctx context.Context, keahlianID uint, req JemaatRequest) (*DetailResponse, error) {
	if _, err := service.repo.FindByID(ctx, keahlianID); err != nil {
		return nil, notFoundOrInternal(err)
	}
	userAda, err := service.repo.UserExists(ctx, req.UserID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if !userAda {
		return nil, apperror.Validation(map[string]string{"id_user": "user tidak ditemukan"})
	}
	baris := &models.UserKeahlian{UserID: req.UserID, KeahlianID: keahlianID}
	if err := service.repo.AddJemaat(ctx, baris); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, apperror.Conflict("User sudah memiliki keahlian ini")
		}
		return nil, apperror.Internal(err)
	}
	return service.Get(ctx, keahlianID)
}

// RemoveJemaat melepas keahlian dari satu jemaat.
func (service *Service) RemoveJemaat(ctx context.Context, keahlianID, userID uint) error {
	if err := service.repo.RemoveJemaat(ctx, keahlianID, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return apperror.NotFound("User tidak memiliki keahlian ini")
		}
		return apperror.Internal(err)
	}
	return nil
}

// ListByUser mengembalikan semua keahlian milik satu jemaat.
func (service *Service) ListByUser(ctx context.Context, userID uint) (*UserKeahlianResponse, error) {
	if err := service.ensureUser(ctx, userID); err != nil {
		return nil, err
	}
	daftarKeahlian, err := service.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &UserKeahlianResponse{UserID: userID, Keahlian: toResponses(daftarKeahlian)}, nil
}

// ReplaceByUser mengganti seluruh keahlian satu jemaat dengan daftar baru (id ganda diabaikan).
func (service *Service) ReplaceByUser(ctx context.Context, userID uint, req ReplaceRequest) (*UserKeahlianResponse, error) {
	if err := service.ensureUser(ctx, userID); err != nil {
		return nil, err
	}
	ids := unique(req.IDs)
	if len(ids) > 0 {
		ada, err := service.repo.CountExisting(ctx, ids)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		if int(ada) != len(ids) {
			return nil, apperror.Validation(map[string]string{"id_keahlian": "ada keahlian yang tidak ditemukan"})
		}
	}
	if err := service.repo.ReplaceUserKeahlian(ctx, userID, ids); err != nil {
		return nil, apperror.Internal(err)
	}
	return service.ListByUser(ctx, userID)
}

func (service *Service) ensureUser(ctx context.Context, userID uint) error {
	ada, err := service.repo.UserExists(ctx, userID)
	if err != nil {
		return apperror.Internal(err)
	}
	if !ada {
		return apperror.NotFound("User tidak ditemukan")
	}
	return nil
}

// unique membuang id ganda dengan urutan tetap.
func unique(ids []uint) []uint {
	sudah := make(map[uint]struct{}, len(ids))
	hasil := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ada := sudah[id]; ada {
			continue
		}
		sudah[id] = struct{}{}
		hasil = append(hasil, id)
	}
	return hasil
}

// notFoundOrInternal: ErrNotFound -> 404, error lain -> 500.
func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("Keahlian tidak ditemukan")
	}
	return apperror.Internal(err)
}

// duplicateOrInternal: ErrDuplicate -> 409. Unique key nama juga berlaku untuk baris yang sudah di-soft delete.
func duplicateOrInternal(err error) error {
	if errors.Is(err, ErrDuplicate) {
		return apperror.Conflict("Nama keahlian sudah dipakai")
	}
	return apperror.Internal(err)
}

// toResponse mengubah model database jadi bentuk JSON untuk client.
func toResponse(keahlian *models.Keahlian) Response {
	return Response{ID: keahlian.ID, Nama: keahlian.Nama, IsSistem: keahlian.IsSistem}
}

func toResponses(daftar []models.Keahlian) []Response {
	items := make([]Response, 0, len(daftar))
	for i := range daftar {
		items = append(items, toResponse(&daftar[i]))
	}
	return items
}
