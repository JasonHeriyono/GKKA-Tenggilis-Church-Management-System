package evaluation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/forms/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"sim-g-backend/models"
)

// GoogleForm: fungsi yang dipakai service untuk membaca dan mengubah Google Form.
// Dibuat interface supaya saat unit test bisa diganti form palsu (tanpa internet).
type GoogleForm interface {
	Ambil(ctx context.Context, jenis string) (*forms.Form, error)
	Ubah(ctx context.Context, jenis string, daftarPerubahan []*forms.Request) (*forms.Form, error)
	AmbilRespon(ctx context.Context, jenis string) ([]*forms.FormResponse, error)
}

// googleForm: implementasi GoogleForm memakai Google Forms API + service account.
type googleForm struct {
	api                 *forms.Service
	formID              map[string]string // jenis kuesioner -> ID Google Form
	emailServiceAccount string            // form harus di-share ke email ini sebagai Editor
}

// NewGoogleForm membaca file JSON service account lalu menyiapkan koneksi ke Google Forms API.
// Mengembalikan nil (tanpa error) kalau belum ada form ID di .env, artinya fitur Google dimatikan.
func NewGoogleForm(fileCredential, formMingguan, formTahunan string) (GoogleForm, error) {
	if formMingguan == "" && formTahunan == "" {
		return nil, nil
	}

	isiFile, err := os.ReadFile(fileCredential)
	if err != nil {
		return nil, fmt.Errorf("file credential Google %q tidak bisa dibaca: %w", fileCredential, err)
	}
	// scope = izin yang diminta: ubah isi form + baca jawaban
	jwtConfig, err := google.JWTConfigFromJSON(isiFile, forms.FormsBodyScope, forms.FormsResponsesReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("%q bukan file JSON service account yang valid: %w", fileCredential, err)
	}
	api, err := forms.NewService(context.Background(), option.WithHTTPClient(jwtConfig.Client(context.Background())))
	if err != nil {
		return nil, err
	}

	return &googleForm{
		api: api,
		formID: map[string]string{
			models.KuesionerMingguan: formMingguan,
			models.KuesionerTahunan:  formTahunan,
		},
		emailServiceAccount: jwtConfig.Email,
	}, nil
}

// errFormBelumDiisi: form ID untuk jenis ini masih kosong di .env.
var errFormBelumDiisi = errors.New("form ID Google untuk jenis ini belum diisi di .env")

// Ambil membaca isi Google Form (judul, daftar pertanyaan, dll).
func (g *googleForm) Ambil(ctx context.Context, jenis string) (*forms.Form, error) {
	formID := g.formID[jenis]
	if formID == "" {
		return nil, errFormBelumDiisi
	}
	form, err := g.api.Forms.Get(formID).Context(ctx).Do()
	if err != nil {
		return nil, g.jelaskanError(err)
	}
	return form, nil
}

// Ubah mengirim semua perubahan dalam SATU panggilan API (kalau satu gagal, semua batal),
// lalu mengembalikan isi form setelah diubah.
func (g *googleForm) Ubah(ctx context.Context, jenis string, daftarPerubahan []*forms.Request) (*forms.Form, error) {
	formID := g.formID[jenis]
	if formID == "" {
		return nil, errFormBelumDiisi
	}
	permintaan := &forms.BatchUpdateFormRequest{Requests: daftarPerubahan, IncludeFormInResponse: true}
	hasil, err := g.api.Forms.BatchUpdate(formID, permintaan).Context(ctx).Do()
	if err != nil {
		return nil, g.jelaskanError(err)
	}
	if hasil.Form == nil { // jaga-jaga kalau Google tidak mengirim isi form terbaru
		return g.Ambil(ctx, jenis)
	}
	return hasil.Form, nil
}

// AmbilRespon membaca semua kiriman form. Google membaginya per halaman (maks 5000), jadi dibaca sampai habis.
func (g *googleForm) AmbilRespon(ctx context.Context, jenis string) ([]*forms.FormResponse, error) {
	formID := g.formID[jenis]
	if formID == "" {
		return nil, errFormBelumDiisi
	}
	var semua []*forms.FormResponse
	halaman := ""
	for {
		panggilan := g.api.Forms.Responses.List(formID).PageSize(5000).Context(ctx)
		if halaman != "" {
			panggilan = panggilan.PageToken(halaman)
		}
		hasil, err := panggilan.Do()
		if err != nil {
			return nil, g.jelaskanError(err)
		}
		semua = append(semua, hasil.Responses...)
		if hasil.NextPageToken == "" {
			return semua, nil
		}
		halaman = hasil.NextPageToken
	}
}

// jelaskanError mengubah error dari Google jadi pesan yang memberi tahu cara memperbaikinya.
func (g *googleForm) jelaskanError(err error) error {
	var errGoogle *googleapi.Error
	if !errors.As(err, &errGoogle) {
		return err
	}
	switch errGoogle.Code {
	case http.StatusForbidden:
		return fmt.Errorf("akses ditolak (403): pastikan Google Forms API sudah di-enable dan form sudah di-share ke %s sebagai Editor", g.emailServiceAccount)
	case http.StatusNotFound:
		return errors.New("form tidak ditemukan (404): cek GOOGLE_FORM_ID_*, ambil dari URL editor /forms/d/<ID>/edit")
	case http.StatusUnauthorized:
		return errors.New("service account tidak valid (401): buat ulang key JSON di Google Cloud")
	}
	return fmt.Errorf("Google Forms API error %d: %s", errGoogle.Code, errGoogle.Message)
}
