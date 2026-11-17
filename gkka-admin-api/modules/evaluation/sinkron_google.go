package evaluation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"google.golang.org/api/forms/v1"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

// geserItem: item 0 di form = "Kode kebaktian" (kode_akses.go), pertanyaan mulai dari item 1.
const geserItem = 1

// SinkronGoogle menyamakan isi Google Form dengan pertanyaan aktif di database.
// Item lama di form diubah/dipindah (bukan dihapus lalu dibuat ulang) supaya ID-nya tetap,
// karena Google Sheets membuat KOLOM BARU untuk setiap ID pertanyaan baru.
func (service *Service) SinkronGoogle(ctx context.Context, jenis string) (*SinkronResponse, error) {
	jenis = strings.ToUpper(strings.TrimSpace(jenis)) // "mingguan " -> "MINGGUAN"
	if !jenisValid(jenis) {
		return nil, apperror.BadRequest("Jenis harus MINGGUAN atau TAHUNAN")
	}
	if service.form == nil {
		return nil, apperror.New(http.StatusServiceUnavailable, "Google Form belum dihubungkan, isi GOOGLE_FORM_ID_* di .env")
	}

	// kunci supaya tidak ada dua sinkron jalan bersamaan (bisa membuat pertanyaan dobel di form)
	service.kunciSinkron.Lock()
	defer service.kunciSinkron.Unlock()

	// 1. ambil pertanyaan aktif dari database (sudah urut berdasarkan urutan)
	aktif := true
	daftarPertanyaan, err := service.repo.ListPertanyaan(ctx, PertanyaanQuery{Jenis: jenis, Aktif: &aktif})
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if len(daftarPertanyaan) == 0 { // jangan sampai form jadi kosong karena admin belum mengisi pertanyaan
		return nil, apperror.BadRequest("Belum ada pertanyaan aktif untuk jenis " + jenis)
	}

	// 2. ambil isi Google Form sekarang
	form, err := service.form.Ambil(ctx, jenis)
	if err != nil {
		return nil, googleError(err)
	}

	// 3. item "Kode kebaktian" harus ada di paling atas
	form, err = service.pastikanItemKode(ctx, jenis, form)
	if err != nil {
		return nil, googleError(err)
	}

	// 4. hitung perubahan pertanyaan (item 1 dst), lalu kirim kalau ada
	daftarPerubahan, hasil := susunPerubahan(daftarPertanyaan, form.Items[geserItem:])
	if len(daftarPerubahan) > 0 {
		form, err = service.form.Ubah(ctx, jenis, daftarPerubahan)
		if err != nil {
			return nil, googleError(err)
		}
	}

	// 5. simpan ID Google ke database. Setelah sinkron, item ke-(i+1) di form = pertanyaan ke-i
	if len(form.Items)-geserItem != len(daftarPertanyaan) {
		return nil, apperror.Internal(fmt.Errorf("jumlah pertanyaan di form (%d) tidak sama dengan database (%d)", len(form.Items)-geserItem, len(daftarPertanyaan)))
	}
	for i := range daftarPertanyaan {
		item := form.Items[i+geserItem]
		daftarPertanyaan[i].GoogleItemID = &item.ItemId
		daftarPertanyaan[i].GoogleQuestionID = &item.QuestionItem.Question.QuestionId
	}
	if err := service.repo.SimpanIDGoogle(ctx, daftarPertanyaan); err != nil {
		return nil, apperror.Internal(err)
	}

	hasil.JenisKuesioner = jenis
	hasil.LinkForm = form.ResponderUri

	// isi form baru berubah, app jemaat harus membaca yang terbaru
	service.cacheForm.kunci.Lock()
	delete(service.cacheForm.isi, jenis)
	service.cacheForm.kunci.Unlock()

	// header ikut diganti sesuai kuesioner yang sedang dibuka, supaya admin langsung melihatnya
	kuesioner, err := service.repo.KuesionerDibuka(ctx, jenis, time.Now())
	if err == nil {
		if err := service.pasangHeader(ctx, jenis, kuesioner); err != nil {
			slog.WarnContext(ctx, "header Google Form gagal diganti", "jenis", jenis, "id_kuesioner", kuesioner.ID, "error", err)
		}
	} else if !errors.Is(err, ErrNotFound) {
		slog.WarnContext(ctx, "kuesioner yang dibuka gagal dibaca", "jenis", jenis, "error", err)
	}
	return &hasil, nil
}

// susunPerubahan menghitung daftar perubahan yang dikirim ke Google:
//  1. hapus item yang tidak dipakai lagi
//  2. untuk setiap pertanyaan: buat item baru, atau pindahkan/ubah item lamanya
//
// Google menjalankan perubahan satu per satu dan posisi item ikut bergeser,
// jadi posisi terbaru dicatat di urutanItem supaya index yang dikirim selalu benar.
// itemDiForm tidak termasuk item kode, jadi posisi yang dikirim ke Google ditambah geserItem.
func susunPerubahan(daftarPertanyaan []models.PertanyaanEvaluasi, itemDiForm []*forms.Item) ([]*forms.Request, SinkronResponse) {
	var daftarPerubahan []*forms.Request
	var hasil SinkronResponse

	itemLama := cariItemLama(daftarPertanyaan, itemDiForm)

	// item yang masih dipakai oleh pertanyaan, selain ini akan dihapus
	itemDipakai := map[string]bool{}
	for _, item := range itemLama {
		if item != nil {
			itemDipakai[item.ItemId] = true
		}
	}

	// urutanItem = daftar ID item di form, sesuai urutan dari atas ke bawah
	var urutanItem []string
	for _, item := range itemDiForm {
		urutanItem = append(urutanItem, item.ItemId)
	}

	// langkah 1: hapus item yang tidak dipakai, mulai dari bawah supaya index item di atasnya tidak bergeser
	for i := len(itemDiForm) - 1; i >= 0; i-- {
		if itemDipakai[itemDiForm[i].ItemId] {
			continue
		}
		daftarPerubahan = append(daftarPerubahan, perintahHapus(i+geserItem))
		urutanItem = slices.Delete(urutanItem, i, i+1) // buang ID ke-i dari daftar
		hasil.Dihapus++
	}

	// langkah 2: pertanyaan ke-i harus ada di posisi ke-i
	for i, pertanyaan := range daftarPertanyaan {
		item := itemLama[i]

		// belum punya item di form -> buat baru di posisi i
		if item == nil {
			daftarPerubahan = append(daftarPerubahan, perintahBuat(pertanyaan, i+geserItem))
			urutanItem = slices.Insert(urutanItem, i, "") // ID item baru belum diketahui, isi kosong dulu
			hasil.Ditambah++
			continue
		}

		// sudah punya item tapi posisinya beda -> pindahkan ke posisi i
		posisiSekarang := slices.Index(urutanItem, item.ItemId)
		if posisiSekarang != i {
			daftarPerubahan = append(daftarPerubahan, perintahPindah(posisiSekarang+geserItem, i+geserItem))
			urutanItem = slices.Delete(urutanItem, posisiSekarang, posisiSekarang+1)
			urutanItem = slices.Insert(urutanItem, i, item.ItemId)
			hasil.Dipindah++
		}

		// isinya beda (teks, tipe, wajib, atau opsi) -> ubah
		if !itemSama(item, pertanyaan) {
			daftarPerubahan = append(daftarPerubahan, perintahUbah(item, pertanyaan, i+geserItem))
			hasil.Diubah++
		}
	}
	return daftarPerubahan, hasil
}

// cariItemLama mencari item di form milik tiap pertanyaan (dari google_item_id yang tersimpan).
// Hasil ke-i = item milik pertanyaan ke-i, nil artinya harus dibuat item baru.
func cariItemLama(daftarPertanyaan []models.PertanyaanEvaluasi, itemDiForm []*forms.Item) []*forms.Item {
	itemByID := map[string]*forms.Item{}
	for _, item := range itemDiForm {
		itemByID[item.ItemId] = item
	}

	itemLama := make([]*forms.Item, len(daftarPertanyaan))
	sudahDiambil := map[string]bool{} // satu item hanya boleh dipakai satu pertanyaan
	for i, pertanyaan := range daftarPertanyaan {
		if pertanyaan.GoogleItemID == nil { // belum pernah dikirim ke Google
			continue
		}
		item := itemByID[*pertanyaan.GoogleItemID]
		if item == nil || sudahDiambil[item.ItemId] { // item sudah dihapus dari form, atau dobel
			continue
		}
		// beda kelompok tipe (misal teks -> pilihan) tidak bisa diubah di Google, jadi dibuat item baru
		if kelompokTipe(tipeDariGoogle(item)) != kelompokTipe(pertanyaan.TipeJawaban) {
			continue
		}
		itemLama[i] = item
		sudahDiambil[item.ItemId] = true
	}
	return itemLama
}

// perintahHapus: hapus item di posisi tertentu.
func perintahHapus(posisiItem int) *forms.Request {
	return &forms.Request{
		DeleteItem: &forms.DeleteItemRequest{Location: posisi(posisiItem)},
	}
}

// perintahBuat: buat item baru dari pertanyaan di posisi tertentu.
func perintahBuat(pertanyaan models.PertanyaanEvaluasi, posisiItem int) *forms.Request {
	return &forms.Request{
		CreateItem: &forms.CreateItemRequest{Item: buatItemGoogle(pertanyaan), Location: posisi(posisiItem)},
	}
}

// perintahPindah: pindahkan item dari satu posisi ke posisi lain.
func perintahPindah(dari, ke int) *forms.Request {
	return &forms.Request{
		MoveItem: &forms.MoveItemRequest{OriginalLocation: posisi(dari), NewLocation: posisi(ke)},
	}
}

// perintahUbah: timpa isi item lama dengan pertanyaan terbaru, ID-nya tetap memakai ID lama
// supaya kolom di Google Sheets tidak bertambah.
func perintahUbah(itemLama *forms.Item, pertanyaan models.PertanyaanEvaluasi, posisiItem int) *forms.Request {
	itemBaru := buatItemGoogle(pertanyaan)
	itemBaru.ItemId = itemLama.ItemId
	itemBaru.QuestionItem.Question.QuestionId = itemLama.QuestionItem.Question.QuestionId

	return &forms.Request{
		UpdateItem: &forms.UpdateItemRequest{
			Item:       itemBaru,
			Location:   posisi(posisiItem),
			UpdateMask: "title,questionItem", // bagian item yang ditimpa
		},
	}
}

// posisi: lokasi item di form (0 = paling atas).
// ForceSendFields wajib supaya index 0 tetap terkirim (tanpa ini angka 0 dianggap kosong).
func posisi(index int) *forms.Location {
	return &forms.Location{Index: int64(index), ForceSendFields: []string{"Index"}}
}

// buatItemGoogle mengubah pertanyaan dari database jadi item Google Form.
func buatItemGoogle(pertanyaan models.PertanyaanEvaluasi) *forms.Item {
	pertanyaanGoogle := &forms.Question{Required: pertanyaan.Wajib}

	switch pertanyaan.TipeJawaban {
	case models.TipeRating:
		pertanyaanGoogle.ScaleQuestion = &forms.ScaleQuestion{Low: 1, High: 5}
	case models.TipeTeksSingkat:
		pertanyaanGoogle.TextQuestion = &forms.TextQuestion{}
	case models.TipeParagraf:
		pertanyaanGoogle.TextQuestion = &forms.TextQuestion{Paragraph: true}
	default: // pilihan ganda, centang, dropdown
		pilihan := &forms.ChoiceQuestion{Type: tipeChoiceGoogle(pertanyaan.TipeJawaban)}
		for _, opsi := range pertanyaan.Opsi {
			pilihan.Options = append(pilihan.Options, &forms.Option{Value: opsi.Label})
		}
		pertanyaanGoogle.ChoiceQuestion = pilihan
	}

	return &forms.Item{
		Title:        pertanyaan.Teks,
		QuestionItem: &forms.QuestionItem{Question: pertanyaanGoogle},
	}
}

// itemSama: true kalau item di form sudah sama dengan pertanyaan, jadi tidak perlu diubah.
func itemSama(item *forms.Item, pertanyaan models.PertanyaanEvaluasi) bool {
	pertanyaanGoogle := item.QuestionItem.Question

	if item.Title != pertanyaan.Teks {
		return false
	}
	if pertanyaanGoogle.Required != pertanyaan.Wajib {
		return false
	}
	if tipeDariGoogle(item) != pertanyaan.TipeJawaban {
		return false
	}
	if pertanyaan.TipeJawaban == models.TipeRating { // skala harus 1 sampai 5
		skala := pertanyaanGoogle.ScaleQuestion
		return skala.Low == 1 && skala.High == 5
	}
	if tipePunyaOpsi(pertanyaan.TipeJawaban) { // opsi dan urutannya harus sama
		return slices.Equal(labelOpsiGoogle(pertanyaanGoogle), labelOpsiDatabase(pertanyaan))
	}
	return true
}

// labelOpsiGoogle: daftar label pilihan jawaban di Google Form.
func labelOpsiGoogle(pertanyaanGoogle *forms.Question) []string {
	var daftarLabel []string
	for _, opsi := range pertanyaanGoogle.ChoiceQuestion.Options {
		daftarLabel = append(daftarLabel, opsi.Value)
	}
	return daftarLabel
}

// labelOpsiDatabase: daftar label pilihan jawaban di database.
func labelOpsiDatabase(pertanyaan models.PertanyaanEvaluasi) []string {
	var daftarLabel []string
	for _, opsi := range pertanyaan.Opsi {
		daftarLabel = append(daftarLabel, opsi.Label)
	}
	return daftarLabel
}

// tipeDariGoogle membaca tipe item Google Form dengan nama tipe yang sama seperti di database.
// Hasil "" artinya bukan pertanyaan yang dikenal (misal judul bagian atau gambar).
func tipeDariGoogle(item *forms.Item) string {
	if item.QuestionItem == nil || item.QuestionItem.Question == nil {
		return ""
	}
	pertanyaanGoogle := item.QuestionItem.Question

	if pertanyaanGoogle.ScaleQuestion != nil {
		return models.TipeRating
	}
	if pertanyaanGoogle.TextQuestion != nil {
		if pertanyaanGoogle.TextQuestion.Paragraph {
			return models.TipeParagraf
		}
		return models.TipeTeksSingkat
	}
	if pertanyaanGoogle.ChoiceQuestion != nil {
		switch pertanyaanGoogle.ChoiceQuestion.Type {
		case "RADIO":
			return models.TipePilihanGanda
		case "CHECKBOX":
			return models.TipeCentang
		case "DROP_DOWN":
			return models.TipeDropdown
		}
	}
	return ""
}

// tipeChoiceGoogle: nama tipe pilihan versi Google Forms API.
func tipeChoiceGoogle(tipeJawaban string) string {
	switch tipeJawaban {
	case models.TipeCentang:
		return "CHECKBOX"
	case models.TipeDropdown:
		return "DROP_DOWN"
	}
	return "RADIO" // pilihan ganda
}

// kelompokTipe: Google bisa mengubah tipe di dalam satu kelompok (misal singkat -> paragraf),
// tapi tidak bisa antar kelompok (misal teks -> pilihan).
func kelompokTipe(tipeJawaban string) string {
	switch tipeJawaban {
	case models.TipeTeksSingkat, models.TipeParagraf:
		return "teks"
	case models.TipePilihanGanda, models.TipeCentang, models.TipeDropdown:
		return "pilihan"
	}
	return tipeJawaban
}

// googleError: form ID kosong -> 400, error lain dari Google -> 502.
// Pesan dari Google ikut ditampilkan supaya admin tahu cara memperbaikinya.
func googleError(err error) error {
	if errors.Is(err, errFormBelumDiisi) {
		return apperror.BadRequest("Form ID Google untuk jenis ini belum diisi di .env")
	}
	errGateway := apperror.New(http.StatusBadGateway, "Gagal sinkron ke Google Form: "+err.Error())
	errGateway.Err = err // disimpan untuk log
	return errGateway
}
