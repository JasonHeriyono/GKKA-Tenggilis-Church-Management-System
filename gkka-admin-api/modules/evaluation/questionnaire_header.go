package evaluation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/forms/v1"

	"sim-g-backend/models"
)

// Header Google Form (judul + deskripsi di atas pertanyaan) diganti tiap minggu sesuai
// kebaktian yang sedang dievaluasi. Formatnya dari template admin (tabel pengaturan),
// kalau belum ada pakai format bawaan di susunHeader.

// templateHeader: format judul/deskripsi dari admin, berisi variabel seperti {nama_kegiatan}.
type templateHeader struct {
	Judul     string
	Deskripsi string
}

// kunciHeader: nama kunci di tabel pengaturan, contoh "header_form.MINGGUAN.judul".
func kunciHeader(jenis string) (kunciJudul, kunciDeskripsi string) {
	return "header_form." + jenis + ".judul", "header_form." + jenis + ".deskripsi"
}

// ambilTemplateHeader: nil kalau admin belum pernah menyimpan template (pakai format bawaan).
func (service *Service) ambilTemplateHeader(ctx context.Context, jenis string) (*templateHeader, error) {
	kunciJudul, kunciDeskripsi := kunciHeader(jenis)
	isi, err := service.repo.AmbilPengaturan(ctx, kunciJudul, kunciDeskripsi)
	if err != nil {
		return nil, err
	}
	judul, ada := isi[kunciJudul]
	if !ada {
		return nil, nil
	}
	return &templateHeader{Judul: judul, Deskripsi: isi[kunciDeskripsi]}, nil
}

// pasangHeader menulis header kuesioner ke Google Form, hanya kalau isinya beda dengan
// yang sekarang ada di form (dibaca dari cache infoGoogle, jadi tidak tiap request ke Google).
func (service *Service) pasangHeader(ctx context.Context, jenis string, kuesioner *models.KuesionerEvaluasi) error {
	template, err := service.ambilTemplateHeader(ctx, jenis)
	if err != nil {
		return err
	}
	judul, deskripsi := susunHeader(kuesioner, template)

	service.cacheForm.kunci.Lock()
	defer service.cacheForm.kunci.Unlock()

	simpanan, ada := service.cacheForm.isi[jenis]
	if ada && simpanan.judulGoogle == judul && simpanan.deskripsiGoogle == deskripsi {
		return nil
	}

	perubahan := []*forms.Request{{
		UpdateFormInfo: &forms.UpdateFormInfoRequest{
			Info:       &forms.Info{Title: judul, Description: deskripsi},
			UpdateMask: "title,description",
		},
	}}
	form, err := service.form.Ubah(ctx, jenis, perubahan)
	if err != nil {
		return err
	}
	if ada && form.Info != nil {
		simpanan.judulGoogle, simpanan.deskripsiGoogle = form.Info.Title, form.Info.Description
		service.cacheForm.isi[jenis] = simpanan
	}
	return nil
}

// susunHeader: tema dan pembicara diambil dari salinan di kuesioner (tidak ikut berubah kalau
// kegiatan diedit setelah evaluasi berjalan), jam dan nats Alkitab dari kegiatan.
func susunHeader(kuesioner *models.KuesionerEvaluasi, template *templateHeader) (judul, deskripsi string) {
	if template != nil {
		judul = isiTemplate(template.Judul, kuesioner)
		if strings.TrimSpace(judul) == "" {
			judul = kuesioner.JudulForm // Google menolak judul kosong
		}
		return judul, isiTemplate(template.Deskripsi, kuesioner)
	}

	judul = kuesioner.JudulForm
	var baris []string

	kegiatan := kuesioner.Kegiatan
	if kegiatan != nil {
		judul = "Evaluasi " + kegiatan.Nama
		baris = append(baris, fmt.Sprintf("%s, %s pukul %s",
			kegiatan.Nama, tanggalIndonesia(kegiatan.Tanggal), jamMenit(kegiatan.WaktuMulai)))
	}
	if isi := teks(kuesioner.Pembicara); isi != "" {
		baris = append(baris, "Pembicara: "+isi)
	}
	if isi := teks(kuesioner.Tema); isi != "" {
		baris = append(baris, "Tema: "+isi)
	}
	if kegiatan != nil {
		if isi := teks(kegiatan.NatsAlkitab); isi != "" {
			baris = append(baris, "Nats Alkitab: "+isi)
		}
	}
	if kuesioner.DitutupPada != nil {
		ditutup := *kuesioner.DitutupPada
		baris = append(baris, "", fmt.Sprintf("Form dapat diisi sampai %s pukul %s.",
			tanggalIndonesia(ditutup), ditutup.Format("15:04")))
	}
	return judul, strings.Join(baris, "\n")
}

// isiTemplate mengganti variabel {..} dengan data kuesioner dan kegiatannya.
func isiTemplate(template string, kuesioner *models.KuesionerEvaluasi) string {
	nama := kuesioner.JudulForm // kuesioner TAHUNAN tidak punya kegiatan
	var tanggal, jam, jamSelesai, nats, komunitas, ditutup string
	if kegiatan := kuesioner.Kegiatan; kegiatan != nil {
		nama = kegiatan.Nama
		tanggal = tanggalIndonesia(kegiatan.Tanggal)
		jam = jamMenit(kegiatan.WaktuMulai)
		jamSelesai = jamMenit(teks(kegiatan.WaktuSelesai)) // kosong kalau tidak diisi
		nats = teks(kegiatan.NatsAlkitab)
		if kegiatan.Komunitas != nil {
			komunitas = kegiatan.Komunitas.Nama
		}
	}
	if kuesioner.DitutupPada != nil {
		ditutup = tanggalIndonesia(*kuesioner.DitutupPada) + " pukul " + kuesioner.DitutupPada.Format("15:04")
	}

	pengganti := strings.NewReplacer(
		"{nama_kegiatan}", nama,
		"{tema}", teks(kuesioner.Tema),
		"{pembicara}", teks(kuesioner.Pembicara),
		"{tanggal}", tanggal,
		"{jam_mulai}", jam,
		"{jam_selesai}", jamSelesai,
		"{komunitas}", komunitas,
		"{nats_alkitab}", nats,
		"{ditutup}", ditutup,
	)
	return pengganti.Replace(template)
}

var (
	namaHari = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	namaBulan = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
)

// tanggalIndonesia: "Minggu, 1 November 2026"
func tanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%s, %d %s %d", namaHari[t.Weekday()], t.Day(), namaBulan[t.Month()-1], t.Year())
}

// jamMenit: "07:00:00" -> "07:00"
func jamMenit(jam string) string {
	if len(jam) >= 5 {
		return jam[:5]
	}
	return jam
}

func teks(isi *string) string {
	if isi == nil {
		return ""
	}
	return strings.TrimSpace(*isi)
}
