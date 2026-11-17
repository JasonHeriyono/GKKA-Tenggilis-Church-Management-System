package schedule

import (
	"testing"
	"time"
)

var wib = time.FixedZone("WIB", 7*60*60)

var header = []string{
	"nama_kegiatan",
	"jenis_kegiatan",
	"tanggal",
	"waktu_mulai",
	"waktu_selesai",
	"tema",
	"nats_alkitab",
	"pembicara",
	"lokasi",
	"deskripsi",
	"komunitas",
	"perlu_evaluasi",
	"petugas",
	"skill",
}

func importRow(activity, jenis, date, start, end, evaluasi, petugas, skill string) []string {
	return []string{
		activity, jenis, date, start, end,
		"", "", "", "", "", "",
		evaluasi, petugas, skill,
	}
}

func sheet(data ...[]string) [][]string {
	return append([][]string{header}, data...)
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"2026-10-25", "2026-10-25", false},
		{"25/10/2026", "2026-10-25", false},
		{"5/1/2026", "2026-01-05", false},
		{"25 Oct 2026", "2026-10-25", false},
		{"46320", "2026-10-25", false},
		{"", "", true},
		{"besok", "", true},
		{"31/02/2026", "", true},
		{"2026", "", true},
	}

	for _, tc := range tests {
		got, err := parseDate(tc.in, wib)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseDate(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			continue
		}
		if err == nil && got.Format("2006-01-02") != tc.want {
			t.Errorf("parseDate(%q) = %s, want %s", tc.in, got.Format("2006-01-02"), tc.want)
		}
	}
}

func TestParseClock(t *testing.T) {
	tests := []struct {
		in      string
		h, m    int
		wantErr bool
	}{
		{"08:00", 8, 0, false},
		{"8:30", 8, 30, false},
		{"08.30", 8, 30, false},
		{"08:00:00", 8, 0, false},
		{"9:15 AM", 9, 15, false},
		{"0.375", 9, 0, false},
		{"0.5", 12, 0, false},
		{"0.999999", 0, 0, false},
		{"", 0, 0, true},
		{"25:00", 0, 0, true},
		{"pagi", 0, 0, true},
	}

	for _, tc := range tests {
		h, m, err := parseClock(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseClock(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			continue
		}
		if err == nil && (h != tc.h || m != tc.m) {
			t.Errorf("parseClock(%q) = %02d:%02d, want %02d:%02d", tc.in, h, m, tc.h, tc.m)
		}
	}
}

func TestParseRows_Valid(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("Ibadah Raya 1", "KEBAKTIAN", "2026-10-25", "08:00", "10:00", "", "budi@x.com", "Usher"),
		importRow("Ibadah Raya 2", "KEBAKTIAN", "2026-10-25", "10:00", "12:00", "YA", "", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Missing) != 0 || len(p.Rows) != 2 {
		t.Fatalf("missing=%v rows=%d", p.Missing, len(p.Rows))
	}

	r := p.Rows[0]
	if !r.Valid() || r.Line != 2 || r.Activity != "Ibadah Raya 1" ||
		r.JenisKegiatan != "KEBAKTIAN" || r.Assignee != "budi@x.com" || r.Skill != "Usher" {
		t.Errorf("row = %+v", r)
	}

	if r.Start == nil || r.Start.Format(time.RFC3339) != "2026-10-25T08:00:00+07:00" {
		t.Errorf("start = %v", r.Start)
	}
	if r.End == nil || r.End.Format(time.RFC3339) != "2026-10-25T10:00:00+07:00" {
		t.Errorf("end = %v", r.End)
	}
	if !r.PerluEvaluasi {
		t.Error("KEBAKTIAN should default to requiring evaluation")
	}
	if p.Rows[1].Line != 3 {
		t.Errorf("line = %d, want 3", p.Rows[1].Line)
	}
}

func TestParseRows_HeaderFleksibel(t *testing.T) {
	rows := [][]string{
		{"PERLU-EVALUASI", "waktu-selesai", "waktu mulai", "nama kegiatan", "tanggal", "jenis kegiatan"},
		{"TIDAK", "11:00", "09:00", "Latihan", "25/10/2026", "EVENT"},
	}

	p, err := parseRows(rows, wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Missing) != 0 || len(p.Rows) != 1 || !p.Rows[0].Valid() {
		t.Fatalf("missing=%v rows=%+v", p.Missing, p.Rows)
	}
	if p.Rows[0].Activity != "Latihan" || p.Rows[0].JenisKegiatan != "EVENT" {
		t.Errorf("row = %+v", p.Rows[0])
	}
	if p.Rows[0].PerluEvaluasi {
		t.Error("TIDAK should set PerluEvaluasi to false")
	}
}

func TestParseRows_KolomWajibHilang(t *testing.T) {
	p, err := parseRows([][]string{
		{"tanggal"},
		{"2026-10-25"},
	}, wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Missing) != 2 ||
		p.Missing[0] != "nama_kegiatan" ||
		p.Missing[1] != "waktu_mulai" {
		t.Errorf("missing = %v", p.Missing)
	}
}

func TestParseRows_ErrorPerBaris(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("A", "KEBAKTIAN", "bukan tanggal", "08:00", "10:00", "", "", ""),
		importRow("B", "KEBAKTIAN", "2026-10-25", "10:00", "09:00", "", "", ""),
		importRow("", "KEBAKTIAN", "2026-10-25", "08:00", "09:00", "", "", ""),
		importRow("D", "KEBAKTIAN", "2026-10-25", "xx", "09:00", "", "", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(p.Rows))
	}

	want := []string{"tanggal", "waktu_selesai", "nama_kegiatan", "waktu_mulai"}
	for i, key := range want {
		if p.Rows[i].Valid() || p.Rows[i].Errors[key] == "" {
			t.Errorf("row %d: errors = %v, want error in %q", p.Rows[i].Line, p.Rows[i].Errors, key)
		}
	}
}

func TestParseRows_BarisKosongDilewati(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("", "", "", "", "", "", "", ""),
		importRow("A", "", "2026-10-25", "08:00", "09:00", "", "", ""),
		importRow("  ", "", "", "", "", "", "", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Rows) != 1 || p.Rows[0].Line != 3 {
		t.Errorf("rows = %+v", p.Rows)
	}
}

func TestParseRows_Duplikat(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("Ibadah", "KEBAKTIAN", "2026-10-25", "08:00", "09:00", "", "Budi", ""),
		importRow("IBADAH", "KEBAKTIAN", "25/10/2026", "08:00", "09:00", "", "budi", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	if !p.Rows[0].Valid() {
		t.Errorf("first row should be valid: %v", p.Rows[0].Errors)
	}
	if p.Rows[1].Errors["baris"] != "duplikat dengan baris 2" {
		t.Errorf("errors = %v", p.Rows[1].Errors)
	}
}

func TestParseRows_SerialExcel(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("Ibadah", "KEBAKTIAN", "46320", "0.375", "0.5", "", "", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	r := p.Rows[0]
	if !r.Valid() || r.Start.Format("2006-01-02 15:04") != "2026-10-25 09:00" ||
		r.End.Format("15:04") != "12:00" {
		t.Errorf("row = %+v errors=%v", r, r.Errors)
	}
}

func TestParseRows_WaktuSelesaiOpsional(t *testing.T) {
	p, err := parseRows(sheet(
		importRow("Ibadah", "KEBAKTIAN", "2026-10-25", "08:00", "", "", "", ""),
	), wib)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Rows) != 1 || !p.Rows[0].Valid() {
		t.Fatalf("rows = %+v", p.Rows)
	}
	if p.Rows[0].End != nil || p.Rows[0].WaktuSelesai != "" {
		t.Errorf("end should be empty: %+v", p.Rows[0])
	}
}

func TestParseRows_JenisKegiatan(t *testing.T) {
	tests := []struct {
		jenis     string
		wantValid bool
		wantEval  bool
	}{
		{"KEBAKTIAN", true, true},
		{"TRAINING", true, false},
		{"EVENT", true, false},
		{"", true, false},
		{"LAINNYA", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.jenis, func(t *testing.T) {
			p, err := parseRows(sheet(
				importRow("Kegiatan", tc.jenis, "2026-10-25", "08:00", "", "", "", ""),
			), wib)
			if err != nil {
				t.Fatal(err)
			}

			got := p.Rows[0]
			if got.Valid() != tc.wantValid {
				t.Errorf("valid = %v, want %v; errors=%v", got.Valid(), tc.wantValid, got.Errors)
			}
			if got.PerluEvaluasi != tc.wantEval {
				t.Errorf("PerluEvaluasi = %v, want %v", got.PerluEvaluasi, tc.wantEval)
			}
		})
	}
}

func TestParseRows_PerluEvaluasi(t *testing.T) {
	tests := []struct {
		value string
		want  bool
		valid bool
	}{
		{"YA", true, true},
		{"TIDAK", false, true},
		{"", true, true},
		{"MUNGKIN", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			p, err := parseRows(sheet(
				importRow("Kegiatan", "KEBAKTIAN", "2026-10-25", "08:00", "", tc.value, "", ""),
			), wib)
			if err != nil {
				t.Fatal(err)
			}

			got := p.Rows[0]
			if got.Valid() != tc.valid {
				t.Errorf("valid = %v, want %v; errors=%v", got.Valid(), tc.valid, got.Errors)
			}
			if got.PerluEvaluasi != tc.want {
				t.Errorf("PerluEvaluasi = %v, want %v", got.PerluEvaluasi, tc.want)
			}
		})
	}
}

func TestParseRows_TerlaluBanyak(t *testing.T) {
	rows := sheet()
	for i := 0; i < MaxImportRows+1; i++ {
		rows = append(rows, importRow("Kegiatan", "EVENT", "2026-10-25", "08:00", "", "", "", ""))
	}

	if _, err := parseRows(rows, wib); err != ErrTooManyRows {
		t.Errorf("err = %v, want ErrTooManyRows", err)
	}
}

func TestPreviewRows(t *testing.T) {
	svc := NewService(nil, wib, nil)

	t.Run("bentrok petugas yang sama dilaporkan dengan nomor baris", func(t *testing.T) {
		resp, err := svc.previewRows(sheet(
			importRow("Ibadah", "KEBAKTIAN", "2026-10-25", "08:00", "10:00", "", "Budi", ""),
			importRow("Latihan", "EVENT", "2026-10-25", "09:00", "11:00", "", "budi", ""),
			importRow("Parkir", "EVENT", "2026-10-25", "09:00", "11:00", "", "Sari", ""),
		))
		if err != nil {
			t.Fatal(err)
		}

		if resp.TotalRows != 3 || resp.ValidRows != 3 || !resp.CanCommit {
			t.Errorf("resp = %+v", resp)
		}
		if len(resp.Clashes) != 1 || resp.Clashes[0].Lines[0] != 2 ||
			resp.Clashes[0].Lines[1] != 3 || resp.Clashes[0].Type != ClashTime {
			t.Errorf("clashes = %+v", resp.Clashes)
		}
	})

	t.Run("tanpa bentrok clashes berupa array kosong", func(t *testing.T) {
		resp, err := svc.previewRows(sheet(
			importRow("A", "EVENT", "2026-10-25", "08:00", "09:00", "", "Budi", ""),
			importRow("B", "EVENT", "2026-10-25", "09:00", "10:00", "", "Budi", ""),
		))
		if err != nil {
			t.Fatal(err)
		}

		if resp.Clashes == nil || len(resp.Clashes) != 0 {
			t.Errorf("clashes = %#v", resp.Clashes)
		}
	})

	t.Run("baris error tidak ikut deteksi bentrok dan menolak commit", func(t *testing.T) {
		resp, err := svc.previewRows(sheet(
			importRow("A", "EVENT", "2026-10-25", "08:00", "10:00", "", "Budi", ""),
			importRow("B", "EVENT", "2026-10-25", "09:00", "salah", "", "Budi", ""),
		))
		if err != nil {
			t.Fatal(err)
		}

		if resp.InvalidRows != 1 || resp.CanCommit || len(resp.Clashes) != 0 {
			t.Errorf("resp = %+v", resp)
		}
	})

	t.Run("kolom wajib hilang menjadi error validasi", func(t *testing.T) {
		if _, err := svc.previewRows([][]string{{"foo"}}); err == nil {
			t.Error("expected an error")
		}
	})
}
