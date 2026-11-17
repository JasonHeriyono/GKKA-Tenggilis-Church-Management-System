// Package identitas: form data diri jemaat (tabel identitas_user).
// Dipakai dua tempat: admin (modul user) dan jemaat sendiri (modul pendaftaran).
package identitas

import (
	"crypto/rand"
	"regexp"
	"strings"
	"time"

	"sim-g-backend/models"
)

// Isian: data diri dalam bentuk JSON. Tanggal format "YYYY-MM-DD", teks kosong = tidak diisi.
type Isian struct {
	NIK                string `json:"nik" binding:"omitempty,numeric,len=16"` // hanya Super Admin
	NamaPanggilan      string `json:"nama_panggilan" binding:"max=50"`
	JenisKelamin       string `json:"jenis_kelamin" binding:"omitempty,oneof=L P"`
	TempatLahir        string `json:"tempat_lahir" binding:"max=100"`
	TanggalLahir       string `json:"tanggal_lahir" binding:"omitempty,datetime=2006-01-02"`
	AlamatKTP          string `json:"alamat_ktp" binding:"max=500"`
	RtRw               string `json:"rt_rw" binding:"max=10"`
	Kelurahan          string `json:"kelurahan" binding:"max=100"`
	Kecamatan          string `json:"kecamatan" binding:"max=100"`
	Kota               string `json:"kota" binding:"max=100"`
	Provinsi           string `json:"provinsi" binding:"max=100"`
	AlamatDomisili     string `json:"alamat_domisili" binding:"max=500"`
	Kewarganegaraan    string `json:"kewarganegaraan" binding:"omitempty,oneof=WNI WNA"`
	StatusPernikahan   string `json:"status_pernikahan" binding:"omitempty,oneof=BELUM_KAWIN KAWIN CERAI_HIDUP CERAI_MATI"`
	TanggalPernikahan  string `json:"tanggal_pernikahan" binding:"omitempty,datetime=2006-01-02"`
	PendidikanTerakhir string `json:"pendidikan_terakhir" binding:"omitempty,oneof=SD SMP SMA D3 S1 S2 S3"`
	Pekerjaan          string `json:"pekerjaan" binding:"max=100"`
	NamaAyah           string `json:"nama_ayah" binding:"max=100"`
	NamaIbu            string `json:"nama_ibu" binding:"max=100"`
	StatusGereja       string `json:"status_gereja" binding:"omitempty,oneof=JEMAAT_GKKA SIMPATISAN TAMU"`
	SudahBaptis        bool   `json:"sudah_baptis"`
	TanggalBaptis      string `json:"tanggal_baptis" binding:"omitempty,datetime=2006-01-02"`
	GerejaBaptis       string `json:"gereja_baptis" binding:"max=150"`
	SudahSidi          bool   `json:"sudah_sidi"`
	TanggalSidi        string `json:"tanggal_sidi" binding:"omitempty,datetime=2006-01-02"`
	GerejaSidi         string `json:"gereja_sidi" binding:"max=150"`
	GerejaAsal         string `json:"gereja_asal" binding:"max=150"` // jika pindahan (atestasi)
	// diisi admin saja
	TanggalBergabung string `json:"tanggal_bergabung" binding:"omitempty,datetime=2006-01-02"`
	TanggalKeluar    string `json:"tanggal_keluar" binding:"omitempty,datetime=2006-01-02"`
	AlasanKeluar     string `json:"alasan_keluar" binding:"omitempty,oneof=PINDAH ATESTASI_KELUAR MENINGGAL"`
}

// Rapikan membuang spasi di awal/akhir semua isian teks.
func (isian *Isian) Rapikan() {
	daftarTeks := []*string{
		&isian.NIK, &isian.NamaPanggilan, &isian.JenisKelamin, &isian.TempatLahir, &isian.TanggalLahir,
		&isian.AlamatKTP, &isian.RtRw, &isian.Kelurahan, &isian.Kecamatan, &isian.Kota, &isian.Provinsi,
		&isian.AlamatDomisili, &isian.Kewarganegaraan, &isian.StatusPernikahan, &isian.TanggalPernikahan,
		&isian.PendidikanTerakhir, &isian.Pekerjaan, &isian.NamaAyah, &isian.NamaIbu, &isian.StatusGereja,
		&isian.TanggalBaptis, &isian.GerejaBaptis, &isian.TanggalSidi, &isian.GerejaSidi, &isian.GerejaAsal,
		&isian.TanggalBergabung, &isian.TanggalKeluar, &isian.AlasanKeluar,
	}
	for _, teks := range daftarTeks {
		*teks = strings.TrimSpace(*teks)
	}
}

// Cek: tanggal tidak boleh di masa depan atau sebelum 1900. Hasil = pesan error per field.
func (isian Isian) Cek() map[string]string {
	errorPerField := map[string]string{}
	daftarTanggal := map[string]string{
		"tanggal_lahir":      isian.TanggalLahir,
		"tanggal_pernikahan": isian.TanggalPernikahan,
		"tanggal_baptis":     isian.TanggalBaptis,
		"tanggal_sidi":       isian.TanggalSidi,
		"tanggal_bergabung":  isian.TanggalBergabung,
		"tanggal_keluar":     isian.TanggalKeluar,
	}
	for namaField, teks := range daftarTanggal {
		if teks == "" {
			continue
		}
		tanggal, err := time.Parse("2006-01-02", teks)
		if err != nil {
			errorPerField[namaField] = "format harus YYYY-MM-DD"
		} else if tanggal.After(time.Now()) || tanggal.Year() < 1900 {
			errorPerField[namaField] = "tanggal tidak masuk akal"
		}
	}
	return errorPerField
}

// PakaiKolomAdminDari: jemaat tidak boleh mengubah NIK dan data keanggotaan,
// jadi nilainya diambil dari data lama.
func (isian *Isian) PakaiKolomAdminDari(lama Isian) {
	isian.NIK = lama.NIK
	isian.TanggalBergabung = lama.TanggalBergabung
	isian.TanggalKeluar = lama.TanggalKeluar
	isian.AlasanKeluar = lama.AlasanKeluar
}

// KolomDataInti: data yang sudah diperiksa sekretariat. Setelah TERVERIFIKASI jemaat tidak bisa
// mengubahnya sendiri (minta sekretariat lewat web admin). "nama" ada di tabel user, sisanya di identitas.
var KolomDataInti = []string{
	"nama", "jenis_kelamin", "tempat_lahir", "tanggal_lahir",
	"sudah_baptis", "tanggal_baptis", "gereja_baptis", "sudah_sidi", "tanggal_sidi", "gereja_sidi",
}

// PakaiDataIntiDari: untuk jemaat TERVERIFIKASI, data inti diambil dari data lama (perubahan diabaikan).
func (isian *Isian) PakaiDataIntiDari(lama Isian) {
	isian.JenisKelamin = lama.JenisKelamin
	isian.TempatLahir = lama.TempatLahir
	isian.TanggalLahir = lama.TanggalLahir
	isian.SudahBaptis = lama.SudahBaptis
	isian.TanggalBaptis = lama.TanggalBaptis
	isian.GerejaBaptis = lama.GerejaBaptis
	isian.SudahSidi = lama.SudahSidi
	isian.TanggalSidi = lama.TanggalSidi
	isian.GerejaSidi = lama.GerejaSidi
}

// kolomWajib: satu data yang harus diisi sebelum jemaat bisa mengajukan verifikasi.
type kolomWajib struct {
	namaField string
	nilai     string
}

// kolomWajibVerifikasi: dipakai app untuk banner "Lengkapi data diri 2/5" (slice supaya urutannya tetap).
func (isian Isian) kolomWajibVerifikasi() []kolomWajib {
	return []kolomWajib{
		{"jenis_kelamin", isian.JenisKelamin},
		{"tempat_lahir", isian.TempatLahir},
		{"tanggal_lahir", isian.TanggalLahir},
		{"alamat_ktp", isian.AlamatKTP},
		{"status_gereja", isian.StatusGereja},
	}
}

// Kelengkapan: berapa data wajib yang sudah terisi, dan field apa saja yang masih kosong.
type Kelengkapan struct {
	Terisi int      `json:"terisi"`
	Total  int      `json:"total"`
	Kurang []string `json:"kurang"`
}

func (isian Isian) HitungKelengkapan() Kelengkapan {
	hasil := Kelengkapan{Kurang: []string{}}
	for _, kolom := range isian.kolomWajibVerifikasi() {
		hasil.Total++
		if kolom.nilai == "" {
			hasil.Kurang = append(hasil.Kurang, kolom.namaField)
		} else {
			hasil.Terisi++
		}
	}
	return hasil
}

// KeModel mengubah isian jadi baris identitas_user milik userID.
func (isian Isian) KeModel(userID uint) models.IdentitasUser {
	return models.IdentitasUser{
		UserID:             userID,
		NIK:                nilKalauKosong(isian.NIK),
		NamaPanggilan:      nilKalauKosong(isian.NamaPanggilan),
		JenisKelamin:       nilKalauKosong(isian.JenisKelamin),
		TempatLahir:        nilKalauKosong(isian.TempatLahir),
		TanggalLahir:       bacaTanggal(isian.TanggalLahir),
		AlamatKTP:          nilKalauKosong(isian.AlamatKTP),
		RtRw:               nilKalauKosong(isian.RtRw),
		Kelurahan:          nilKalauKosong(isian.Kelurahan),
		Kecamatan:          nilKalauKosong(isian.Kecamatan),
		Kota:               nilKalauKosong(isian.Kota),
		Provinsi:           nilKalauKosong(isian.Provinsi),
		AlamatDomisili:     nilKalauKosong(isian.AlamatDomisili),
		Kewarganegaraan:    nilKalauKosong(isian.Kewarganegaraan),
		StatusPernikahan:   nilKalauKosong(isian.StatusPernikahan),
		TanggalPernikahan:  bacaTanggal(isian.TanggalPernikahan),
		PendidikanTerakhir: nilKalauKosong(isian.PendidikanTerakhir),
		Pekerjaan:          nilKalauKosong(isian.Pekerjaan),
		NamaAyah:           nilKalauKosong(isian.NamaAyah),
		NamaIbu:            nilKalauKosong(isian.NamaIbu),
		StatusGereja:       nilKalauKosong(isian.StatusGereja),
		SudahBaptis:        isian.SudahBaptis,
		TanggalBaptis:      bacaTanggal(isian.TanggalBaptis),
		GerejaBaptis:       nilKalauKosong(isian.GerejaBaptis),
		SudahSidi:          isian.SudahSidi,
		TanggalSidi:        bacaTanggal(isian.TanggalSidi),
		GerejaSidi:         nilKalauKosong(isian.GerejaSidi),
		GerejaAsal:         nilKalauKosong(isian.GerejaAsal),
		TanggalBergabung:   bacaTanggal(isian.TanggalBergabung),
		TanggalKeluar:      bacaTanggal(isian.TanggalKeluar),
		AlasanKeluar:       nilKalauKosong(isian.AlasanKeluar),
	}
}

// DariModel kebalikan KeModel. data nil (belum pernah diisi) = isian kosong.
func DariModel(data *models.IdentitasUser) Isian {
	if data == nil {
		return Isian{}
	}
	return Isian{
		NIK:                teks(data.NIK),
		NamaPanggilan:      teks(data.NamaPanggilan),
		JenisKelamin:       teks(data.JenisKelamin),
		TempatLahir:        teks(data.TempatLahir),
		TanggalLahir:       tulisTanggal(data.TanggalLahir),
		AlamatKTP:          teks(data.AlamatKTP),
		RtRw:               teks(data.RtRw),
		Kelurahan:          teks(data.Kelurahan),
		Kecamatan:          teks(data.Kecamatan),
		Kota:               teks(data.Kota),
		Provinsi:           teks(data.Provinsi),
		AlamatDomisili:     teks(data.AlamatDomisili),
		Kewarganegaraan:    teks(data.Kewarganegaraan),
		StatusPernikahan:   teks(data.StatusPernikahan),
		TanggalPernikahan:  tulisTanggal(data.TanggalPernikahan),
		PendidikanTerakhir: teks(data.PendidikanTerakhir),
		Pekerjaan:          teks(data.Pekerjaan),
		NamaAyah:           teks(data.NamaAyah),
		NamaIbu:            teks(data.NamaIbu),
		StatusGereja:       teks(data.StatusGereja),
		SudahBaptis:        data.SudahBaptis,
		TanggalBaptis:      tulisTanggal(data.TanggalBaptis),
		GerejaBaptis:       teks(data.GerejaBaptis),
		SudahSidi:          data.SudahSidi,
		TanggalSidi:        tulisTanggal(data.TanggalSidi),
		GerejaSidi:         teks(data.GerejaSidi),
		GerejaAsal:         teks(data.GerejaAsal),
		TanggalBergabung:   tulisTanggal(data.TanggalBergabung),
		TanggalKeluar:      tulisTanggal(data.TanggalKeluar),
		AlasanKeluar:       teks(data.AlasanKeluar),
	}
}

// hurufKodeJemaat: tanpa 0/O dan 1/I supaya tidak tertukar saat dibaca.
const hurufKodeJemaat = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// BuatKodeJemaat membuat kode acak, contoh "JK7M2QXD" (huruf J + 8 karakter).
func BuatKodeJemaat() (string, error) {
	acak := make([]byte, 8)
	if _, err := rand.Read(acak); err != nil {
		return "", err
	}
	kode := "J"
	for _, angka := range acak {
		kode += string(hurufKodeJemaat[int(angka)%len(hurufKodeJemaat)])
	}
	return kode, nil
}

// polaNoHP: 8-15 angka, boleh diawali + untuk nomor luar negeri (contoh +6591234567).
var polaNoHP = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

// NoHPValid dipakai setelah RapikanNoHP.
func NoHPValid(noHP string) bool { return polaNoHP.MatchString(noHP) }

// RapikanNoHP membuang spasi, strip, kurung, dan titik, lalu mengubah awalan +62/62 menjadi 0
// supaya satu nomor tidak terdaftar dua kali dengan penulisan berbeda. Contoh "+62 812-345" -> "0812345".
// Nomor luar negeri (selain +62) tetap memakai + di depan, contoh "+65 9123 4567" -> "+6591234567".
func RapikanNoHP(noHP string) string {
	noHP = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(strings.TrimSpace(noHP))
	if strings.HasPrefix(noHP, "+62") {
		return "0" + noHP[3:]
	}
	if strings.HasPrefix(noHP, "62") && len(noHP) >= 10 {
		return "0" + noHP[2:]
	}
	return noHP
}

// bacaTanggal: "2000-05-17" -> jam 12:00 UTC supaya tanggal tidak bergeser oleh zona waktu driver.
func bacaTanggal(isi string) *time.Time {
	if isi == "" {
		return nil
	}
	tanggal, err := time.Parse("2006-01-02", isi)
	if err != nil {
		return nil
	}
	tanggal = tanggal.Add(12 * time.Hour)
	return &tanggal
}

func tulisTanggal(tanggal *time.Time) string {
	if tanggal == nil {
		return ""
	}
	return tanggal.Format("2006-01-02")
}

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
