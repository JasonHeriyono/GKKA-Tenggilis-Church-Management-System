package schedule

import (
	"fmt"
	"time"
)

// ClashType menjelaskan jenis bentrok yang ditemukan.
type ClashType string

const (
	// ClashTime: satu orang ditugaskan di dua kegiatan yang waktunya tumpang tindih.
	ClashTime ClashType = "time"
	// ClashSkill: orang yang ditugaskan tidak punya skill yang dibutuhkan kegiatan.
	ClashSkill ClashType = "skill"
)

// Slot adalah satu penugasan dalam jadwal (dari Excel import maupun dari DB).
// Dibuat sebagai struct sendiri agar clash detection murni (tanpa DB / models),
// jadi bisa dites sebelum ERD final. Tipe ID disesuaikan dengan models nanti.
type Slot struct {
	Title           string
	Start           time.Time
	End             time.Time
	UserID          uint // 0 = belum ada orang yang ditugaskan
	RequiredSkillID uint // 0 = kegiatan tidak butuh skill khusus
}

// Clash adalah satu temuan bentrok. A dan B adalah INDEKS pada slice input,
// bukan ID database, supaya baris Excel yang belum tersimpan tetap bisa dilaporkan.
// Untuk ClashSkill hanya A yang dipakai (B = -1).
type Clash struct {
	Type    ClashType `json:"type"`
	A       int       `json:"a"`
	B       int       `json:"b"`
	Message string    `json:"message"`
}

// Overlaps: dua rentang waktu bentrok jika a.start < b.end && b.start < a.end.
// Kegiatan yang berurutan (a.end == b.start) TIDAK dianggap bentrok.
// Rentang tidak valid (end <= start) dianggap tidak bentrok; validasi
// rentang dilakukan terpisah di parser/service.
func Overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	if !aEnd.After(aStart) || !bEnd.After(bStart) {
		return false
	}
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// DetectTimeClashes mencari pasangan slot dengan UserID sama yang waktunya tumpang tindih.
func DetectTimeClashes(slots []Slot) []Clash {
	var clashes []Clash
	for i := 0; i < len(slots); i++ {
		if slots[i].UserID == 0 {
			continue
		}
		for j := i + 1; j < len(slots); j++ {
			if slots[i].UserID != slots[j].UserID {
				continue
			}
			if Overlaps(slots[i].Start, slots[i].End, slots[j].Start, slots[j].End) {
				clashes = append(clashes, Clash{
					Type: ClashTime,
					A:    i,
					B:    j,
					Message: fmt.Sprintf("bentrok waktu: %q dan %q (user %d)",
						slots[i].Title, slots[j].Title, slots[i].UserID),
				})
			}
		}
	}
	return clashes
}

// DetectSkillClashes memeriksa apakah orang yang ditugaskan punya skill yang dibutuhkan.
// userSkills: UserID -> himpunan SkillID milik user tersebut.
func DetectSkillClashes(slots []Slot, userSkills map[uint]map[uint]bool) []Clash {
	var clashes []Clash
	for i, s := range slots {
		if s.UserID == 0 || s.RequiredSkillID == 0 {
			continue
		}
		if !userSkills[s.UserID][s.RequiredSkillID] {
			clashes = append(clashes, Clash{
				Type: ClashSkill,
				A:    i,
				B:    -1,
				Message: fmt.Sprintf("skill tidak sesuai: user %d tidak punya skill %d untuk %q",
					s.UserID, s.RequiredSkillID, s.Title),
			})
		}
	}
	return clashes
}

// DetectClashes menjalankan semua pemeriksaan (waktu lalu skill) dan menggabungkan hasilnya.
func DetectClashes(slots []Slot, userSkills map[uint]map[uint]bool) []Clash {
	return append(DetectTimeClashes(slots), DetectSkillClashes(slots, userSkills)...)
}
