package evaluation

// PertanyaanRequest dipakai untuk tambah dan ubah pertanyaan. Urutan opsi = urutan di array.
type PertanyaanRequest struct {
	JenisKuesioner string   `json:"jenis_kuesioner" binding:"required,oneof=MINGGUAN TAHUNAN"`
	Teks           string   `json:"teks_pertanyaan" binding:"required"`
	TipeJawaban    string   `json:"tipe_jawaban" binding:"required,oneof=RATING TEKS_SINGKAT PILIHAN_GANDA CENTANG DROPDOWN PARAGRAF"`
	Urutan         int      `json:"urutan" binding:"min=1"` //Urutan minimal dari 1
	Wajib          bool     `json:"wajib"`
	Opsi           []string `json:"opsi" binding:"dive,max=255"` //dive ini untuk validasi item di dalam array
}

// AktifRequest: untuk data pertanyaan (Yang aktif akan muncul di form evaluasi)
type AktifRequest struct {
	IsAktif *bool `json:"is_aktif" binding:"required"`
}

// PertanyaanQuery: ?jenis=MINGGUAN&aktif=true
type PertanyaanQuery struct {
	Jenis string
	Aktif *bool
}

type OpsiResponse struct {
	ID     uint   `json:"id"`
	Label  string `json:"label"`
	Urutan int    `json:"urutan"`
}

// SinkronResponse: ringkasan perubahan di Google Form setelah sinkron.
type SinkronResponse struct {
	JenisKuesioner string `json:"jenis_kuesioner"`
	Ditambah       int    `json:"ditambah"`
	Diubah         int    `json:"diubah"`
	Dipindah       int    `json:"dipindah"`
	Dihapus        int    `json:"dihapus"`
	LinkForm       string `json:"link_form"` // link untuk jemaat mengisi form
}

type PertanyaanResponse struct {
	ID             uint           `json:"id"`
	JenisKuesioner string         `json:"jenis_kuesioner"`
	Teks           string         `json:"teks_pertanyaan"`
	TipeJawaban    string         `json:"tipe_jawaban"`
	Urutan         int            `json:"urutan"`
	Wajib          bool           `json:"wajib"`
	IsAktif        bool           `json:"is_aktif"`
	Opsi           []OpsiResponse `json:"opsi"`
}

// Status baris import: BARU = pertanyaan baru dibuat, SAMA = pertanyaan lama dipakai ulang.
const (
	StatusImportBaru = "BARU"
	StatusImportSama = "SAMA"
)

// ImportBarisPertanyaan: satu baris dari sheet "Pertanyaan" di template Excel.
type ImportBarisPertanyaan struct {
	Baris          int               `json:"baris"` // nomor baris di Excel (baris header = 1)
	JenisKuesioner string            `json:"jenis_kuesioner"`
	Urutan         int               `json:"urutan"`
	Teks           string            `json:"teks_pertanyaan"`
	TipeJawaban    string            `json:"tipe_jawaban"`
	Opsi           []string          `json:"opsi"`
	Wajib          bool              `json:"wajib"`
	Status         string            `json:"status,omitempty"` // BARU / SAMA, diisi kalau baris tidak error
	Errors         map[string]string `json:"errors,omitempty"` // nama kolom -> pesan error
}

// ImportPreviewResponse: hasil POST /pertanyaan/import/preview. Belum ada data yang disimpan.
type ImportPreviewResponse struct {
	TotalBaris        int                     `json:"total_baris"`
	BarisValid        int                     `json:"baris_valid"`
	BarisError        int                     `json:"baris_error"`
	BisaDisimpan      bool                    `json:"bisa_disimpan"`      // true kalau tidak ada baris error
	AkanDinonaktifkan []PertanyaanResponse    `json:"akan_dinonaktifkan"` // pertanyaan lama yang tidak ada di Excel
	Baris             []ImportBarisPertanyaan `json:"baris"`
}

// ImportSimpanResponse: ringkasan hasil POST /pertanyaan/import.
type ImportSimpanResponse struct {
	Ditambah      int `json:"ditambah"`
	Diperbarui    int `json:"diperbarui"`
	Dinonaktifkan int `json:"dinonaktifkan"`
}

type UpdateHeaderRequest struct {
	Jenis     string `json:"jenis_kuesioner" binding:"required"`
	Judul     string `json:"judul" binding:"required"`
	Deskripsi string `json:"deskripsi"`
}
