package pengaturan

import (
	"strconv"
	"time"
)

// Kunci pengaturan yang dibaca modul lain.
const (
	KunciLamaBukaHari   = "evaluasi.lama_buka_hari"
	KunciJamTutup       = "evaluasi.jam_tutup"
	KunciDurasiKegiatan = "kegiatan.durasi_bawaan_menit"
)

// Aturan: satu pengaturan yang boleh diubah admin.
// Pengaturan baru cukup ditambah satu baris di daftarAturan, tanpa migrasi.
type Aturan struct {
	Kunci     string
	Bawaan    string // dipakai kalau admin belum pernah menyimpan
	Deskripsi string
	Cek       func(nilai string) string // pesan error, "" = nilai benar
}

var daftarAturan = []Aturan{
	// minimal H+1 supaya jam tutup selalu setelah kebaktian selesai
	{KunciLamaBukaHari, "2", "Form evaluasi mingguan ditutup H+N setelah kebaktian", angkaAntara(1, 7)},
	// jam form ditutup pada hari H+N, contoh 23:59
	{KunciJamTutup, "23:59", "Jam form evaluasi ditutup (format JJ:MM)", formatJam},
	// kalau jam selesai kegiatan kosong, form dibuka jam mulai + durasi ini (maks 12 jam)
	{KunciDurasiKegiatan, "120", "Lama kegiatan (menit) kalau jam selesai tidak diisi", angkaAntara(15, 720)},
}

// cariAturan: nil kalau kunci tidak dikenal.
func cariAturan(kunci string) *Aturan {
	for i := range daftarAturan {
		if daftarAturan[i].Kunci == kunci {
			return &daftarAturan[i]
		}
	}
	return nil
}

// Nilai: nilai dari database, atau nilai bawaan kalau belum disimpan.
// isi = hasil baca tabel pengaturan (kunci -> nilai).
func Nilai(isi map[string]string, kunci string) string {
	if nilai, ada := isi[kunci]; ada {
		return nilai
	}
	if aturan := cariAturan(kunci); aturan != nil {
		return aturan.Bawaan
	}
	return ""
}

// Angka: seperti Nilai tapi dijadikan angka. Isi database yang rusak diganti nilai bawaan.
func Angka(isi map[string]string, kunci string) int {
	angka, err := strconv.Atoi(Nilai(isi, kunci))
	if err != nil {
		angka, _ = strconv.Atoi(cariAturan(kunci).Bawaan)
	}
	return angka
}

// angkaAntara: nilai harus angka bulat dari min sampai max.
func angkaAntara(min, max int) func(string) string {
	return func(nilai string) string {
		angka, err := strconv.Atoi(nilai)
		if err != nil || angka < min || angka > max {
			return "harus angka " + strconv.Itoa(min) + " sampai " + strconv.Itoa(max)
		}
		return ""
	}
}

// formatJam: contoh "23:59".
func formatJam(nilai string) string {
	if _, err := time.Parse("15:04", nilai); err != nil {
		return "format harus JJ:MM, contoh 23:59"
	}
	return ""
}
