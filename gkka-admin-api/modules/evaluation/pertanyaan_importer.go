package evaluation

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"sim-g-backend/models"
)

const (
	// nama sheet di template yang dibaca; sheet Petunjuk dan Contoh diabaikan
	namaSheetPertanyaan = "Pertanyaan"

	// batas Google Form: maksimal 400 item per form (sudah diuji langsung)
	maksPertanyaanPerJenis = 400

	maksPanjangOpsi = 255      // sama dengan kolom label di tabel opsi_pertanyaan
	maksUkuranUnzip = 50 << 20 // batas isi file setelah di-unzip (cegah zip bomb)
	maksUkuranXML   = 20 << 20
)

var (
	ErrSheetTidakAda           = errors.New("sheet Pertanyaan tidak ditemukan")
	ErrPertanyaanTerlaluBanyak = fmt.Errorf("maksimal %d pertanyaan per jenis kuesioner", maksPertanyaanPerJenis)
)

// kolom yang wajib ada di baris header. opsi_jawaban boleh tidak ada (dianggap kosong semua).
var kolomWajibImport = []string{"jenis_kuesioner", "urutan", "teks_pertanyaan", "tipe_jawaban", "wajib"}

// bacaSheetPertanyaan buka file Excel lalu ambil semua baris dari sheet "Pertanyaan".
func bacaSheetPertanyaan(file io.Reader) ([][]string, error) {
	excel, err := excelize.OpenReader(file, excelize.Options{
		UnzipSizeLimit:    maksUkuranUnzip,
		UnzipXMLSizeLimit: maksUkuranXML,
	})
	if err != nil {
		return nil, fmt.Errorf("buka file Excel: %w", err)
	}
	defer excel.Close()

	// cari sheet bernama "Pertanyaan" (huruf besar/kecil tidak masalah)
	for _, namaSheet := range excel.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(namaSheet), namaSheetPertanyaan) {
			// RawCellValue: angka dibaca apa adanya, misal urutan 1 -> "1"
			return excel.GetRows(namaSheet, excelize.Options{RawCellValue: true})
		}
	}
	return nil, ErrSheetTidakAda
}

// parseBarisPertanyaan cek isi sheet baris per baris. Baris pertama = header, baris kosong dilewati.
// Hasil: daftar baris (yang salah berisi Errors) dan nama kolom wajib yang tidak ada di header.
// Fungsi ini tidak membaca database, jadi mudah dites.
func parseBarisPertanyaan(rows [][]string) ([]ImportBarisPertanyaan, []string, error) {
	hasil := []ImportBarisPertanyaan{}
	if len(rows) == 0 {
		return hasil, kolomWajibImport, nil
	}

	// catat posisi tiap kolom dari baris header, contoh: "teks_pertanyaan" -> kolom ke-2
	posisiKolom := map[string]int{}
	for i, judul := range rows[0] {
		nama := rapikanHeader(judul)
		if _, sudahAda := posisiKolom[nama]; !sudahAda && nama != "" {
			posisiKolom[nama] = i
		}
	}
	var kolomHilang []string
	for _, nama := range kolomWajibImport {
		if _, ada := posisiKolom[nama]; !ada {
			kolomHilang = append(kolomHilang, nama)
		}
	}
	if len(kolomHilang) > 0 {
		return hasil, kolomHilang, nil
	}

	// ambil isi sel berdasarkan nama kolom; kolom yang tidak ada -> teks kosong
	ambil := func(baris []string, namaKolom string) string {
		posisi, ada := posisiKolom[namaKolom]
		if !ada || posisi >= len(baris) {
			return ""
		}
		return strings.TrimSpace(baris[posisi])
	}

	jumlahPerJenis := map[string]int{}
	urutanDipakai := map[string]int{} // "MINGGUAN|1" -> nomor baris yang pertama memakai
	teksDipakai := map[string]int{}   // "MINGGUAN|teks huruf kecil" -> nomor baris

	for i, isiBaris := range rows[1:] {
		if barisKosong(isiBaris) {
			continue
		}

		baris := ImportBarisPertanyaan{Baris: i + 2, Opsi: []string{}} // +2: baris Excel mulai dari 1 dan ada header
		errorPerKolom := map[string]string{}

		// jenis_kuesioner
		baris.JenisKuesioner = strings.ToUpper(ambil(isiBaris, "jenis_kuesioner"))
		if baris.JenisKuesioner == "" {
			errorPerKolom["jenis_kuesioner"] = "wajib diisi"
		} else if !jenisValid(baris.JenisKuesioner) {
			errorPerKolom["jenis_kuesioner"] = "harus MINGGUAN atau TAHUNAN"
		} else {
			jumlahPerJenis[baris.JenisKuesioner]++
			if jumlahPerJenis[baris.JenisKuesioner] > maksPertanyaanPerJenis {
				return nil, nil, ErrPertanyaanTerlaluBanyak
			}
		}

		// urutan
		teksUrutan := ambil(isiBaris, "urutan")
		urutan, err := strconv.Atoi(teksUrutan)
		if teksUrutan == "" {
			errorPerKolom["urutan"] = "wajib diisi"
		} else if err != nil || urutan < 1 {
			errorPerKolom["urutan"] = "harus angka bulat minimal 1"
		} else {
			baris.Urutan = urutan
		}

		// teks_pertanyaan
		baris.Teks = ambil(isiBaris, "teks_pertanyaan")
		if baris.Teks == "" {
			errorPerKolom["teks_pertanyaan"] = "wajib diisi"
		}

		// tipe_jawaban
		baris.TipeJawaban = strings.ToUpper(ambil(isiBaris, "tipe_jawaban"))
		if baris.TipeJawaban == "" {
			errorPerKolom["tipe_jawaban"] = "wajib diisi"
		} else if !tipeValid(baris.TipeJawaban) {
			errorPerKolom["tipe_jawaban"] = "harus RATING, TEKS_SINGKAT, PARAGRAF, PILIHAN_GANDA, CENTANG, atau DROPDOWN"
		}

		// opsi_jawaban: "Iya;Tidak" -> ["Iya", "Tidak"]. Dicek kalau tipenya sudah benar.
		if _, adaError := errorPerKolom["tipe_jawaban"]; !adaError {
			daftarOpsi := pecahOpsi(ambil(isiBaris, "opsi_jawaban"))
			labelOpsi, pesan := cekOpsi(baris.TipeJawaban, daftarOpsi)
			if pesan == "" {
				pesan = cekPanjangOpsi(labelOpsi)
			}
			if pesan != "" {
				errorPerKolom["opsi_jawaban"] = pesan
			} else if labelOpsi != nil {
				baris.Opsi = labelOpsi
			}
		}

		// wajib: YA / TIDAK
		switch strings.ToUpper(ambil(isiBaris, "wajib")) {
		case "YA":
			baris.Wajib = true
		case "TIDAK":
			baris.Wajib = false
		case "":
			errorPerKolom["wajib"] = "wajib diisi"
		default:
			errorPerKolom["wajib"] = "harus YA atau TIDAK"
		}

		// urutan dan teks tidak boleh dobel dalam jenis yang sama
		if baris.JenisKuesioner != "" && baris.Urutan > 0 {
			kunci := baris.JenisKuesioner + "|" + strconv.Itoa(baris.Urutan)
			if barisPertama, dobel := urutanDipakai[kunci]; dobel {
				errorPerKolom["urutan"] = fmt.Sprintf("dobel dengan baris %d", barisPertama)
			} else {
				urutanDipakai[kunci] = baris.Baris
			}
		}
		if baris.JenisKuesioner != "" && baris.Teks != "" {
			kunci := baris.JenisKuesioner + "|" + strings.ToLower(baris.Teks)
			if barisPertama, dobel := teksDipakai[kunci]; dobel {
				errorPerKolom["teks_pertanyaan"] = fmt.Sprintf("dobel dengan baris %d", barisPertama)
			} else {
				teksDipakai[kunci] = baris.Baris
			}
		}

		if len(errorPerKolom) > 0 {
			baris.Errors = errorPerKolom
		}
		hasil = append(hasil, baris)
	}
	return hasil, nil, nil
}

// rapikanHeader: " Teks Pertanyaan " -> "teks_pertanyaan"
func rapikanHeader(judul string) string {
	judul = strings.ToLower(strings.TrimSpace(judul))
	return strings.NewReplacer(" ", "_", "-", "_").Replace(judul)
}

// barisKosong: true kalau semua sel di baris ini kosong.
func barisKosong(baris []string) bool {
	for _, isi := range baris {
		if strings.TrimSpace(isi) != "" {
			return false
		}
	}
	return true
}

// pecahOpsi: "Iya; Tidak;" -> ["Iya", "Tidak"]. Potongan kosong (misal titik koma di akhir) dibuang.
func pecahOpsi(teks string) []string {
	daftarOpsi := []string{}
	for _, opsi := range strings.Split(teks, ";") {
		opsi = strings.TrimSpace(opsi)
		if opsi != "" {
			daftarOpsi = append(daftarOpsi, opsi)
		}
	}
	return daftarOpsi
}

// cekPanjangOpsi: label opsi maksimal 255 karakter. Hasil "" artinya lolos.
func cekPanjangOpsi(daftarLabel []string) string {
	for _, label := range daftarLabel {
		if utf8.RuneCountInString(label) > maksPanjangOpsi {
			return fmt.Sprintf("opsi \"%s...\" lebih dari %d karakter", string([]rune(label)[:20]), maksPanjangOpsi)
		}
	}
	return ""
}

// tipeValid: tipe jawaban harus salah satu dari 6 tipe yang dikenal.
func tipeValid(tipeJawaban string) bool {
	switch tipeJawaban {
	case models.TipeRating, models.TipeTeksSingkat, models.TipeParagraf,
		models.TipePilihanGanda, models.TipeCentang, models.TipeDropdown:
		return true
	}
	return false
}
