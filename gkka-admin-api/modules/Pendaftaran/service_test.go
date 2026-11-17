package pendaftaran

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/pagination"
	"sim-g-backend/models"
)

var sekarang = time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)

// ---------- fake repository di memori (dengan rollback transaksi) ----------

type fakeRepo struct {
	users        map[uint]models.User
	profil       map[uint]models.ProfilJemaat
	pend         map[uint]models.Pendaftaran
	keahlian     map[uint]models.Keahlian
	userKeahlian []models.UserKeahlian
	nextUser     uint
	nextPend     uint

	noRole          bool
	dupOnCreateUser int // CreateUser pertama sebanyak ini gagal ErrDuplicate
	createUserCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:    map[uint]models.User{},
		profil:   map[uint]models.ProfilJemaat{},
		pend:     map[uint]models.Pendaftaran{},
		keahlian: map[uint]models.Keahlian{1: {ID: 1, Nama: "Usher"}, 2: {ID: 2, Nama: "Pemusik"}},
		nextUser: 100,
		nextPend: 1,
	}
}

func ptr(s string) *string { return &s }

func (r *fakeRepo) tambahUser(u models.User) models.User {
	r.users[u.ID] = u
	return u
}

func (r *fakeRepo) Tx(_ context.Context, fn func(Repository) error) error {
	users, profil, pend := map[uint]models.User{}, map[uint]models.ProfilJemaat{}, map[uint]models.Pendaftaran{}
	for k, v := range r.users {
		users[k] = v
	}
	for k, v := range r.profil {
		profil[k] = v
	}
	for k, v := range r.pend {
		pend[k] = v
	}
	rows := append([]models.UserKeahlian(nil), r.userKeahlian...)
	if err := fn(r); err != nil {
		r.users, r.profil, r.pend, r.userKeahlian = users, profil, pend, rows // rollback
		return err
	}
	return nil
}

func (r *fakeRepo) CreatePendaftaran(_ context.Context, p *models.Pendaftaran) error {
	p.ID = r.nextPend
	r.nextPend++
	r.pend[p.ID] = *p
	return nil
}

func (r *fakeRepo) FindPendaftaran(_ context.Context, id uint) (*models.Pendaftaran, error) {
	p, ok := r.pend[id]
	if !ok {
		return nil, ErrNotFound
	}
	if p.UserID != nil {
		if u, ok := r.users[*p.UserID]; ok {
			p.User = &u
		}
	}
	return &p, nil
}

func (r *fakeRepo) ListPendaftaran(_ context.Context, q ListQuery) ([]models.Pendaftaran, int64, error) {
	var out []models.Pendaftaran
	for _, p := range r.pend {
		if (q.Status == "" || p.Status == q.Status) && (q.Jenis == "" || p.Jenis == q.Jenis) {
			out = append(out, p)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeRepo) PendingExistsForUser(_ context.Context, userID uint) (bool, error) {
	for _, p := range r.pend {
		if p.Status == models.PendaftaranMenunggu && p.UserID != nil && *p.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) PendingContactTaken(_ context.Context, email, phone string) (bool, bool, error) {
	var e, t bool
	for _, p := range r.pend {
		if p.Status != models.PendaftaranMenunggu {
			continue
		}
		var is isian
		_ = json.Unmarshal([]byte(p.DataIsian), &is)
		e = e || (email != "" && is.Email == email)
		t = t || (phone != "" && is.NoTelepon == phone)
	}
	return e, t, nil
}

func (r *fakeRepo) ClaimPendaftaran(_ context.Context, id uint, status string, actorID uint, catatan string, at time.Time) (bool, error) {
	p, ok := r.pend[id]
	if !ok || p.Status != models.PendaftaranMenunggu {
		return false, nil
	}
	p.Status, p.PasswordHash, p.DiperiksaOleh, p.DiperiksaPada = status, nil, &actorID, &at
	if catatan != "" {
		p.CatatanAdmin = &catatan
	}
	r.pend[id] = p
	return true, nil
}

func (r *fakeRepo) LinkPendaftaranUser(_ context.Context, id, userID uint) error {
	p := r.pend[id]
	p.UserID = &userID
	r.pend[id] = p
	return nil
}

func (r *fakeRepo) FindUserByID(_ context.Context, id uint) (*models.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (r *fakeRepo) FindClaimCandidates(_ context.Context, kode, email, phone string) ([]models.User, error) {
	var out []models.User
	for _, u := range r.users {
		if (kode != "" && u.KodeJemaat == kode) ||
			(email != "" && u.Email != nil && *u.Email == email) ||
			(phone != "" && u.NoTelepon != nil && *u.NoTelepon == phone) {
			out = append(out, u)
		}
	}
	return out, nil
}

func (r *fakeRepo) ContactTaken(_ context.Context, email, phone string) (bool, bool, error) {
	var e, t bool
	for _, u := range r.users {
		e = e || (email != "" && u.Email != nil && *u.Email == email)
		t = t || (phone != "" && u.NoTelepon != nil && *u.NoTelepon == phone)
	}
	return e, t, nil
}

func (r *fakeRepo) FindRoleByName(_ context.Context, name string) (*models.Role, error) {
	if r.noRole || name != models.RoleJemaat {
		return nil, ErrNotFound
	}
	return &models.Role{ID: 1, Nama: name}, nil
}

func (r *fakeRepo) CreateUser(_ context.Context, u *models.User) error {
	r.createUserCalls++
	if r.dupOnCreateUser > 0 {
		r.dupOnCreateUser--
		return ErrDuplicate
	}
	u.ID = r.nextUser
	r.nextUser++
	if u.Profil != nil {
		u.Profil.UserID = u.ID
		r.profil[u.ID] = *u.Profil
	}
	r.users[u.ID] = *u
	return nil
}

func (r *fakeRepo) ActivateClaimedUser(_ context.Context, userID uint, hash string, email, phone *string) error {
	u := r.users[userID]
	if u.PasswordHash != nil {
		return ErrStateChanged
	}
	u.PasswordHash, u.IsAktif = &hash, true
	if email != nil {
		u.Email = email
	}
	if phone != nil {
		u.NoTelepon = phone
	}
	r.users[userID] = u
	return nil
}

// UpsertProfil meniru Updates(struct) GORM: hanya field yang terisi yang menimpa data lama.
func (r *fakeRepo) UpsertProfil(_ context.Context, userID uint, p *models.ProfilJemaat) error {
	cur, ada := r.profil[userID]
	if !ada {
		p.UserID = userID
		r.profil[userID] = *p
		return nil
	}
	if p.JenisKelamin != nil {
		cur.JenisKelamin = p.JenisKelamin
	}
	if p.Alamat != nil {
		cur.Alamat = p.Alamat
	}
	if p.TanggalLahir != nil {
		cur.TanggalLahir = p.TanggalLahir
	}
	r.profil[userID] = cur
	return nil
}

func (r *fakeRepo) AddKeahlianSignup(_ context.Context, userID uint, ids []uint) error {
	for _, id := range ids {
		ada := false
		for _, row := range r.userKeahlian {
			ada = ada || (row.UserID == userID && row.KeahlianID == id)
		}
		if !ada {
			f := false
			r.userKeahlian = append(r.userKeahlian, models.UserKeahlian{UserID: userID, KeahlianID: id, Sumber: "SIGNUP", Terverifikasi: &f})
		}
	}
	return nil
}

func (r *fakeRepo) FindKeahlianByIDs(_ context.Context, ids []uint) ([]models.Keahlian, error) {
	var out []models.Keahlian
	for _, id := range ids {
		if k, ok := r.keahlian[id]; ok {
			out = append(out, k)
		}
	}
	return out, nil
}

// ---------- helper tes ----------

func statusOf(err error) int {
	if err == nil {
		return 0
	}
	return apperror.From(err).Status
}

func newSvc(t *testing.T) (*Service, *fakeRepo) {
	t.Helper()
	repo := newFakeRepo()
	svc := NewService(repo)
	svc.now = func() time.Time { return sekarang }
	return svc, repo
}

// jemaat lama yang sudah terdata tetapi belum punya akun login
func jemaatLama(repo *fakeRepo) models.User {
	return repo.tambahUser(models.User{ID: 5, KodeJemaat: "J-005", Nama: "Budi Santoso", RoleID: 1, Role: models.Role{ID: 1, Nama: models.RoleJemaat}})
}

func signupValid() SignupRequest {
	return SignupRequest{
		KodeJemaat: "J-005", Nama: "  budi   santoso ", Email: "Budi@Example.com", NoTelepon: "+62 812-0000-1111",
		Password: "PasswordKuat1", PersetujuanData: true,
		IsianProfil: IsianProfil{JenisKelamin: "L", Alamat: "Surabaya", Keahlian: []uint{2, 2}},
	}
}

func daftarValid() DaftarRequest {
	return DaftarRequest{
		Nama: "Sari  Dewi", Email: "Sari@Example.com", NoTelepon: "0812-3456-7891", Password: "PasswordKuat1", PersetujuanData: true,
		IsianProfil: IsianProfil{
			JenisKelamin: "P", TanggalLahir: "2000-02-29", Alamat: "Surabaya", Kota: "Surabaya",
			StatusKeanggotaan: "SIMPATISAN", Keahlian: []uint{2}, DataTambahan: map[string]any{"hobi": "musik"},
		},
	}
}

func stored(t *testing.T, repo *fakeRepo, id uint) models.Pendaftaran {
	t.Helper()
	p, ok := repo.pend[id]
	if !ok {
		t.Fatalf("pendaftaran %d tidak ada", id)
	}
	return p
}

// ---------- Signup (JEMAAT_LAMA) ----------

func TestSignup_Sukses(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)

	res, err := svc.Signup(context.Background(), signupValid())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != models.PendaftaranMenunggu || res.Jenis != models.PendaftaranJemaatLama {
		t.Errorf("res = %+v", res)
	}
	p := stored(t, repo, res.ID)
	if p.UserID == nil || *p.UserID != 5 || p.Status != models.PendaftaranMenunggu {
		t.Errorf("pendaftaran = %+v", p)
	}
	if !p.PersetujuanDataPada.Equal(sekarang) {
		t.Errorf("waktu persetujuan data = %v, want %v", p.PersetujuanDataPada, sekarang)
	}
	if p.PasswordHash == nil || *p.PasswordHash == "PasswordKuat1" || bcrypt.CompareHashAndPassword([]byte(*p.PasswordHash), []byte("PasswordKuat1")) != nil {
		t.Error("password harus tersimpan sebagai hash yang cocok")
	}
	if strings.Contains(strings.ToLower(p.DataIsian), "password") || strings.Contains(p.DataIsian, "PasswordKuat1") {
		t.Errorf("data_isian tidak boleh memuat password: %s", p.DataIsian)
	}
	is, _ := parseIsian(&p)
	if is.Nama != "budi santoso" || is.Email != "budi@example.com" || is.NoTelepon != "081200001111" || len(is.Keahlian) != 1 {
		t.Errorf("isian = %+v", is)
	}
}

func TestSignup_TidakCocokPesanSama(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)

	tidakAda := signupValid()
	tidakAda.KodeJemaat, tidakAda.Email, tidakAda.NoTelepon = "J-999", "", ""
	namaBeda := signupValid()
	namaBeda.Nama = "Orang Lain"

	var pesan []string
	for _, req := range []SignupRequest{tidakAda, namaBeda} {
		_, err := svc.Signup(context.Background(), req)
		if statusOf(err) != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", statusOf(err))
		}
		pesan = append(pesan, apperror.From(err).Message)
	}
	if pesan[0] != pesan[1] {
		t.Errorf("pesan harus sama agar data jemaat tidak bisa ditebak: %q vs %q", pesan[0], pesan[1])
	}
	if len(repo.pend) != 0 {
		t.Error("tidak boleh ada pendaftaran tersimpan")
	}
}

func TestSignup_SudahAktifDanSudahAdaPendaftaran(t *testing.T) {
	svc, repo := newSvc(t)
	u := jemaatLama(repo)

	// sudah ada pendaftaran menunggu
	if _, err := svc.Signup(context.Background(), signupValid()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Signup(context.Background(), signupValid()); statusOf(err) != http.StatusConflict {
		t.Errorf("pendaftaran ganda: status = %d, want 409", statusOf(err))
	}
	// akun sudah punya password
	u.PasswordHash = ptr("hash")
	repo.users[u.ID] = u
	repo.pend = map[uint]models.Pendaftaran{}
	if _, err := svc.Signup(context.Background(), signupValid()); statusOf(err) != http.StatusConflict {
		t.Errorf("akun sudah aktif: status = %d, want 409", statusOf(err))
	}
}

func TestSignup_DataCocokDenganDuaJemaat(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	repo.tambahUser(models.User{ID: 6, KodeJemaat: "J-006", Nama: "Budi Santoso", Email: ptr("budi@example.com")}) // nama sama, email milik user lain

	if _, err := svc.Signup(context.Background(), signupValid()); statusOf(err) != http.StatusConflict {
		t.Errorf("status = %d, want 409", statusOf(err))
	}
}

func TestSignup_Validasi(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)

	bad := map[string]func(*SignupRequest){
		"tanpa identitas":    func(r *SignupRequest) { r.KodeJemaat, r.Email, r.NoTelepon = "", "", "" },
		"tanpa persetujuan":  func(r *SignupRequest) { r.PersetujuanData = false },
		"nama hanya spasi":   func(r *SignupRequest) { r.Nama = "   " },
		"telepon salah":      func(r *SignupRequest) { r.NoTelepon = "abc" },
		"password 73 byte":   func(r *SignupRequest) { r.Password = strings.Repeat("a", 73) },
		"tanggal masa depan": func(r *SignupRequest) { r.TanggalLahir = "2999-01-01" },
		"tanggal tidak ada":  func(r *SignupRequest) { r.TanggalLahir = "2026-02-31" },
		"keahlian tidak ada": func(r *SignupRequest) { r.Keahlian = []uint{2, 777} },
		"data_tambahan 5 KB": func(r *SignupRequest) { r.DataTambahan = map[string]any{"x": strings.Repeat("a", 5000)} },
	}
	for name, mutate := range bad {
		req := signupValid()
		mutate(&req)
		if _, err := svc.Signup(context.Background(), req); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("%s: status = %d, want 422", name, statusOf(err))
		}
	}
	if len(repo.pend) != 0 {
		t.Error("tidak boleh ada pendaftaran tersimpan saat validasi gagal")
	}
}

// ---------- Daftar (BARU) ----------

func TestDaftar_Sukses(t *testing.T) {
	svc, repo := newSvc(t)
	res, err := svc.Daftar(context.Background(), daftarValid())
	if err != nil {
		t.Fatal(err)
	}
	p := stored(t, repo, res.ID)
	if res.Jenis != models.PendaftaranBaru || p.UserID != nil || p.PasswordHash == nil || p.Status != models.PendaftaranMenunggu {
		t.Errorf("pendaftaran = %+v", p)
	}
	if len(repo.users) != 0 {
		t.Error("user belum boleh dibuat sebelum disetujui")
	}
	is, _ := parseIsian(&p)
	if is.Email != "sari@example.com" || is.NoTelepon != "081234567891" || is.Nama != "Sari Dewi" {
		t.Errorf("isian dinormalisasi = %+v", is)
	}
}

func TestDaftar_TanpaPasswordUntukAnak(t *testing.T) {
	svc, repo := newSvc(t)
	req := DaftarRequest{Nama: "Anak Kecil", PersetujuanData: true, IsianProfil: IsianProfil{JenisKelamin: "L", TanggalLahir: "2018-05-01"}}
	res, err := svc.Daftar(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if p := stored(t, repo, res.ID); p.PasswordHash != nil {
		t.Error("tanpa password, password_hash harus NULL")
	}
}

func TestDaftar_Validasi(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.Daftar(context.Background(), DaftarRequest{Nama: "X", PersetujuanData: true})
	if statusOf(err) != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", statusOf(err))
	}
	f := apperror.From(err).Fields
	if f["jenis_kelamin"] == "" || f["tanggal_lahir"] == "" {
		t.Errorf("fields = %v", f)
	}

	bad := map[string]func(*DaftarRequest){
		"password tanpa email": func(r *DaftarRequest) { r.Email = "" },
		"tanpa persetujuan":    func(r *DaftarRequest) { r.PersetujuanData = false },
		"keahlian tidak ada":   func(r *DaftarRequest) { r.Keahlian = []uint{999} },
	}
	for name, mutate := range bad {
		req := daftarValid()
		mutate(&req)
		if _, err := svc.Daftar(context.Background(), req); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("%s: status = %d, want 422", name, statusOf(err))
		}
	}
}

func TestDaftar_KontakSudahDipakai(t *testing.T) {
	svc, repo := newSvc(t)
	repo.tambahUser(models.User{ID: 9, KodeJemaat: "J-009", Nama: "Lama", Email: ptr("sari@example.com"), NoTelepon: ptr("081234567891")})

	_, err := svc.Daftar(context.Background(), daftarValid())
	if statusOf(err) != http.StatusConflict {
		t.Fatalf("status = %d, want 409", statusOf(err))
	}
	if f := apperror.From(err).Fields; f["email"] == "" || f["no_telepon"] == "" {
		t.Errorf("fields = %v", f)
	}

	// pendaftar kedua dengan kontak yang sama dengan pendaftaran MENUNGGU
	svc2, _ := newSvc(t)
	if _, err := svc2.Daftar(context.Background(), daftarValid()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc2.Daftar(context.Background(), daftarValid()); statusOf(err) != http.StatusConflict {
		t.Errorf("pendaftaran ganda: status = %d, want 409", statusOf(err))
	}
}

// ---------- Setujui ----------

func TestSetujui_Baru(t *testing.T) {
	svc, repo := newSvc(t)
	res, err := svc.Daftar(context.Background(), daftarValid())
	if err != nil {
		t.Fatal(err)
	}
	hash := *stored(t, repo, res.ID).PasswordHash

	out, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{CatatanAdmin: "data lengkap"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != models.PendaftaranDisetujui || out.UserID == nil || len(out.KodeJemaat) != 9 || out.KodeJemaat[0] != 'J' {
		t.Fatalf("out = %+v", out)
	}

	u := repo.users[*out.UserID]
	if !u.IsAktif || u.RoleID != 1 || u.Role.Nama != models.RoleJemaat || u.Nama != "Sari Dewi" {
		t.Errorf("user = %+v", u)
	}
	if u.Email == nil || *u.Email != "sari@example.com" || u.NoTelepon == nil || *u.NoTelepon != "081234567891" {
		t.Errorf("kontak user = %v / %v", u.Email, u.NoTelepon)
	}
	if u.PasswordHash == nil || *u.PasswordHash != hash {
		t.Error("password hash harus dipindahkan ke user")
	}

	pr, ada := repo.profil[u.ID]
	if !ada || pr.JenisKelamin == nil || *pr.JenisKelamin != "P" || pr.Kota == nil || *pr.Kota != "Surabaya" ||
		pr.TanggalLahir == nil || pr.TanggalLahir.Format("2006-01-02") != "2000-02-29" ||
		pr.StatusKeanggotaan == nil || *pr.StatusKeanggotaan != "SIMPATISAN" {
		t.Errorf("profil = %+v", pr)
	}
	if pr.TanggalBergabung == nil || pr.TanggalBergabung.Format("2006-01-02") != "2026-10-27" {
		t.Errorf("tanggal bergabung = %v", pr.TanggalBergabung)
	}
	if pr.DataTambahan == nil || !strings.Contains(*pr.DataTambahan, "musik") {
		t.Errorf("data_tambahan = %v", pr.DataTambahan)
	}

	if len(repo.userKeahlian) != 1 {
		t.Fatalf("user_keahlian = %+v", repo.userKeahlian)
	}
	uk := repo.userKeahlian[0]
	if uk.UserID != u.ID || uk.KeahlianID != 2 || uk.Sumber != "SIGNUP" || uk.Terverifikasi == nil || *uk.Terverifikasi {
		t.Errorf("keahlian signup harus SIGNUP dan belum terverifikasi: %+v", uk)
	}

	p := stored(t, repo, res.ID)
	if p.Status != models.PendaftaranDisetujui || p.PasswordHash != nil || p.UserID == nil || *p.UserID != u.ID ||
		p.DiperiksaOleh == nil || *p.DiperiksaOleh != 1 || p.CatatanAdmin == nil || *p.CatatanAdmin != "data lengkap" {
		t.Errorf("pendaftaran = %+v", p)
	}

	// tidak bisa diproses dua kali
	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); statusOf(err) != http.StatusConflict {
		t.Errorf("setujui ulang: status = %d, want 409", statusOf(err))
	}
}

func TestSetujui_BaruTanpaPassword(t *testing.T) {
	svc, repo := newSvc(t)
	res, err := svc.Daftar(context.Background(), DaftarRequest{Nama: "Anak Kecil", PersetujuanData: true, IsianProfil: IsianProfil{JenisKelamin: "L", TanggalLahir: "2018-05-01"}})
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{})
	if err != nil {
		t.Fatal(err)
	}
	u := repo.users[*out.UserID]
	if u.PasswordHash != nil || !u.IsAktif || u.Email != nil || u.NoTelepon != nil {
		t.Errorf("jemaat tanpa akun login harus tanpa password dan kontak: %+v", u)
	}
}

func TestSetujui_BaruKontakBentrokSetelahDaftar(t *testing.T) {
	svc, repo := newSvc(t)
	res, _ := svc.Daftar(context.Background(), daftarValid())
	repo.tambahUser(models.User{ID: 9, KodeJemaat: "J-009", Nama: "Penyerobot", Email: ptr("sari@example.com")}) // dibuat setelah formulir masuk

	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); statusOf(err) != http.StatusConflict {
		t.Errorf("status = %d, want 409", statusOf(err))
	}
	if stored(t, repo, res.ID).Status != models.PendaftaranMenunggu {
		t.Error("pendaftaran harus tetap MENUNGGU")
	}
}

func TestSetujui_BaruKodeBentrokDiulangDanRollback(t *testing.T) {
	svc, repo := newSvc(t)
	res, _ := svc.Daftar(context.Background(), daftarValid())

	repo.dupOnCreateUser = 2 // dua kode pertama bentrok, ketiga berhasil
	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); err != nil {
		t.Fatalf("harus berhasil setelah mengulang: %v", err)
	}
	if repo.createUserCalls != 3 || len(repo.users) != 1 || stored(t, repo, res.ID).Status != models.PendaftaranDisetujui {
		t.Errorf("calls=%d users=%d status=%s", repo.createUserCalls, len(repo.users), stored(t, repo, res.ID).Status)
	}

	// selalu bentrok: seluruh transaksi di-rollback, pendaftaran tetap MENUNGGU dan password tidak hilang
	svc2, repo2 := newSvc(t)
	res2, _ := svc2.Daftar(context.Background(), daftarValid())
	repo2.dupOnCreateUser = 99
	if _, err := svc2.Setujui(context.Background(), 1, res2.ID, SetujuiRequest{}); statusOf(err) != http.StatusConflict {
		t.Errorf("status = %d, want 409", statusOf(err))
	}
	p := stored(t, repo2, res2.ID)
	if p.Status != models.PendaftaranMenunggu || p.PasswordHash == nil || len(repo2.users) != 0 || len(repo2.userKeahlian) != 0 {
		t.Errorf("rollback gagal: %+v users=%d", p, len(repo2.users))
	}
	if repo2.createUserCalls != maxKodeTry {
		t.Errorf("createUserCalls = %d, want %d", repo2.createUserCalls, maxKodeTry)
	}
}

func TestSetujui_BaruRoleBelumAda(t *testing.T) {
	svc, repo := newSvc(t)
	res, _ := svc.Daftar(context.Background(), daftarValid())
	repo.noRole = true
	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); statusOf(err) != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (seed belum dijalankan)", statusOf(err))
	}
}

func TestSetujui_Lama(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	repo.profil[5] = models.ProfilJemaat{UserID: 5, Alamat: ptr("Alamat Lama"), Kota: ptr("Malang")}

	res, err := svc.Signup(context.Background(), signupValid())
	if err != nil {
		t.Fatal(err)
	}
	hash := *stored(t, repo, res.ID).PasswordHash

	out, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if out.UserID == nil || *out.UserID != 5 || out.KodeJemaat != "J-005" {
		t.Errorf("out = %+v", out)
	}
	u := repo.users[5]
	if !u.IsAktif || u.PasswordHash == nil || *u.PasswordHash != hash {
		t.Errorf("akun harus aktif dengan password dari formulir: %+v", u)
	}
	if u.Email == nil || *u.Email != "budi@example.com" || u.NoTelepon == nil {
		t.Errorf("email/telepon yang masih kosong harus terisi: %v / %v", u.Email, u.NoTelepon)
	}
	if pr := repo.profil[5]; pr.JenisKelamin == nil || *pr.JenisKelamin != "L" || pr.Alamat == nil || *pr.Alamat != "Surabaya" || pr.Kota == nil || *pr.Kota != "Malang" {
		t.Errorf("profil: isian baru menimpa, data lama yang tidak diisi tetap: %+v", pr)
	}
	if len(repo.userKeahlian) != 1 || repo.userKeahlian[0].Sumber != "SIGNUP" {
		t.Errorf("user_keahlian = %+v", repo.userKeahlian)
	}
	if p := stored(t, repo, res.ID); p.Status != models.PendaftaranDisetujui || p.PasswordHash != nil {
		t.Errorf("pendaftaran = %+v", p)
	}
}

func TestSetujui_LamaKontakSudahDipakaiTidakDitimpa(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	repo.tambahUser(models.User{ID: 6, KodeJemaat: "J-006", Nama: "Orang Lain", Email: ptr("budi@example.com")})
	req := signupValid()
	req.Email = ""
	req.NoTelepon = "081200001111"
	res, err := svc.Signup(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// email ditambahkan ke isian secara manual untuk meniru formulir yang memuat email orang lain
	p := repo.pend[res.ID]
	is, _ := parseIsian(&p)
	is.Email = "budi@example.com"
	raw, _ := json.Marshal(is)
	p.DataIsian = string(raw)
	repo.pend[res.ID] = p

	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); err != nil {
		t.Fatal(err)
	}
	if u := repo.users[5]; u.Email != nil {
		t.Errorf("email milik user lain tidak boleh dipakai: %v", *u.Email)
	}
}

func TestSetujui_LamaAkunSudahAktifSetelahSignup(t *testing.T) {
	svc, repo := newSvc(t)
	u := jemaatLama(repo)
	res, _ := svc.Signup(context.Background(), signupValid())
	u.PasswordHash = ptr("sudah-aktif")
	repo.users[u.ID] = u

	if _, err := svc.Setujui(context.Background(), 1, res.ID, SetujuiRequest{}); statusOf(err) != http.StatusConflict {
		t.Errorf("status = %d, want 409", statusOf(err))
	}
	if stored(t, repo, res.ID).Status != models.PendaftaranMenunggu {
		t.Error("pendaftaran harus tetap MENUNGGU")
	}
}

func TestSetujui_TidakAda(t *testing.T) {
	svc, _ := newSvc(t)
	if _, err := svc.Setujui(context.Background(), 1, 404, SetujuiRequest{}); statusOf(err) != http.StatusNotFound {
		t.Errorf("status = %d, want 404", statusOf(err))
	}
}

// ---------- Tolak dan Duplikat ----------

func TestTolak(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	res, _ := svc.Signup(context.Background(), signupValid())

	if _, err := svc.Tolak(context.Background(), 1, res.ID, TolakRequest{CatatanAdmin: "   "}); statusOf(err) != http.StatusUnprocessableEntity {
		t.Errorf("tanpa alasan: status = %d, want 422", statusOf(err))
	}
	out, err := svc.Tolak(context.Background(), 1, res.ID, TolakRequest{CatatanAdmin: "data tidak sesuai"})
	if err != nil || out.Status != models.PendaftaranDitolak {
		t.Fatalf("out = %+v err = %v", out, err)
	}
	p := stored(t, repo, res.ID)
	if p.PasswordHash != nil || p.CatatanAdmin == nil || *p.CatatanAdmin != "data tidak sesuai" {
		t.Errorf("pendaftaran = %+v", p)
	}
	if repo.users[5].PasswordHash != nil || repo.users[5].IsAktif {
		t.Error("akun jemaat tidak boleh berubah saat ditolak")
	}
	if _, err := svc.Tolak(context.Background(), 1, res.ID, TolakRequest{CatatanAdmin: "lagi"}); statusOf(err) != http.StatusConflict {
		t.Errorf("tolak ulang: status = %d, want 409", statusOf(err))
	}
}

func TestTandaiDuplikat(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	res, _ := svc.Daftar(context.Background(), daftarValid())

	if _, err := svc.TandaiDuplikat(context.Background(), 1, res.ID, DuplikatRequest{UserID: 404}); statusOf(err) != http.StatusUnprocessableEntity {
		t.Errorf("jemaat tidak ada: status = %d, want 422", statusOf(err))
	}
	out, err := svc.TandaiDuplikat(context.Background(), 1, res.ID, DuplikatRequest{UserID: 5, CatatanAdmin: "sudah terdata"})
	if err != nil || out.Status != models.PendaftaranDuplikat || out.KodeJemaat != "J-005" {
		t.Fatalf("out = %+v err = %v", out, err)
	}
	p := stored(t, repo, res.ID)
	if p.Status != models.PendaftaranDuplikat || p.UserID == nil || *p.UserID != 5 || p.PasswordHash != nil {
		t.Errorf("pendaftaran = %+v", p)
	}
	if len(repo.users) != 1 {
		t.Error("tidak boleh membuat user baru")
	}
}

// ---------- List dan Get ----------

func TestGet_TidakMenampilkanPassword(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	res, _ := svc.Signup(context.Background(), signupValid())

	d, err := svc.Get(context.Background(), res.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(d)
	if strings.Contains(strings.ToLower(string(raw)), "password") || strings.Contains(string(raw), "$fake$") {
		t.Errorf("detail tidak boleh memuat password: %s", raw)
	}
	if d.User == nil || d.User.KodeJemaat != "J-005" || d.User.AkunAktif {
		t.Errorf("user ringkas = %+v", d.User)
	}
	if len(d.Keahlian) != 1 || d.Keahlian[0].Nama != "Pemusik" {
		t.Errorf("keahlian = %+v", d.Keahlian)
	}
	if _, err := svc.Get(context.Background(), 404); statusOf(err) != http.StatusNotFound {
		t.Errorf("status = %d, want 404", statusOf(err))
	}
}

func TestList(t *testing.T) {
	svc, repo := newSvc(t)
	jemaatLama(repo)
	a, _ := svc.Signup(context.Background(), signupValid())
	_, _ = svc.Daftar(context.Background(), daftarValid())
	_, _ = svc.Tolak(context.Background(), 1, a.ID, TolakRequest{CatatanAdmin: "x"})

	q := ListQuery{Params: pagination.Params{Page: 1, Limit: 20}}
	all, err := svc.List(context.Background(), q)
	if err != nil || len(all.Items) != 2 {
		t.Fatalf("all = %+v err = %v", all, err)
	}
	q.Status = models.PendaftaranMenunggu
	menunggu, _ := svc.List(context.Background(), q)
	if len(menunggu.Items) != 1 || menunggu.Items[0].Jenis != models.PendaftaranBaru || menunggu.Items[0].Nama != "Sari Dewi" {
		t.Errorf("menunggu = %+v", menunggu.Items)
	}
	q.Status = "SALAH"
	if _, err := svc.List(context.Background(), q); statusOf(err) != http.StatusBadRequest {
		t.Errorf("status salah: status = %d, want 400", statusOf(err))
	}
}

// ---------- fungsi kecil ----------

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"081234567890": "081234567890", "0812-3456-7890": "081234567890", "+62 812 3456 7890": "081234567890",
		"6281234567890": "081234567890", "(0812) 3456.7890": "081234567890", "": "", "  ": "",
	}
	for in, want := range cases {
		if got := normalizePhone(in); got != want {
			t.Errorf("normalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerateKode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		k, err := generateKode()
		if err != nil || len(k) != 9 || k[0] != 'J' {
			t.Fatalf("kode = %q err = %v", k, err)
		}
		for _, ch := range k[1:] {
			if !strings.ContainsRune(kodeAlphabet, ch) {
				t.Fatalf("karakter %q di luar alfabet pada %q", ch, k)
			}
		}
		seen[k] = true
	}
	if len(seen) < 190 {
		t.Errorf("hanya %d kode unik dari 200", len(seen))
	}
}

func TestProfilKosongDanRapikan(t *testing.T) {
	if !profilKosong(IsianProfil{Keahlian: []uint{1}}) {
		t.Error("keahlian saja tidak dihitung sebagai isian profil")
	}
	if profilKosong(IsianProfil{Kota: "Surabaya"}) {
		t.Error("kota terisi = profil tidak kosong")
	}
	if profilKosong(IsianProfil{DataTambahan: map[string]any{"a": 1}}) {
		t.Error("data_tambahan terisi = profil tidak kosong")
	}
	r := rapikanProfil(IsianProfil{Kota: "  Surabaya ", Keahlian: []uint{3, 3, 1}})
	if r.Kota != "Surabaya" || len(r.Keahlian) != 2 {
		t.Errorf("r = %+v", r)
	}
	if !samaNama("  budi   SANTOSO", "Budi Santoso") || samaNama("Budi", "Budi Santoso") {
		t.Error("samaNama salah")
	}
}
