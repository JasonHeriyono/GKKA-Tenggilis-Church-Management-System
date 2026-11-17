package user

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/identitas"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
)

// Service: pengelolaan akun jemaat dan role (khusus Super Admin; dijaga di routes lewat RequireRole).
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, q ListQuery) (*ListResponse, error) {
	q.Q = strings.TrimSpace(q.Q)
	if q.StatusVerifikasi != "" && !statusValid(q.StatusVerifikasi) {
		return nil, apperror.BadRequest("status_verifikasi harus BELUM_TERVERIFIKASI, MENUNGGU_VERIFIKASI, atau TERVERIFIKASI")
	}
	users, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := pagination.CekHalaman(q.Params, total); err != nil {
		return nil, err
	}
	items := make([]Response, 0, len(users))
	for i := range users {
		items = append(items, toResponse(&users[i]))
	}
	return &ListResponse{Items: items, Meta: pagination.NewMeta(q.Params, total)}, nil
}

func (s *Service) Get(ctx context.Context, id uint) (*Response, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	r := toResponse(u)
	return &r, nil
}

// Create: admin menambah jemaat manual. Langsung TERVERIFIKASI, kode jemaat dibuat otomatis kalau kosong.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Response, error) {
	if err := rapikanDanCek(&req.DataRequest); err != nil {
		return nil, err
	}
	role, err := s.repo.FindRoleByName(ctx, req.Role)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.Validation(map[string]string{"role": "role tidak dikenal"})
		}
		return nil, apperror.Internal(err)
	}

	kode := req.KodeJemaat
	if kode == "" {
		kode, err = identitas.BuatKodeJemaat()
		if err != nil {
			return nil, apperror.Internal(err)
		}
	}
	dataIdentitas := req.Identitas.KeModel(0) // id_user diisi GORM setelah user tersimpan
	u := &models.User{
		KodeJemaat:       &kode,
		Nama:             req.Nama,
		Email:            nilIfEmpty(req.Email),
		NoTelepon:        nilIfEmpty(req.NoTelepon),
		RoleID:           role.ID,
		Role:             *role,
		StatusVerifikasi: models.Terverifikasi, // data dari admin dianggap sudah diperiksa
		IsAktif:          true,
		Identitas:        &dataIdentitas,
	}
	if req.Password != "" {
		if err := passwordTerlaluPanjang(req.Password); err != nil {
			return nil, err
		}
		hash, err := hashPassword(req.Password)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		u.PasswordHash = &hash
	}

	if err := s.repo.Create(ctx, u); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, apperror.Conflict("Kode jemaat, email, no. telepon, atau NIK sudah terdaftar")
		}
		return nil, apperror.Internal(err)
	}
	r := toResponse(u)
	return &r, nil
}

// Update: admin mengedit data utama dan data diri jemaat. Kode jemaat kosong = tidak diubah.
func (s *Service) Update(ctx context.Context, id uint, req UpdateRequest) (*Response, error) {
	if err := rapikanDanCek(&req); err != nil {
		return nil, err
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if req.KodeJemaat != "" {
		u.KodeJemaat = &req.KodeJemaat
	}
	u.Nama = req.Nama
	u.Email = nilIfEmpty(req.Email)
	u.NoTelepon = nilIfEmpty(req.NoTelepon)

	if err := s.repo.Update(ctx, u, req.Identitas.KeModel(u.ID)); err != nil {
		if errors.Is(err, ErrDuplicate) {
			return nil, apperror.Conflict("Kode jemaat, email, no. telepon, atau NIK sudah dipakai jemaat lain")
		}
		return nil, apperror.Internal(err)
	}
	return s.Get(ctx, id)
}

// Delete: hapus sementara (data absensi/pelayanan tetap ada). Akun sendiri tidak boleh dihapus.
func (s *Service) Delete(ctx context.Context, actorID, id uint) error {
	if actorID == id {
		return apperror.BadRequest("Tidak dapat menghapus akun sendiri")
	}
	if err := s.repo.Delete(ctx, id, actorID); err != nil {
		return notFoundOrInternal(err)
	}
	return nil
}

// ChangeRole mengganti role user. Akun sendiri tidak boleh diubah, supaya sistem tidak kehilangan Super Admin terakhir.
func (s *Service) ChangeRole(ctx context.Context, actorID, id uint, roleName string) (*Response, error) {
	if actorID == id {
		return nil, apperror.BadRequest("Tidak dapat mengubah role akun sendiri")
	}
	role, err := s.repo.FindRoleByName(ctx, roleName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, apperror.Validation(map[string]string{"role": "role tidak dikenal"})
		}
		return nil, apperror.Internal(err)
	}
	if err := s.repo.UpdateRole(ctx, id, role.ID); err != nil {
		return nil, notFoundOrInternal(err)
	}
	return s.Get(ctx, id)
}

// SetActive mengaktifkan atau menonaktifkan akun. Akun sendiri tidak boleh dinonaktifkan.
func (s *Service) SetActive(ctx context.Context, actorID, id uint, aktif bool) (*Response, error) {
	if actorID == id && !aktif {
		return nil, apperror.BadRequest("Tidak dapat menonaktifkan akun sendiri")
	}
	if err := s.repo.SetActive(ctx, id, aktif); err != nil {
		return nil, notFoundOrInternal(err)
	}
	return s.Get(ctx, id)
}

// SetPassword mengatur password (sekaligus mengaktifkan akun yang belum punya password).
// Semua sesi login user tersebut dicabut.
func (s *Service) SetPassword(ctx context.Context, id uint, password string) error {
	if err := passwordTerlaluPanjang(password); err != nil {
		return err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return apperror.Internal(err)
	}
	if err := s.repo.SetPassword(ctx, id, hash); err != nil {
		return notFoundOrInternal(err)
	}
	return nil
}

// rapikanDanCek: buang spasi berlebih, lalu cek isian yang tidak bisa dicek lewat binding.
func rapikanDanCek(req *DataRequest) error {
	req.KodeJemaat = strings.TrimSpace(req.KodeJemaat)
	req.Nama = strings.Join(strings.Fields(req.Nama), " ")
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.NoTelepon = identitas.RapikanNoHP(req.NoTelepon)
	req.Identitas.Rapikan()

	errorPerField := req.Identitas.Cek()
	if req.Nama == "" { // binding "required" masih meloloskan teks yang isinya cuma spasi
		errorPerField["nama"] = "wajib diisi"
	}
	if req.NoTelepon != "" && !identitas.NoHPValid(req.NoTelepon) {
		errorPerField["no_telepon"] = "format tidak valid (contoh: 081234567890 atau +6591234567)"
	}
	if len(errorPerField) > 0 {
		return apperror.Validation(errorPerField)
	}
	return nil
}

func statusValid(status string) bool {
	return status == models.BelumTerverifikasi || status == models.MenungguVerifikasi || status == models.Terverifikasi
}

const maxPasswordBytes = 72

func passwordTerlaluPanjang(password string) error {
	if len(password) > maxPasswordBytes { // len() pada string = jumlah byte
		return apperror.Validation(map[string]string{"password": "terlalu panjang (maks 72 byte)"})
	}
	return nil
}

func hashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(b), err
}

func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("User tidak ditemukan")
	}
	return apperror.Internal(err)
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toResponse(u *models.User) Response {
	return Response{
		ID:               u.ID,
		KodeJemaat:       u.KodeJemaat,
		Nama:             u.Nama,
		Email:            u.Email,
		NoTelepon:        u.NoTelepon,
		Role:             u.Role.Nama,
		StatusVerifikasi: u.StatusVerifikasi,
		IsAktif:          u.IsAktif,
		AccountActivated: u.PasswordHash != nil,
		Identitas:        identitas.DariModel(u.Identitas),
	}
}
