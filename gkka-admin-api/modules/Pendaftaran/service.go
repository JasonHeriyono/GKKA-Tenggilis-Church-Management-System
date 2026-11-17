package pendaftaran

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/identitas"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
	"sim-g-backend/modules/auth"
)

// Alur (ERD R4):
//  1. Signup: akun dibuat, jemaat langsung masuk dengan status BELUM_TERVERIFIKASI.
//  2. Jemaat melengkapi data diri (PUT /profil/saya), lalu mengajukan verifikasi (POST /verifikasi).
//  3. Sekretariat menyetujui (TERVERIFIKASI + kode jemaat) atau menolak (kembali BELUM_TERVERIFIKASI).

// PenerbitToken: modul auth, supaya jemaat langsung masuk setelah signup tanpa login ulang.
type PenerbitToken interface {
	MasukOtomatis(ctx context.Context, userID uint) (*auth.LoginResponse, error)
}

type Service struct {
	repo  Repository
	token PenerbitToken
	now   func() time.Time
}

func NewService(repo Repository, token PenerbitToken) *Service {
	return &Service{repo: repo, token: token, now: time.Now}
}

// ================= 1. Signup =================

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*auth.LoginResponse, error) {
	nama := rapikanNama(req.Nama)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	noHP := identitas.RapikanNoHP(req.NoTelepon)

	errorPerField := cekKontak(nama, email, noHP)
	if len(req.Password) > 72 { // batas bcrypt dalam byte (binding menghitung karakter)
		errorPerField["password"] = "maksimal 72 byte"
	}
	if !req.PersetujuanData {
		errorPerField["persetujuan_data"] = "wajib disetujui"
	}
	dataDiri := identitas.Isian{JenisKelamin: req.JenisKelamin, TanggalLahir: req.TanggalLahir, StatusGereja: req.StatusGereja}
	for namaField, pesan := range dataDiri.Cek() {
		errorPerField[namaField] = pesan
	}
	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}
	if err := s.cekKontakBelumDipakai(ctx, email, noHP, 0); err != nil {
		return nil, err
	}

	role, err := s.repo.FindRoleByName(ctx, models.RoleJemaat)
	if err != nil { // role belum ada = seed belum dijalankan
		return nil, apperror.Internal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	passwordHash := string(hash)
	sekarang := s.now()
	dataIdentitas := dataDiri.KeModel(0) // id_user diisi GORM setelah user tersimpan
	user := &models.User{
		Nama:                nama,
		Email:               nilKalauKosong(email),
		NoTelepon:           nilKalauKosong(noHP),
		PasswordHash:        &passwordHash,
		RoleID:              role.ID,
		Role:                *role,
		StatusVerifikasi:    models.BelumTerverifikasi,
		PersetujuanDataPada: &sekarang,
		IsAktif:             true,
		Identitas:           &dataIdentitas,
	}
	err = s.repo.CreateUser(ctx, user)
	if errors.Is(err, ErrDuplicate) { // email / no HP baru saja dipakai orang lain di waktu yang sama
		return nil, apperror.Conflict("Email atau no. telepon sudah terdaftar")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	// signup = buat akun + langsung masuk. Kalau token gagal dibuat, akun dibatalkan
	// supaya jemaat bisa langsung daftar ulang dengan email/no HP yang sama.
	hasil, err := s.token.MasukOtomatis(ctx, user.ID)
	if err != nil {
		if errHapus := s.repo.HapusUserPermanen(ctx, user.ID); errHapus != nil {
			err = errors.Join(err, errHapus)
		}
		return nil, apperror.Internal(err)
	}
	return hasil, nil
}

// ================= 2. Profil sendiri + ajukan verifikasi =================

func (s *Service) ProfilSaya(ctx context.Context, userID uint) (*ProfilResponse, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, userNotFoundOrInternal(err)
	}
	dataDiri := identitas.DariModel(user.Identitas)
	hasil := &ProfilResponse{
		ID:               user.ID,
		KodeJemaat:       user.KodeJemaat,
		Nama:             user.Nama,
		Email:            user.Email,
		NoTelepon:        user.NoTelepon,
		StatusVerifikasi: user.StatusVerifikasi,
		Identitas:        dataDiri,
		Kelengkapan:      dataDiri.HitungKelengkapan(),
		KolomTerkunci:    []string{},
	}
	hasil.Identitas.NIK = "" // NIK hanya untuk Super Admin
	if user.StatusVerifikasi == models.Terverifikasi {
		hasil.KolomTerkunci = identitas.KolomDataInti
	}

	terakhir, err := s.repo.PendaftaranTerakhir(ctx, userID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}
	if terakhir != nil {
		hasil.PengajuanTerakhir = &PengajuanRingkas{
			ID: terakhir.ID, Status: terakhir.Status, CatatanAdmin: terakhir.CatatanAdmin, DiajukanPada: terakhir.CreatedAt,
		}
	}
	return hasil, nil
}

// UbahProfilSaya: jemaat mengubah data dirinya. Tidak bisa saat pengajuan sedang diperiksa.
func (s *Service) UbahProfilSaya(ctx context.Context, userID uint, req ProfilRequest) (*ProfilResponse, error) {
	nama := rapikanNama(req.Nama)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	noHP := identitas.RapikanNoHP(req.NoTelepon)
	req.Identitas.Rapikan()

	errorPerField := cekKontak(nama, email, noHP)
	for namaField, pesan := range req.Identitas.Cek() {
		errorPerField[namaField] = pesan
	}
	if len(errorPerField) > 0 {
		return nil, apperror.Validation(errorPerField)
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, userNotFoundOrInternal(err)
	}
	if user.StatusVerifikasi == models.MenungguVerifikasi {
		return nil, apperror.Conflict("Data sedang diperiksa sekretariat, tunggu sampai selesai untuk mengubahnya")
	}
	if err := s.cekKontakBelumDipakai(ctx, email, noHP, userID); err != nil {
		return nil, err
	}

	// NIK dan data keanggotaan hanya boleh diubah admin, pakai nilai lama
	dataLama := identitas.DariModel(user.Identitas)
	req.Identitas.PakaiKolomAdminDari(dataLama)
	// sudah TERVERIFIKASI: data inti sudah diperiksa sekretariat, perubahan dari jemaat diabaikan
	if user.StatusVerifikasi == models.Terverifikasi {
		req.Identitas.PakaiDataIntiDari(dataLama)
		nama = user.Nama
	}
	user.Nama = nama
	user.Email = nilKalauKosong(email)
	user.NoTelepon = nilKalauKosong(noHP)
	err = s.repo.UpdateProfil(ctx, user, req.Identitas.KeModel(userID))
	if errors.Is(err, ErrDuplicate) {
		return nil, apperror.Conflict("Email atau no. telepon sudah dipakai akun lain")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return s.ProfilSaya(ctx, userID)
}

// AjukanVerifikasi: data wajib harus lengkap; salinan data disimpan untuk diperiksa sekretariat.
func (s *Service) AjukanVerifikasi(ctx context.Context, userID uint) (*PengajuanRingkas, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, userNotFoundOrInternal(err)
	}
	switch user.StatusVerifikasi {
	case models.MenungguVerifikasi:
		return nil, apperror.Conflict("Pengajuan Anda sedang diperiksa sekretariat")
	case models.Terverifikasi:
		return nil, apperror.Conflict("Akun Anda sudah terverifikasi")
	}

	dataDiri := identitas.DariModel(user.Identitas)
	kelengkapan := dataDiri.HitungKelengkapan()
	if len(kelengkapan.Kurang) > 0 {
		errorPerField := map[string]string{}
		for _, namaField := range kelengkapan.Kurang {
			errorPerField[namaField] = "wajib diisi sebelum mengajukan verifikasi"
		}
		return nil, apperror.Validation(errorPerField)
	}

	dataDiri.NIK = "" // salinan untuk pemeriksa tidak perlu NIK
	salinan, err := json.Marshal(dataDiajukan{
		Nama: user.Nama, Email: teks(user.Email), NoTelepon: teks(user.NoTelepon), Identitas: dataDiri,
	})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	pengajuan := &models.Pendaftaran{UserID: userID, DataDiajukan: string(salinan), Status: models.PendaftaranMenunggu}
	err = s.repo.Tx(ctx, func(tx Repository) error {
		// status diubah dulu supaya klik dua kali tidak membuat dua pengajuan
		berhasil, err := tx.UbahStatusUser(ctx, userID, models.BelumTerverifikasi, models.MenungguVerifikasi)
		if err != nil {
			return err
		}
		if !berhasil {
			return ErrStateChanged
		}
		return tx.CreatePendaftaran(ctx, pengajuan)
	})
	if errors.Is(err, ErrStateChanged) {
		return nil, apperror.Conflict("Pengajuan Anda sedang diperiksa sekretariat")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &PengajuanRingkas{ID: pengajuan.ID, Status: pengajuan.Status, DiajukanPada: pengajuan.CreatedAt}, nil
}

// ================= 3. Sisi Super Admin =================

func (s *Service) List(ctx context.Context, q ListQuery) (*ListResponse, error) {
	q.Q = strings.TrimSpace(q.Q)
	if q.Status != "" && q.Status != models.PendaftaranMenunggu && q.Status != models.PendaftaranDisetujui && q.Status != models.PendaftaranDitolak {
		return nil, apperror.BadRequest("status harus MENUNGGU, DISETUJUI, atau DITOLAK")
	}
	daftar, total, err := s.repo.ListPendaftaran(ctx, q)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := pagination.CekHalaman(q.Params, total); err != nil {
		return nil, err
	}
	items := make([]ListItem, 0, len(daftar))
	for i := range daftar {
		items = append(items, toListItem(&daftar[i]))
	}
	return &ListResponse{Items: items, Meta: pagination.NewMeta(q.Params, total)}, nil
}

func (s *Service) Get(ctx context.Context, id uint) (*DetailResponse, error) {
	p, err := s.repo.FindPendaftaran(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	var salinan dataDiajukan
	if err := json.Unmarshal([]byte(p.DataDiajukan), &salinan); err != nil {
		return nil, apperror.Internal(err)
	}
	hasil := &DetailResponse{
		ListItem:      toListItem(p),
		DataDiajukan:  salinan,
		DiperiksaOleh: p.DiperiksaOleh,
		CatatanAdmin:  p.CatatanAdmin,
	}
	if p.User != nil {
		hasil.StatusVerifikasi = p.User.StatusVerifikasi
		hasil.KodeJemaat = p.User.KodeJemaat
	}
	return hasil, nil
}

// Setujui: akun jadi TERVERIFIKASI dan mendapat kode jemaat (kalau belum punya).
func (s *Service) Setujui(ctx context.Context, actorID, id uint, req SetujuiRequest) (*ProsesResponse, error) {
	p, err := s.muatMenunggu(ctx, id)
	if err != nil {
		return nil, err
	}
	kode := ""
	if p.User != nil && p.User.KodeJemaat != nil {
		kode = *p.User.KodeJemaat
	} else {
		kode, err = identitas.BuatKodeJemaat()
		if err != nil {
			return nil, apperror.Internal(err)
		}
	}
	err = s.repo.Tx(ctx, func(tx Repository) error {
		if err := s.klaim(ctx, tx, p.ID, models.PendaftaranDisetujui, actorID, req.CatatanAdmin); err != nil {
			return err
		}
		return tx.SetujuiUser(ctx, p.UserID, kode)
	})
	if err != nil {
		return nil, ubahError(err)
	}
	return &ProsesResponse{ID: p.ID, Status: models.PendaftaranDisetujui, UserID: p.UserID, KodeJemaat: &kode}, nil
}

// Tolak: alasan wajib; jemaat kembali BELUM_TERVERIFIKASI dan bisa memperbaiki data lalu mengajukan lagi.
func (s *Service) Tolak(ctx context.Context, actorID, id uint, req TolakRequest) (*ProsesResponse, error) {
	catatan := strings.TrimSpace(req.CatatanAdmin)
	if catatan == "" {
		return nil, apperror.Validation(map[string]string{"catatan_admin": "wajib diisi"})
	}
	p, err := s.muatMenunggu(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.repo.Tx(ctx, func(tx Repository) error {
		if err := s.klaim(ctx, tx, p.ID, models.PendaftaranDitolak, actorID, catatan); err != nil {
			return err
		}
		_, err := tx.UbahStatusUser(ctx, p.UserID, models.MenungguVerifikasi, models.BelumTerverifikasi)
		return err
	})
	if err != nil {
		return nil, ubahError(err)
	}
	return &ProsesResponse{ID: p.ID, Status: models.PendaftaranDitolak, UserID: p.UserID}, nil
}

// ================= pembantu =================

// muatMenunggu mengambil pengajuan dan memastikan statusnya masih MENUNGGU.
func (s *Service) muatMenunggu(ctx context.Context, id uint) (*models.Pendaftaran, error) {
	p, err := s.repo.FindPendaftaran(ctx, id)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}
	if p.Status != models.PendaftaranMenunggu {
		return nil, apperror.Conflict(fmt.Sprintf("Pengajuan sudah diproses (status: %s)", p.Status))
	}
	if p.User == nil { // jaga-jaga: akun pengaju sudah dihapus
		return nil, apperror.Conflict("Akun yang mengajukan sudah dihapus")
	}
	return p, nil
}

// klaim mengunci pengajuan (MENUNGGU -> status baru) di dalam transaksi; mencegah dua admin memproses bersamaan.
func (s *Service) klaim(ctx context.Context, tx Repository, id uint, status string, actorID uint, catatan string) error {
	berhasil, err := tx.ClaimPendaftaran(ctx, id, status, actorID, strings.TrimSpace(catatan), s.now())
	if err != nil {
		return err
	}
	if !berhasil {
		return ErrStateChanged
	}
	return nil
}

// cekKontak: nama wajib, email atau no HP wajib salah satu (untuk login), format no HP.
func cekKontak(nama, email, noHP string) map[string]string {
	errorPerField := map[string]string{}
	if nama == "" {
		errorPerField["nama"] = "wajib diisi"
	}
	if email == "" && noHP == "" {
		errorPerField["email"] = "isi email atau no. telepon (salah satu) untuk login"
	}
	if noHP != "" && !identitas.NoHPValid(noHP) {
		errorPerField["no_telepon"] = "format tidak valid (contoh: 081234567890 atau +6591234567)"
	}
	return errorPerField
}

func (s *Service) cekKontakBelumDipakai(ctx context.Context, email, noHP string, kecualiUserID uint) error {
	emailDipakai, noHPDipakai, err := s.repo.KontakDipakai(ctx, email, noHP, kecualiUserID)
	if err != nil {
		return apperror.Internal(err)
	}
	if !emailDipakai && !noHPDipakai {
		return nil
	}
	errKonflik := apperror.Conflict("Email atau no. telepon sudah terdaftar")
	errKonflik.Fields = map[string]string{}
	if emailDipakai {
		errKonflik.Fields["email"] = "sudah terdaftar"
	}
	if noHPDipakai {
		errKonflik.Fields["no_telepon"] = "sudah terdaftar"
	}
	return errKonflik
}

func ubahError(err error) error {
	switch {
	case errors.Is(err, ErrStateChanged):
		return apperror.Conflict("Pengajuan sudah diproses admin lain. Muat ulang lalu coba lagi.")
	case errors.Is(err, ErrDuplicate):
		return apperror.Conflict("Kode jemaat bentrok, silakan coba lagi")
	}
	return apperror.Internal(err)
}

func notFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("Pengajuan tidak ditemukan")
	}
	return apperror.Internal(err)
}

func userNotFoundOrInternal(err error) error {
	if errors.Is(err, ErrNotFound) {
		return apperror.Unauthorized("Akun tidak ditemukan")
	}
	return apperror.Internal(err)
}

func toListItem(p *models.Pendaftaran) ListItem {
	item := ListItem{
		ID: p.ID, Status: p.Status, UserID: p.UserID, DiajukanPada: p.CreatedAt, DiperiksaPada: p.DiperiksaPada,
	}
	if p.User != nil { // nama/kontak terbaru dari akun
		item.Nama, item.Email, item.NoTelepon = p.User.Nama, p.User.Email, p.User.NoTelepon
	}
	return item
}

// rapikanNama: "  budi   santoso " -> "budi santoso"
func rapikanNama(nama string) string { return strings.Join(strings.Fields(nama), " ") }

func nilKalauKosong(isi string) *string {
	if isi == "" {
		return nil
	}
	return &isi
}

func teks(isi *string) string {
	if isi == nil {
		return ""
	}
	return *isi
}
