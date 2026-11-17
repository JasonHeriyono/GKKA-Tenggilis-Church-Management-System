package schedule

import (
	"context"
	"net/http"
	"testing"
	"time"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

// fakeRepo hanya berisi data di memori, supaya service dites tanpa database.
type fakeRepo struct {
	kegiatan   map[uint]*models.Kegiatan
	users      map[uint]*models.User
	keahlian   map[uint]*models.Keahlian
	userSkills map[[2]uint]bool // {user, keahlian} -> terverifikasi
	lainnya    []models.JadwalPelayanan
	tersimpan  map[uint]*models.JadwalPelayanan
	dupOnNext  bool
}

func jamPtr(s string) *string { return &s }

func newFakeRepo() *fakeRepo {
	tanggal := time.Date(2026, 11, 1, 0, 0, 0, 0, wib)
	return &fakeRepo{
		kegiatan: map[uint]*models.Kegiatan{
			1: {ID: 1, Nama: "Ibadah Umum", Tanggal: tanggal, WaktuMulai: "08:00:00", WaktuSelesai: jamPtr("10:00:00")},
		},
		users: map[uint]*models.User{
			2: {ID: 2, KodeJemaat: "J-002", Nama: "Sari", IsAktif: true, Role: models.Role{Nama: models.RoleJemaat}},
			3: {ID: 3, KodeJemaat: "G-003", Nama: "Tamu", IsAktif: true, Role: models.Role{Nama: models.RoleGuest}},
			4: {ID: 4, KodeJemaat: "J-004", Nama: "Budi", IsAktif: false, Role: models.Role{Nama: models.RoleJemaat}},
		},
		keahlian:   map[uint]*models.Keahlian{5: {ID: 5, Nama: "Singer"}},
		userSkills: map[[2]uint]bool{{2, 5}: true},
		tersimpan:  map[uint]*models.JadwalPelayanan{},
	}
}

func (r *fakeRepo) ListKegiatan(context.Context, KegiatanQuery) ([]models.Kegiatan, int64, error) {
	return nil, 0, nil
}
func (r *fakeRepo) FindKegiatan(_ context.Context, id uint) (*models.Kegiatan, error) {
	if k, ok := r.kegiatan[id]; ok {
		return k, nil
	}
	return nil, ErrNotFound
}
func (r *fakeRepo) CreateKegiatan(context.Context, *models.Kegiatan) error { return nil }
func (r *fakeRepo) UpdateKegiatan(context.Context, *models.Kegiatan) error { return nil }
func (r *fakeRepo) DeleteKegiatan(context.Context, uint) error             { return nil }
func (r *fakeRepo) KomunitasAda(context.Context, uint) (bool, error)       { return true, nil }

func (r *fakeRepo) FindUser(_ context.Context, id uint) (*models.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}
func (r *fakeRepo) FindKeahlian(_ context.Context, id uint) (*models.Keahlian, error) {
	if k, ok := r.keahlian[id]; ok {
		return k, nil
	}
	return nil, ErrNotFound
}
func (r *fakeRepo) UserPunyaKeahlian(_ context.Context, userID, keahlianID uint) (bool, error) {
	return r.userSkills[[2]uint{userID, keahlianID}], nil
}
func (r *fakeRepo) CreatePenugasan(_ context.Context, p *models.JadwalPelayanan) error {
	if r.dupOnNext {
		return ErrDuplicate
	}
	p.ID = 77
	r.tersimpan[p.ID] = p
	return nil
}
func (r *fakeRepo) FindPenugasan(_ context.Context, kegiatanID, id uint) (*models.JadwalPelayanan, error) {
	p, ok := r.tersimpan[id]
	if !ok || p.KegiatanID != kegiatanID {
		return nil, ErrNotFound
	}
	hasil := *p
	hasil.User = *r.users[p.UserID]
	hasil.Keahlian = *r.keahlian[p.KeahlianID]
	return &hasil, nil
}
func (r *fakeRepo) ListPetugasKegiatan(context.Context, uint) ([]models.JadwalPelayanan, error) {
	return nil, nil
}
func (r *fakeRepo) DeletePenugasan(_ context.Context, kegiatanID, id uint) error {
	p, ok := r.tersimpan[id]
	if !ok || p.KegiatanID != kegiatanID {
		return ErrNotFound
	}
	delete(r.tersimpan, id)
	return nil
}
func (r *fakeRepo) ListPenugasanUserPadaTanggal(context.Context, uint, string, uint) ([]models.JadwalPelayanan, error) {
	return r.lainnya, nil
}
func (r *fakeRepo) ListJadwalUser(context.Context, uint, JadwalSayaQuery) ([]models.JadwalPelayanan, int64, error) {
	return nil, 0, nil
}

func statusErr(err error) int { return apperror.From(err).Status }

// penugasanLain: tugas jemaat 2 di kegiatan lain pada tanggal yang sama.
func penugasanLain(mulai string, selesai *string) models.JadwalPelayanan {
	return models.JadwalPelayanan{
		ID: 9, KegiatanID: 8, UserID: 2, KeahlianID: 5,
		Kegiatan: models.Kegiatan{
			ID: 8, Nama: "Latihan Paduan Suara", Tanggal: time.Date(2026, 11, 1, 0, 0, 0, 0, wib),
			WaktuMulai: mulai, WaktuSelesai: selesai,
		},
	}
}

func (r *fakeRepo) FindKomunitasByNama(_ context.Context, _ string) (*models.Komunitas, error) {
	return nil, ErrNotFound
}

func (r *fakeRepo) ImportKegiatan(_ context.Context, _ []models.Kegiatan, _ []models.Kegiatan) error {
	return nil
}

func (r *fakeRepo) FindKegiatanByUnik(_ context.Context, _, _, _ string) (*models.Kegiatan, error) {
	return nil, ErrNotFound
}

func TestTugaskanPetugas_Berhasil(t *testing.T) {
	svc := NewService(newFakeRepo(), wib, nil)

	got, err := svc.TugaskanPetugas(context.Background(), 1, PenugasanRequest{UserID: 2, KeahlianID: 5})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.ID != 77 || got.KegiatanID != 1 || got.UserID != 2 || got.KeahlianID != 5 {
		t.Errorf("response salah: %+v", got)
	}
	if got.Nama != "Sari" || got.NamaKeahlian != "Singer" || got.KodeJemaat != "J-002" {
		t.Errorf("nama jemaat/keahlian tidak terisi: %+v", got)
	}
	if got.Status != models.PenugasanTerkonfirmasi {
		t.Errorf("status = %q, want %q", got.Status, models.PenugasanTerkonfirmasi)
	}
	if len(got.Peringatan) != 0 {
		t.Errorf("peringatan = %v, want kosong", got.Peringatan)
	}
}

func TestTugaskanPetugas_KegiatanTidakAda(t *testing.T) {
	svc := NewService(newFakeRepo(), wib, nil)
	_, err := svc.TugaskanPetugas(context.Background(), 99, PenugasanRequest{UserID: 2, KeahlianID: 5})
	if statusErr(err) != http.StatusNotFound {
		t.Errorf("status = %d, want 404", statusErr(err))
	}
}

func TestTugaskanPetugas_Validasi(t *testing.T) {
	tests := []struct {
		nama  string
		req   PenugasanRequest
		field string
	}{
		{"jemaat tidak ada", PenugasanRequest{UserID: 99, KeahlianID: 5}, "id_user"},
		{"guest belum jemaat", PenugasanRequest{UserID: 3, KeahlianID: 5}, "id_user"},
		{"akun tidak aktif", PenugasanRequest{UserID: 4, KeahlianID: 5}, "id_user"},
		{"keahlian tidak ada", PenugasanRequest{UserID: 2, KeahlianID: 99}, "id_keahlian"},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			svc := NewService(newFakeRepo(), wib, nil)
			_, err := svc.TugaskanPetugas(context.Background(), 1, tc.req)
			if statusErr(err) != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", statusErr(err))
			}
			if _, ada := apperror.From(err).Fields[tc.field]; !ada {
				t.Errorf("fields = %v, want ada %q", apperror.From(err).Fields, tc.field)
			}
		})
	}
}

func TestTugaskanPetugas_TanpaKeahlianAtauBelumTerverifikasi(t *testing.T) {
	repo := newFakeRepo()
	repo.keahlian[6] = &models.Keahlian{ID: 6, Nama: "Pemusik"} // jemaat 2 tidak punya (atau belum diverifikasi)
	svc := NewService(repo, wib, nil)

	_, err := svc.TugaskanPetugas(context.Background(), 1, PenugasanRequest{UserID: 2, KeahlianID: 6})
	if statusErr(err) != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", statusErr(err))
	}
	if _, ada := apperror.From(err).Fields["id_keahlian"]; !ada {
		t.Errorf("fields = %v, want ada id_keahlian", apperror.From(err).Fields)
	}
	if len(repo.tersimpan) != 0 {
		t.Error("penugasan tidak boleh tersimpan")
	}
}

func TestTugaskanPetugas_Duplikat(t *testing.T) {
	repo := newFakeRepo()
	repo.dupOnNext = true
	svc := NewService(repo, wib, nil)

	_, err := svc.TugaskanPetugas(context.Background(), 1, PenugasanRequest{UserID: 2, KeahlianID: 5})
	if statusErr(err) != http.StatusConflict {
		t.Errorf("status = %d, want 409", statusErr(err))
	}
}

func TestTugaskanPetugas_BentrokWaktu(t *testing.T) {
	tests := []struct {
		nama        string
		lain        models.JadwalPelayanan
		wantWarning int
	}{
		{"tumpang tindih -> peringatan", penugasanLain("07:00:00", jamPtr("09:00:00")), 1},
		{"berurutan tidak bentrok", penugasanLain("06:00:00", jamPtr("08:00:00")), 0},
		{"jam berbeda jauh tidak bentrok", penugasanLain("12:00:00", jamPtr("13:00:00")), 0},
		{"tanpa jam selesai jatuh di dalam kegiatan", penugasanLain("09:00:00", nil), 1},
		{"tanpa jam selesai di luar kegiatan", penugasanLain("11:00:00", nil), 0},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			repo := newFakeRepo()
			repo.lainnya = []models.JadwalPelayanan{tc.lain}
			svc := NewService(repo, wib, nil)

			got, err := svc.TugaskanPetugas(context.Background(), 1, PenugasanRequest{UserID: 2, KeahlianID: 5})
			if err != nil {
				t.Fatalf("err = %v (bentrok hanya peringatan, penugasan harus tetap tersimpan)", err)
			}
			if len(got.Peringatan) != tc.wantWarning {
				t.Errorf("peringatan = %v, want %d", got.Peringatan, tc.wantWarning)
			}
			if len(repo.tersimpan) != 1 {
				t.Errorf("tersimpan = %d, want 1", len(repo.tersimpan))
			}
		})
	}
}

func TestHapusPetugas(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, wib, nil)
	if _, err := svc.TugaskanPetugas(context.Background(), 1, PenugasanRequest{UserID: 2, KeahlianID: 5}); err != nil {
		t.Fatal(err)
	}

	if err := svc.HapusPetugas(context.Background(), 2, 77); statusErr(err) != http.StatusNotFound {
		t.Errorf("kegiatan salah: status = %d, want 404", statusErr(err))
	}
	if err := svc.HapusPetugas(context.Background(), 1, 77); err != nil {
		t.Errorf("hapus: err = %v", err)
	}
	if err := svc.HapusPetugas(context.Background(), 1, 77); statusErr(err) != http.StatusNotFound {
		t.Errorf("hapus kedua kali: status = %d, want 404", statusErr(err))
	}
}

func TestJadwalSaya_FormatTanggalSalah(t *testing.T) {
	svc := NewService(newFakeRepo(), wib, nil)
	_, err := svc.JadwalSaya(context.Background(), 2, JadwalSayaQuery{Dari: "01-11-2026"})
	if statusErr(err) != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", statusErr(err))
	}
}
