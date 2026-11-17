package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/helpers/currentuser"
	"sim-g-backend/helpers/identitas"
	"sim-g-backend/models"
)

const issuer = "sim-g"

// polaKetikanNoHP: identifier login yang dianggap no HP, contoh "+62 812-3456-7890" atau "(021) 555.1234".
var polaKetikanNoHP = regexp.MustCompile(`^\+?[0-9][0-9 ()\-.]*$`)

// ErrInvalidToken: access token rusak, kedaluwarsa, atau ditandatangani pihak lain.
var ErrInvalidToken = errors.New("token tidak valid")

// Service berisi logika login, refresh token, dan verifikasi access token. Tidak tahu soal HTTP.
type Service struct {
	repo       Repository
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
	dummyHash  []byte // dibandingkan saat user tidak ada, supaya waktu respons tidak membocorkan keberadaan akun
}

func NewService(repo Repository, jwtSecret string, accessTTL, refreshTTL time.Duration) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	return &Service{
		repo:       repo,
		secret:     []byte(jwtSecret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
		dummyHash:  dummy,
	}
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func invalidCredentials() *apperror.AppError {
	return apperror.Unauthorized("Identifier atau password salah")
}

// Login memverifikasi kredensial lalu menerbitkan access + refresh token.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, apperror.Validation(map[string]string{"identifier": "wajib diisi"})
	}
	// no HP disimpan dalam bentuk rapi ("+62 812-345" -> "0812345"), jadi yang diketik saat login ikut dirapikan.
	// Hanya kalau isinya memang no HP (angka, +, spasi, strip, kurung, titik), supaya email seperti
	// "08budi@gmail.com" tidak ikut dibuang titiknya.
	if polaKetikanNoHP.MatchString(identifier) {
		identifier = identitas.RapikanNoHP(identifier)
	}
	u, err := s.repo.FindUserByIdentifier(ctx, identifier)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal(err)
	}
	found := err == nil

	hash := s.dummyHash
	if found && u.PasswordHash != nil {
		hash = []byte(*u.PasswordHash)
	}
	pwErr := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if !found || u.PasswordHash == nil || pwErr != nil {
		return nil, invalidCredentials() // pesan sama untuk semua kasus agar akun tidak bisa ditebak
	}
	if !u.IsAktif {
		return nil, apperror.Forbidden("Akun tidak aktif, hubungi pengurus gereja")
	}

	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	info, err := s.userInfo(ctx, u)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &LoginResponse{TokenResponse: *tokens, User: *info}, nil
}

// MasukOtomatis menerbitkan token untuk user yang baru dibuat lewat signup,
// supaya jemaat langsung masuk tanpa login ulang.
func (s *Service) MasukOtomatis(ctx context.Context, userID uint) (*LoginResponse, error) {
	u, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	info, err := s.userInfo(ctx, u)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{TokenResponse: *tokens, User: *info}, nil
}

// Refresh menukar refresh token dengan pasangan token baru (rotasi: token lama langsung dicabut).
// Token yang sudah dicabut tapi dipakai lagi dianggap bocor, sehingga semua sesi user itu dicabut.
func (s *Service) Refresh(ctx context.Context, rawToken string) (*TokenResponse, error) {
	rt, err := s.repo.FindRefreshToken(ctx, hashToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.Unauthorized("Refresh token tidak valid")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if rt.RevokedAt != nil {
		_ = s.repo.RevokeAllUserTokens(ctx, rt.UserID)
		return nil, apperror.Unauthorized("Sesi tidak berlaku lagi, silakan login ulang")
	}
	if !rt.ExpiredAt.After(s.now()) {
		return nil, apperror.Unauthorized("Sesi sudah berakhir, silakan login ulang")
	}

	u, err := s.repo.FindUserByID(ctx, rt.UserID)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.Unauthorized("Akun tidak ditemukan")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if !u.IsAktif {
		_ = s.repo.RevokeAllUserTokens(ctx, u.ID)
		return nil, apperror.Forbidden("Akun tidak aktif, hubungi pengurus gereja")
	}

	access, err := s.signAccess(u)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	raw, next, err := s.newRefreshToken(u.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	rotated, err := s.repo.RotateRefreshToken(ctx, rt.ID, next)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if !rotated {
		_ = s.repo.RevokeAllUserTokens(ctx, u.ID)
		return nil, apperror.Unauthorized("Sesi tidak berlaku lagi, silakan login ulang")
	}
	return s.tokenResponse(access, raw), nil
}

// Logout mencabut refresh token. Selalu sukses (idempoten) agar token tidak bisa ditebak.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if err := s.repo.RevokeRefreshToken(ctx, hashToken(rawToken)); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

// Me mengembalikan profil singkat user yang sedang login.
func (s *Service) Me(ctx context.Context, userID uint) (*UserInfo, error) {
	u, err := s.repo.FindUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.Unauthorized("Akun tidak ditemukan")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}
	info, err := s.userInfo(ctx, u)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return info, nil
}

// ParseAccessToken dipakai middleware.Auth. Stateless: hanya memeriksa tanda tangan, issuer, dan masa berlaku.
// Perubahan role atau penonaktifan akun berlaku paling lambat saat access token berikutnya diperbarui.
func (s *Service) ParseAccessToken(token string) (currentuser.User, error) {
	var cl claims
	parsed, err := jwt.ParseWithClaims(token, &cl,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil || !parsed.Valid {
		return currentuser.User{}, ErrInvalidToken
	}
	id, err := strconv.ParseUint(cl.Subject, 10, 32)
	if err != nil || id == 0 || cl.Role == "" {
		return currentuser.User{}, ErrInvalidToken
	}
	return currentuser.User{ID: uint(id), Role: cl.Role}, nil
}

func (s *Service) issueTokens(ctx context.Context, u *models.User) (*TokenResponse, error) {
	access, err := s.signAccess(u)
	if err != nil {
		return nil, err
	}
	raw, rt, err := s.newRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	return s.tokenResponse(access, raw), nil
}

func (s *Service) signAccess(u *models.User) (string, error) {
	now := s.now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: u.Role.Nama,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   strconv.FormatUint(uint64(u.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	})
	return t.SignedString(s.secret)
}

// newRefreshToken membuat token acak 256-bit. Ke client dikirim token mentahnya; ke DB hanya hash-nya.
func (s *Service) newRefreshToken(userID uint) (raw string, rt *models.RefreshToken, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, &models.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(raw),
		ExpiredAt: s.now().Add(s.refreshTTL),
	}, nil
}

func (s *Service) tokenResponse(access, refresh string) *TokenResponse {
	return &TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// userInfo = toUserInfo + daftar keahlian user.
func (s *Service) userInfo(ctx context.Context, u *models.User) (*UserInfo, error) {
	daftar, err := s.repo.ListKeahlianUser(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	info := toUserInfo(u)
	info.Keahlian = make([]KeahlianInfo, 0, len(daftar))
	for _, k := range daftar {
		info.Keahlian = append(info.Keahlian, KeahlianInfo{ID: k.ID, Nama: k.Nama})
	}
	return &info, nil
}

func toUserInfo(u *models.User) UserInfo {
	return UserInfo{ID: u.ID, KodeJemaat: u.KodeJemaat, Nama: u.Nama, Email: u.Email, Role: u.Role.Nama,
		StatusVerifikasi: u.StatusVerifikasi}
}
