package pengaturan

import "time"

// PengaturanResponse: satu pengaturan untuk web admin.
type PengaturanResponse struct {
	Kunci      string     `json:"kunci"`
	Nilai      string     `json:"nilai"`
	Bawaan     string     `json:"bawaan"`
	Deskripsi  string     `json:"deskripsi"`
	DiubahOleh *uint      `json:"diubah_oleh"` // null = belum pernah diubah admin
	UpdatedAt  *time.Time `json:"updated_at"`
}

// UpdatePengaturanRequest, contoh: {"pengaturan": {"evaluasi.lama_buka_hari": "3"}}
type UpdatePengaturanRequest struct {
	Pengaturan map[string]string `json:"pengaturan" binding:"required"`
}
