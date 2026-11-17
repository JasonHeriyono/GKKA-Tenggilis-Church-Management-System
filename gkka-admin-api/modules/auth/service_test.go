package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

const testPassword = "Rahasia123"

// ---- fake repository (di memori) ----

type fakeRepo struct {
	users  map[uint]*models.User
	tokens map[string]*models.RefreshToken // kunci: hash token
	nextID uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[uint]*models.User{}, tokens: map[string]*models.RefreshToken{}, nextID: 1}
}

func (r *fakeRepo) FindUserByIdentifier(_ context.Context, id string) (*models.User, error) {
	for _, u := range r.users {
		if u.KodeJemaat == id || (u.Email != nil && *u.Email == id) || (u.NoTelepon != nil && *u.NoTelepon == id) {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (r *fakeRepo) FindUserByID(_ context.Context, id uint) (*models.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (r *fakeRepo) ListKeahlianUser(_ context.Context, _ uint) ([]models.Keahlian, error) {
	return nil, nil
}

func (r *fakeRepo) CreateRefreshToken(_ context.Context, t *models.RefreshToken) error {
	t.ID = r.nextID
	r.nextID++
	r.tokens[t.TokenHash] = t
	return nil
}

func (r *fakeRepo) FindRefreshToken(_ context.Context, hash string) (*models.RefreshToken, error) {
	if t, ok := r.tokens[hash]; ok {
		return t, nil
	}
	return nil, ErrNotFound
}

func (r *fakeRepo) RotateRefreshToken(ctx context.Context, oldID uint, next *models.RefreshToken) (bool, error) {
	for _, t := range r.tokens {
		if t.ID == oldID {
			if t.RevokedAt != nil {
				return false, nil
			}
			now := time.Now()
			t.RevokedAt = &now
			return true, r.CreateRefreshToken(ctx, next)
		}
	}
	return false, nil
}

func (r *fakeRepo) RevokeRefreshToken(_ context.Context, hash string) error {
	if t, ok := r.tokens[hash]; ok && t.RevokedAt == nil {
		now := time.Now()
		t.RevokedAt = &now
	}
	return nil
}

func (r *fakeRepo) RevokeAllUserTokens(_ context.Context, userID uint) error {
	now := time.Now()
	for _, t := range r.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &now
		}
	}
	return nil
}

func (r *fakeRepo) activeTokens(userID uint) int {
	n := 0
	for _, t := range r.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			n++
		}
	}
	return n
}

// ---- helper ----

type env struct {
	svc   *Service
	repo  *fakeRepo
	user  *models.User
	clock time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	h := string(hash)
	email, tel := "budi@example.com", "081234567890"
	u := &models.User{
		ID: 7, KodeJemaat: "J-007", Nama: "Budi", Email: &email, NoTelepon: &tel,
		PasswordHash: &h, IsAktif: true, RoleID: 3, Role: models.Role{ID: 3, Nama: models.RoleSuperAdmin},
	}
	repo := newFakeRepo()
	repo.users[u.ID] = u

	e := &env{repo: repo, user: u, clock: time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)}
	e.svc = NewService(repo, "rahasia-test-rahasia-test-rahasia-test", 15*time.Minute, 7*24*time.Hour)
	e.svc.now = func() time.Time { return e.clock }
	return e
}

func (e *env) login(t *testing.T) *LoginResponse {
	t.Helper()
	res, err := e.svc.Login(context.Background(), LoginRequest{Identifier: "budi@example.com", Password: testPassword})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return res
}

func statusOf(err error) int {
	if err == nil {
		return 0
	}
	return apperror.From(err).Status
}

// ---- Login ----

func TestLogin_Sukses_SemuaJenisIdentifier(t *testing.T) {
	e := newEnv(t)
	for _, id := range []string{"budi@example.com", "J-007", "081234567890", "  J-007  "} {
		res, err := e.svc.Login(context.Background(), LoginRequest{Identifier: id, Password: testPassword})
		if err != nil {
			t.Fatalf("login %q: %v", id, err)
		}
		if res.User.ID != 7 || res.User.Role != models.RoleSuperAdmin || res.TokenType != "Bearer" || res.ExpiresIn != 900 {
			t.Errorf("response = %+v", res)
		}
		u, err := e.svc.ParseAccessToken(res.AccessToken)
		if err != nil || u.ID != 7 || u.Role != models.RoleSuperAdmin {
			t.Errorf("parse = %+v, %v", u, err)
		}
	}
}

func TestLogin_RefreshTokenDisimpanSebagaiHash(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	if _, plain := e.repo.tokens[res.RefreshToken]; plain {
		t.Fatal("refresh token mentah tidak boleh disimpan")
	}
	if _, ok := e.repo.tokens[hashToken(res.RefreshToken)]; !ok {
		t.Fatal("hash refresh token tidak tersimpan")
	}
}

func TestLogin_Gagal_PesanSamaUntukSemuaKasus(t *testing.T) {
	e := newEnv(t)
	noPass := &models.User{ID: 8, KodeJemaat: "J-008", Nama: "Belum Aktif", IsAktif: true, Role: models.Role{Nama: models.RoleJemaat}}
	e.repo.users[8] = noPass

	cases := map[string]LoginRequest{
		"password salah":     {Identifier: "J-007", Password: "salah-salah"},
		"user tidak ada":     {Identifier: "tidak-ada", Password: testPassword},
		"akun belum aktif":   {Identifier: "J-008", Password: testPassword},
		"password kosong db": {Identifier: "J-008", Password: ""},
	}
	var msg string
	for name, req := range cases {
		_, err := e.svc.Login(context.Background(), req)
		if statusOf(err) != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, statusOf(err))
			continue
		}
		if m := apperror.From(err).Message; msg == "" {
			msg = m
		} else if m != msg {
			t.Errorf("%s: pesan berbeda %q vs %q (membocorkan keberadaan akun)", name, m, msg)
		}
	}
}

func TestLogin_AkunNonaktif(t *testing.T) {
	e := newEnv(t)
	e.user.IsAktif = false
	_, err := e.svc.Login(context.Background(), LoginRequest{Identifier: "J-007", Password: testPassword})
	if statusOf(err) != http.StatusForbidden {
		t.Errorf("status = %d, want 403", statusOf(err))
	}
	// password salah pada akun nonaktif tetap 401 (tidak membocorkan status akun)
	_, err = e.svc.Login(context.Background(), LoginRequest{Identifier: "J-007", Password: "salah-salah"})
	if statusOf(err) != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", statusOf(err))
	}
}

// ---- Refresh ----

func TestRefresh_RotasiDanDeteksiPemakaianUlang(t *testing.T) {
	e := newEnv(t)
	first := e.login(t)

	second, err := e.svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.RefreshToken == first.RefreshToken || second.AccessToken == "" {
		t.Fatal("token harus berganti")
	}
	if _, err := e.svc.ParseAccessToken(second.AccessToken); err != nil {
		t.Errorf("access token baru tidak valid: %v", err)
	}
	if n := e.repo.activeTokens(7); n != 1 {
		t.Errorf("token aktif = %d, want 1 (token lama harus dicabut)", n)
	}

	// token lama dipakai lagi => dianggap bocor, semua sesi dicabut
	if _, err := e.svc.Refresh(context.Background(), first.RefreshToken); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("pemakaian ulang: status = %d, want 401", statusOf(err))
	}
	if n := e.repo.activeTokens(7); n != 0 {
		t.Errorf("token aktif = %d, want 0 setelah pemakaian ulang", n)
	}
	if _, err := e.svc.Refresh(context.Background(), second.RefreshToken); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("token baru harus ikut dicabut, status = %d", statusOf(err))
	}
}

func TestRefresh_TokenTidakDikenal(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.Refresh(context.Background(), "ngawur"); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", statusOf(err))
	}
}

func TestRefresh_Kedaluwarsa(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	e.clock = e.clock.Add(7*24*time.Hour + time.Second)
	if _, err := e.svc.Refresh(context.Background(), res.RefreshToken); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", statusOf(err))
	}
}

func TestRefresh_MemakaiRoleTerbaru(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	e.user.Role = models.Role{ID: 2, Nama: models.RoleMajelis} // role diubah Super Admin
	next, err := e.svc.Refresh(context.Background(), res.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := e.svc.ParseAccessToken(next.AccessToken)
	if u.Role != models.RoleMajelis {
		t.Errorf("role = %q, want %q", u.Role, models.RoleMajelis)
	}
}

func TestRefresh_AkunDinonaktifkan(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	e.user.IsAktif = false
	if _, err := e.svc.Refresh(context.Background(), res.RefreshToken); statusOf(err) != http.StatusForbidden {
		t.Errorf("status = %d, want 403", statusOf(err))
	}
	if n := e.repo.activeTokens(7); n != 0 {
		t.Errorf("token aktif = %d, want 0", n)
	}
}

// ---- Logout ----

func TestLogout(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	if err := e.svc.Logout(context.Background(), res.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Refresh(context.Background(), res.RefreshToken); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("setelah logout: status = %d, want 401", statusOf(err))
	}
	if err := e.svc.Logout(context.Background(), "token-tidak-ada"); err != nil {
		t.Errorf("logout token tidak dikenal harus tetap sukses, got %v", err)
	}
}

// ---- Me ----

func TestMe(t *testing.T) {
	e := newEnv(t)
	info, err := e.svc.Me(context.Background(), 7)
	if err != nil || info.Nama != "Budi" || info.Role != models.RoleSuperAdmin {
		t.Errorf("info = %+v, err = %v", info, err)
	}
	if _, err := e.svc.Me(context.Background(), 999); statusOf(err) != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", statusOf(err))
	}
}

// ---- ParseAccessToken ----

func TestParseAccessToken_Kedaluwarsa(t *testing.T) {
	e := newEnv(t)
	res := e.login(t)
	e.clock = e.clock.Add(15*time.Minute + time.Second)
	if _, err := e.svc.ParseAccessToken(res.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseAccessToken_Ditolak(t *testing.T) {
	e := newEnv(t)
	valid := e.login(t).AccessToken

	other := NewService(newFakeRepo(), "secret-lain-secret-lain-secret-lain!!", time.Minute, time.Hour)
	other.now = e.svc.now
	otherToken, _ := other.signAccess(e.user)

	wrongIssuer, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: models.RoleSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "pihak-lain", Subject: "7", ExpiresAt: jwt.NewNumericDate(e.clock.Add(time.Hour)),
		},
	}).SignedString(e.svc.secret)

	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role:             models.RoleSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "7"},
	}).SignedString(e.svc.secret)

	noRole, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{Issuer: issuer, Subject: "7", ExpiresAt: jwt.NewNumericDate(e.clock.Add(time.Hour))},
	}).SignedString(e.svc.secret)

	// alg "none": header + payload tanpa tanda tangan
	enc := base64.RawURLEncoding.EncodeToString
	none := enc([]byte(`{"alg":"none","typ":"JWT"}`)) + "." +
		enc([]byte(`{"iss":"sim-g","sub":"7","role":"Super Admin","exp":4102444800}`)) + "."

	tampered := valid[:len(valid)-2] + "xx"
	if strings.HasSuffix(valid, "xx") {
		tampered = valid[:len(valid)-2] + "yy"
	}

	cases := map[string]string{
		"kosong":              "",
		"sampah":              "bukan-jwt",
		"tanda tangan diubah": tampered,
		"secret berbeda":      otherToken,
		"issuer berbeda":      wrongIssuer,
		"tanpa exp":           noExp,
		"tanpa role":          noRole,
		"alg none":            none,
	}
	for name, tok := range cases {
		if _, err := e.svc.ParseAccessToken(tok); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
	if _, err := e.svc.ParseAccessToken(valid); err != nil {
		t.Errorf("token valid ditolak: %v", err)
	}
}
