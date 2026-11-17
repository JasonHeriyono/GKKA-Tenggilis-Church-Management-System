package schedule

import (
	"testing"
	"time"
)

// at membuat waktu pada hari yang sama, jam:menit tertentu.
func at(h, m int) time.Time {
	return time.Date(2026, 10, 25, h, m, 0, 0, time.UTC)
}

func TestOverlaps(t *testing.T) {
	tests := []struct {
		name string
		a1   time.Time
		a2   time.Time
		b1   time.Time
		b2   time.Time
		want bool
	}{
		{"tumpang tindih sebagian", at(8, 0), at(10, 0), at(9, 0), at(11, 0), true},
		{"B di dalam A", at(8, 0), at(12, 0), at(9, 0), at(10, 0), true},
		{"A di dalam B", at(9, 0), at(10, 0), at(8, 0), at(12, 0), true},
		{"berurutan (end == start) tidak bentrok", at(8, 0), at(9, 0), at(9, 0), at(10, 0), false},
		{"terpisah jauh", at(8, 0), at(9, 0), at(13, 0), at(14, 0), false},
		{"rentang A tidak valid", at(10, 0), at(9, 0), at(9, 0), at(11, 0), false},
		{"durasi nol", at(9, 0), at(9, 0), at(8, 0), at(10, 0), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Overlaps(tc.a1, tc.a2, tc.b1, tc.b2); got != tc.want {
				t.Errorf("Overlaps() = %v, want %v", got, tc.want)
			}
			// Hasilnya harus simetris.
			if got := Overlaps(tc.b1, tc.b2, tc.a1, tc.a2); got != tc.want {
				t.Errorf("Overlaps() terbalik = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDetectTimeClashes(t *testing.T) {
	t.Run("user sama, waktu bentrok", func(t *testing.T) {
		slots := []Slot{
			{Title: "Ibadah 1", Start: at(8, 0), End: at(10, 0), UserID: 1},
			{Title: "Latihan Musik", Start: at(9, 0), End: at(11, 0), UserID: 1},
		}
		got := DetectTimeClashes(slots)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].Type != ClashTime || got[0].A != 0 || got[0].B != 1 {
			t.Errorf("clash = %+v", got[0])
		}
	})

	t.Run("user berbeda tidak bentrok", func(t *testing.T) {
		slots := []Slot{
			{Title: "Ibadah 1", Start: at(8, 0), End: at(10, 0), UserID: 1},
			{Title: "Ibadah 1 (usher)", Start: at(8, 0), End: at(10, 0), UserID: 2},
		}
		if got := DetectTimeClashes(slots); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("berurutan tidak bentrok", func(t *testing.T) {
		slots := []Slot{
			{Title: "Ibadah 1", Start: at(8, 0), End: at(9, 0), UserID: 1},
			{Title: "Ibadah 2", Start: at(9, 0), End: at(10, 0), UserID: 1},
		}
		if got := DetectTimeClashes(slots); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("slot tanpa user diabaikan", func(t *testing.T) {
		slots := []Slot{
			{Title: "A", Start: at(8, 0), End: at(10, 0), UserID: 0},
			{Title: "B", Start: at(9, 0), End: at(11, 0), UserID: 0},
		}
		if got := DetectTimeClashes(slots); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("tiga slot saling bentrok menghasilkan tiga pasangan", func(t *testing.T) {
		slots := []Slot{
			{Title: "A", Start: at(8, 0), End: at(12, 0), UserID: 1},
			{Title: "B", Start: at(9, 0), End: at(11, 0), UserID: 1},
			{Title: "C", Start: at(10, 0), End: at(13, 0), UserID: 1},
		}
		if got := DetectTimeClashes(slots); len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})
}

func TestDetectSkillClashes(t *testing.T) {
	skills := map[uint]map[uint]bool{
		1: {10: true},           // user 1 punya skill 10
		2: {10: true, 20: true}, // user 2 punya skill 10 dan 20
	}

	t.Run("skill tidak dimiliki", func(t *testing.T) {
		slots := []Slot{{Title: "Singer", Start: at(8, 0), End: at(9, 0), UserID: 1, RequiredSkillID: 20}}
		got := DetectSkillClashes(slots, skills)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].Type != ClashSkill || got[0].A != 0 || got[0].B != -1 {
			t.Errorf("clash = %+v", got[0])
		}
	})

	t.Run("skill dimiliki", func(t *testing.T) {
		slots := []Slot{{Title: "Singer", Start: at(8, 0), End: at(9, 0), UserID: 2, RequiredSkillID: 20}}
		if got := DetectSkillClashes(slots, skills); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("tanpa kebutuhan skill atau tanpa user diabaikan", func(t *testing.T) {
		slots := []Slot{
			{Title: "Umum", Start: at(8, 0), End: at(9, 0), UserID: 1, RequiredSkillID: 0},
			{Title: "Kosong", Start: at(8, 0), End: at(9, 0), UserID: 0, RequiredSkillID: 20},
		}
		if got := DetectSkillClashes(slots, skills); len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("user tidak ada di map dianggap tidak punya skill", func(t *testing.T) {
		slots := []Slot{{Title: "Singer", Start: at(8, 0), End: at(9, 0), UserID: 99, RequiredSkillID: 10}}
		if got := DetectSkillClashes(slots, skills); len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})
}

func TestDetectClashes_Gabungan(t *testing.T) {
	skills := map[uint]map[uint]bool{1: {10: true}}
	slots := []Slot{
		{Title: "Ibadah", Start: at(8, 0), End: at(10, 0), UserID: 1, RequiredSkillID: 10},
		{Title: "Latihan", Start: at(9, 0), End: at(11, 0), UserID: 1, RequiredSkillID: 20}, // bentrok waktu + skill 20 tidak ada
	}
	got := DetectClashes(slots, skills)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Type != ClashTime || got[1].Type != ClashSkill {
		t.Errorf("urutan jenis = %v, %v", got[0].Type, got[1].Type)
	}
}
