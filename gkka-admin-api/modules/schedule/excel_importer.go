package schedule

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

const (
	// MaxImportRows membatasi jumlah baris data per file.
	MaxImportRows = 1000

	maxActivityLen = 150
	maxUnzipSize   = 50 << 20 // batas ukuran isi file setelah di-unzip (cegah zip bomb)
	maxXMLSize     = 20 << 20
)

var (
	ErrNoSheet     = errors.New("file Excel tidak memiliki sheet Kegiatan")
	ErrTooManyRows = fmt.Errorf("jumlah baris melebihi batas %d", MaxImportRows)
)

type column struct {
	key      string
	required bool
}

// templateColumns: kolom sheet "Kegiatan" di template_import/template_kegiatan.xlsx. Urutan kolom
// di Excel bebas karena dicocokkan lewat nama header (huruf besar/kecil dan spasi diabaikan).
// Kolom lain di template (program_training, periode_program) belum dipakai.
var templateColumns = []column{
	{"nama_kegiatan", true},
	{"jenis_kegiatan", false},
	{"tanggal", true},
	{"waktu_mulai", true},
	{"waktu_selesai", false},
	{"tema", false},
	{"nats_alkitab", false},
	{"pembicara", false},
	{"lokasi", false},
	{"deskripsi", false},
	{"komunitas", false},
	{"perlu_evaluasi", false},
	{"petugas", false}, // tidak ada di template baru; kalau diisi dipakai untuk cek bentrok petugas
	{"skill", false},
}

// namaSheetKegiatan: sheet yang dibaca. Sheet lain (Petunjuk, Contoh, Referensi) diabaikan.
const namaSheetKegiatan = "Kegiatan"

// Isi kolom jenis_kegiatan yang diterima.
const (
	JenisKebaktian = "KEBAKTIAN"
	JenisTraining  = "TRAINING"
	JenisEvent     = "EVENT"
)

// readSheet membaca sheet "Kegiatan" dari file .xlsx menjadi baris-baris teks.
// RawCellValue: sel tanggal/jam dibaca sebagai angka serial Excel, bukan teks berformat.
func readSheet(r io.Reader) ([][]string, error) {
	f, err := excelize.OpenReader(r, excelize.Options{
		UnzipSizeLimit:    maxUnzipSize,
		UnzipXMLSizeLimit: maxXMLSize,
	})
	if err != nil {
		return nil, fmt.Errorf("buka file Excel: %w", err)
	}
	defer f.Close()

	for _, namaSheet := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(namaSheet), namaSheetKegiatan) {
			rows, err := f.GetRows(namaSheet, excelize.Options{RawCellValue: true})
			if err != nil {
				return nil, fmt.Errorf("baca sheet %q: %w", namaSheet, err)
			}
			return rows, nil
		}
	}
	return nil, ErrNoSheet
}

// parsedSheet adalah hasil parseRows.
type parsedSheet struct {
	Rows    []ImportRow
	Missing []string // kolom wajib yang tidak ada di header
}

// parseRows memvalidasi isi sheet per baris. Baris pertama adalah header; baris kosong dilewati.
// Fungsi ini murni (tanpa I/O) sehingga mudah diuji.
func parseRows(rows [][]string, loc *time.Location) (*parsedSheet, error) {
	out := &parsedSheet{Rows: []ImportRow{}}
	if len(rows) == 0 {
		out.Missing = requiredKeys()
		return out, nil
	}

	idx := map[string]int{}
	for i, h := range rows[0] {
		key := normalizeHeader(h)
		if _, dup := idx[key]; !dup && key != "" {
			idx[key] = i
		}
	}
	for _, c := range templateColumns {
		if _, ok := idx[c.key]; !ok && c.required {
			out.Missing = append(out.Missing, c.key)
		}
	}
	if len(out.Missing) > 0 {
		return out, nil
	}
	col := func(key string) int {
		if v, ok := idx[key]; ok {
			return v
		}
		return -1
	}

	seen := map[string]int{} // kunci baris -> nomor baris pertama (deteksi duplikat dalam file)
	for i, raw := range rows[1:] {
		if blankRow(raw) {
			continue
		}
		if len(out.Rows) >= MaxImportRows {
			return nil, ErrTooManyRows
		}

		row := ImportRow{
			Line:          i + 2,
			Activity:      cell(raw, col("nama_kegiatan")),
			JenisKegiatan: strings.ToUpper(cell(raw, col("jenis_kegiatan"))),
			Tema:          cell(raw, col("tema")),
			NatsAlkitab:   cell(raw, col("nats_alkitab")),
			Pembicara:     cell(raw, col("pembicara")),
			Lokasi:        cell(raw, col("lokasi")),
			Deskripsi:     cell(raw, col("deskripsi")),
			Komunitas:     cell(raw, col("komunitas")),
			Assignee:      cell(raw, col("petugas")),
			Skill:         cell(raw, col("skill")),
		}
		errs := map[string]string{}

		if row.Activity == "" {
			errs["nama_kegiatan"] = "wajib diisi"
		} else if utf8.RuneCountInString(row.Activity) > maxActivityLen {
			errs["nama_kegiatan"] = fmt.Sprintf("maksimal %d karakter", maxActivityLen)
		}

		switch row.JenisKegiatan {
		case "", JenisKebaktian, JenisTraining, JenisEvent:
		default:
			errs["jenis_kegiatan"] = "harus KEBAKTIAN, TRAINING, atau EVENT"
		}

		// perlu_evaluasi kosong: otomatis YA untuk KEBAKTIAN, TIDAK untuk yang lain
		switch strings.ToUpper(cell(raw, col("perlu_evaluasi"))) {
		case "YA":
			row.PerluEvaluasi = true
		case "TIDAK":
			row.PerluEvaluasi = false
		case "":
			row.PerluEvaluasi = row.JenisKegiatan == JenisKebaktian
		default:
			errs["perlu_evaluasi"] = "harus YA atau TIDAK"
		}

		date, dateErr := parseDate(cell(raw, col("tanggal")), loc)
		if dateErr != nil {
			errs["tanggal"] = dateErr.Error()
		} else {
			row.Date = date.Format("2006-01-02")
		}

		sh, sm, startErr := parseClock(cell(raw, col("waktu_mulai")))
		if startErr != nil {
			errs["waktu_mulai"] = startErr.Error()
		} else {
			row.WaktuMulai = fmt.Sprintf("%02d:%02d", sh, sm)
		}
		if dateErr == nil && startErr == nil {
			y, mo, d := date.Date()
			start := time.Date(y, mo, d, sh, sm, 0, 0, loc)
			row.Start = &start
		}

		// waktu_selesai boleh kosong; kalau diisi harus setelah waktu_mulai
		if teksSelesai := cell(raw, col("waktu_selesai")); teksSelesai != "" {
			eh, em, endErr := parseClock(teksSelesai)
			if endErr != nil {
				errs["waktu_selesai"] = endErr.Error()
			} else {
				row.WaktuSelesai = fmt.Sprintf("%02d:%02d", eh, em)
				if row.Start != nil {
					y, mo, d := row.Start.Date()
					end := time.Date(y, mo, d, eh, em, 0, 0, loc)
					if !end.After(*row.Start) {
						errs["waktu_selesai"] = "harus setelah waktu_mulai"
					} else {
						row.End = &end
					}
				}
			}
		}

		if len(errs) == 0 {
			// sama dengan aturan unik di database: nama + tanggal + jam mulai
			key := strings.Join([]string{row.Date, row.WaktuMulai, strings.ToLower(row.Activity)}, "|")
			if first, dup := seen[key]; dup {
				errs["baris"] = fmt.Sprintf("duplikat dengan baris %d", first)
			} else {
				seen[key] = row.Line
			}
		}

		if len(errs) > 0 {
			row.Errors = errs
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func requiredKeys() []string {
	var keys []string
	for _, c := range templateColumns {
		if c.required {
			keys = append(keys, c.key)
		}
	}
	return keys
}

func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	return strings.NewReplacer(" ", "_", "-", "_").Replace(h)
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func blankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// Rentang angka serial Excel yang dianggap tanggal wajar: tahun 2000 sampai 2100.
const (
	serialMin = 36526
	serialMax = 73415
)

// parseDate menerima angka serial Excel, 2026-10-25, 25/10/2026 (hari/bulan/tahun), dan 25 Oct 2026.
func parseDate(s string, loc *time.Location) (time.Time, error) {
	const hint = "format tanggal tidak dikenali (contoh: 2026-10-25 atau 25/10/2026)"
	if s == "" {
		return time.Time{}, errors.New("wajib diisi")
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && f >= serialMin && f <= serialMax {
		base := time.Date(1899, 12, 30, 0, 0, 0, 0, loc)
		return base.AddDate(0, 0, int(math.Floor(f))), nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "2 Jan 2006", "2 January 2006"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New(hint)
}

// parseClock menerima pecahan hari Excel (0.375 = 09:00), 08:00, 08.00, 08:00:00, dan 8:00 AM.
func parseClock(s string) (hour, minute int, err error) {
	if s == "" {
		return 0, 0, errors.New("wajib diisi")
	}
	if f, perr := strconv.ParseFloat(s, 64); perr == nil && f >= 0 && f < 1 {
		total := int(math.Round(f * 24 * 60))
		return total / 60 % 24, total % 60, nil
	}
	for _, layout := range []string{"15:04", "15.04", "15:04:05", "3:04 PM", "3:04PM"} {
		if t, perr := time.Parse(layout, s); perr == nil {
			return t.Hour(), t.Minute(), nil
		}
	}
	return 0, 0, errors.New("format jam tidak dikenali (contoh: 08:00)")
}
