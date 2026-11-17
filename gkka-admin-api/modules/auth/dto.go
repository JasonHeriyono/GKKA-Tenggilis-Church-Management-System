package auth

// LoginRequest: identifier boleh email, kode jemaat, atau no. telepon.
type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required,max=100"`
	Password   string `json:"password" binding:"required,max=72"`
}

// RefreshRequest dipakai untuk /auth/refresh dan /auth/logout.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik sampai access token kedaluwarsa
}

type UserInfo struct {
	ID         uint    `json:"id"`
	KodeJemaat *string `json:"kode_jemaat"` // null kalau belum terverifikasi
	Nama       string  `json:"nama"`
	Email      *string `json:"email"`
	Role       string  `json:"role"`
	// BELUM_TERVERIFIKASI / MENUNGGU_VERIFIKASI / TERVERIFIKASI, dipakai app untuk banner "Lengkapi data diri"
	StatusVerifikasi string `json:"status_verifikasi"`
	// keahlian yang sudah diverifikasi admin, dipakai app untuk "pelayanan yang dapat diikuti"
	Keahlian []KeahlianInfo `json:"keahlian"`
}

type KeahlianInfo struct {
	ID   uint   `json:"id"`
	Nama string `json:"nama"`
}

type LoginResponse struct {
	TokenResponse
	User UserInfo `json:"user"`
}
