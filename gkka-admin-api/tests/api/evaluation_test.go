package api_test

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sim-g-backend/config"
	"sim-g-backend/database"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestEvaluationAPIIntegration(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run API integration tests")
	}

	adminID := os.Getenv("SIM_G_TEST_IDENTIFIER")
	adminPassword := os.Getenv("SIM_G_TEST_PASSWORD")
	if adminID == "" || adminPassword == "" {
		t.Fatal("set SIM_G_TEST_IDENTIFIER and SIM_G_TEST_PASSWORD")
	}

	admin := loginAPI(t, adminID, adminPassword)
	t.Log("Super Admin login: PASS")

	// Public form endpoint.
	status, body := admin.request(t, http.MethodGet, "/evaluasi/form", nil, false)
	if status != http.StatusBadRequest {
		t.Errorf("missing form type: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Public form rejects missing type: PASS")
	}

	status, body = admin.request(t, http.MethodGet, "/evaluasi/form?jenis=INVALID", nil, false)
	if status != http.StatusBadRequest {
		t.Errorf("invalid form type: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Public form rejects invalid type: PASS")
	}

	// Authentication and authorization.
	status, body = admin.request(t, http.MethodGet, "/pertanyaan", nil, false)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated question list: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated question list rejected: PASS")
	}

	status, body = admin.request(t, http.MethodPost, "/pertanyaan", map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"teks_pertanyaan": "Unauthorized test",
		"tipe_jawaban":    "RATING",
		"urutan":          1,
	}, false)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated question creation: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated question creation rejected: PASS")
	}

	// Optional Jemaat permission tests.
	jemaatID := os.Getenv("SIM_G_JEMAAT_IDENTIFIER")
	jemaatPassword := os.Getenv("SIM_G_JEMAAT_PASSWORD")

	if jemaatID != "" && jemaatPassword != "" {
		jemaat := loginAPI(t, jemaatID, jemaatPassword)

		status, body = jemaat.request(t, http.MethodGet, "/pertanyaan", nil, true)
		if status != http.StatusForbidden {
			t.Errorf("Jemaat question list: expected 403, got %d: %s", status, body)
		} else {
			t.Log("Jemaat cannot read questions: PASS")
		}

		status, body = jemaat.request(t, http.MethodPost, "/pertanyaan", map[string]any{
			"jenis_kuesioner": "MINGGUAN",
			"teks_pertanyaan": "Unauthorized test",
			"tipe_jawaban":    "RATING",
			"urutan":          1,
		}, true)
		if status != http.StatusForbidden {
			t.Errorf("Jemaat question creation: expected 403, got %d: %s", status, body)
		} else {
			t.Log("Jemaat cannot create questions: PASS")
		}
	} else {
		t.Log("Jemaat permission tests skipped: credentials not configured")
	}

	// Optional Majelis permission tests.
	majelisID := os.Getenv("SIM_G_MAJELIS_IDENTIFIER")
	majelisPassword := os.Getenv("SIM_G_MAJELIS_PASSWORD")

	if (majelisID == "") != (majelisPassword == "") {
		t.Fatal("configure both SIM_G_MAJELIS_IDENTIFIER and SIM_G_MAJELIS_PASSWORD")
	}

	var majelis *apiClient
	if majelisID != "" {
		majelis = loginAPI(t, majelisID, majelisPassword)
		t.Log("Majelis login: PASS")
	} else {
		t.Log("Majelis permission tests skipped: credentials not configured")
	}

	// Invalid filters.
	status, body = admin.request(t, http.MethodGet, "/pertanyaan?jenis=INVALID", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid question type filter: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid question type filter rejected: PASS")
	}

	status, body = admin.request(t, http.MethodGet, "/pertanyaan?aktif=INVALID", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid active filter: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid active filter rejected: PASS")
	}

	status, body = admin.request(t, http.MethodGet, "/pertanyaan/not-a-number", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid question ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid question ID rejected: PASS")
	}

	// Create a question with a unique text and ordering number.
	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	questionText := "API integration question " + unique
	urutan := int(time.Now().UnixNano()%1000000) + 1

	question := map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"teks_pertanyaan": questionText,
		"tipe_jawaban":    "RATING",
		"urutan":          urutan,
		"wajib":           true,
		"opsi":            []string{},
	}

	status, body = admin.request(t, http.MethodPost, "/pertanyaan", question, true)
	if status != http.StatusCreated {
		t.Fatalf("create question: expected 201, got %d: %s", status, body)
	}

	questionID := findID(body)
	if questionID == "" {
		t.Fatalf("create question response has no ID: %s", body)
	}
	t.Log("Create question: PASS")

	detailPath := "/pertanyaan/" + questionID

	// Check that Majelis can read the newly created question.
	if majelis != nil {
		status, body = majelis.request(t, http.MethodGet, detailPath, nil, true)
		if status != http.StatusOK {
			t.Errorf("Majelis get question: expected 200, got %d: %s", status, body)
		} else {
			t.Log("Majelis can read questions: PASS")
		}

		status, body = majelis.request(t, http.MethodPost, "/pertanyaan", question, true)
		if status != http.StatusForbidden {
			t.Errorf("Majelis create question: expected 403, got %d: %s", status, body)
		} else {
			t.Log("Majelis cannot create questions: PASS")
		}
	}

	// Get question details.
	status, body = admin.request(t, http.MethodGet, detailPath, nil, true)
	if status != http.StatusOK {
		t.Fatalf("get question: expected 200, got %d: %s", status, body)
	}
	t.Log("Get question details: PASS")

	// List questions.
	status, body = admin.request(t, http.MethodGet, "/pertanyaan?jenis=MINGGUAN&aktif=true", nil, true)
	if status != http.StatusOK {
		t.Errorf("list questions: expected 200, got %d: %s", status, body)
	} else {
		t.Log("List and filter questions: PASS")
	}

	// Invalid answer type/options combination.
	invalidQuestion := map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"teks_pertanyaan": "Invalid rating question",
		"tipe_jawaban":    "RATING",
		"urutan":          1,
		"opsi":            []string{"Option A"},
	}

	status, body = admin.request(t, http.MethodPost, "/pertanyaan", invalidQuestion, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("invalid rating options: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Invalid answer options rejected: PASS")
	}

	// Choice questions must have options.
	invalidQuestion["tipe_jawaban"] = "PILIHAN_GANDA"
	invalidQuestion["opsi"] = []string{}

	status, body = admin.request(t, http.MethodPost, "/pertanyaan", invalidQuestion, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("choice question without options: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Choice question without options rejected: PASS")
	}

	// Duplicate option labels must be rejected.
	invalidQuestion["opsi"] = []string{"Option A", "option a"}

	status, body = admin.request(t, http.MethodPost, "/pertanyaan", invalidQuestion, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("duplicate options: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Duplicate options rejected: PASS")
	}

	// Update the created question.
	updatedQuestion := map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"teks_pertanyaan": questionText + " updated",
		"tipe_jawaban":    "TEKS_SINGKAT",
		"urutan":          urutan,
		"wajib":           false,
		"opsi":            []string{},
	}

	status, body = admin.request(t, http.MethodPut, detailPath, updatedQuestion, true)
	if status != http.StatusOK {
		t.Fatalf("update question: expected 200, got %d: %s", status, body)
	}
	t.Log("Update question: PASS")

	// Deactivate the question.
	status, body = admin.request(t, http.MethodPatch, detailPath+"/aktif", map[string]bool{
		"is_aktif": false,
	}, true)
	if status != http.StatusOK {
		t.Fatalf("deactivate question: expected 200, got %d: %s", status, body)
	}
	t.Log("Deactivate question: PASS")

	status, body = admin.request(t, http.MethodGet, "/pertanyaan?jenis=MINGGUAN&aktif=false", nil, true)
	if status != http.StatusOK {
		t.Errorf("list inactive questions: expected 200, got %d: %s", status, body)
	} else {
		t.Log("List inactive questions: PASS")
	}

	// Reactivate before deleting.
	status, body = admin.request(t, http.MethodPatch, detailPath+"/aktif", map[string]bool{
		"is_aktif": true,
	}, true)
	if status != http.StatusOK {
		t.Fatalf("reactivate question: expected 200, got %d: %s", status, body)
	}
	t.Log("Reactivate question: PASS")

	// Delete the question.
	status, body = admin.request(t, http.MethodDelete, detailPath, nil, true)
	if status != http.StatusOK {
		t.Fatalf("delete question: expected 200, got %d: %s", status, body)
	}
	t.Log("Delete question: PASS")

	// Soft-deleted questions should no longer be accessible.
	status, body = admin.request(t, http.MethodGet, detailPath, nil, true)
	if status != http.StatusNotFound {
		t.Errorf("get deleted question: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Deleted question is no longer accessible: PASS")
	}

	// Nonexistent question ID.
	status, body = admin.request(t, http.MethodGet, "/pertanyaan/999999999", nil, true)
	if status != http.StatusNotFound {
		t.Errorf("nonexistent question: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Nonexistent question rejected: PASS")
	}

	// Invalid update and activation IDs.
	status, body = admin.request(t, http.MethodPut, "/pertanyaan/999999999", updatedQuestion, true)
	if status != http.StatusNotFound {
		t.Errorf("update nonexistent question: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Update nonexistent question rejected: PASS")
	}

	status, body = admin.request(t, http.MethodPatch, "/pertanyaan/999999999/aktif", map[string]bool{
		"is_aktif": true,
	}, true)
	if status != http.StatusNotFound {
		t.Errorf("activate nonexistent question: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Activate nonexistent question rejected: PASS")
	}

	// The public form endpoint accepts only supported questionnaire types.
	// A valid request may return 503 if Google Forms credentials are unavailable.
	status, body = admin.request(t, http.MethodGet, "/evaluasi/form?jenis=MINGGUAN", nil, false)
	if status != http.StatusOK && status != http.StatusServiceUnavailable {
		t.Errorf("public weekly form: expected 200 or 503, got %d: %s", status, body)
	} else {
		t.Log("Public weekly form endpoint: PASS")
	}

	// Weekly questionnaire integration tests.
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatalf("load project .env: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load database config: %v", err)
	}

	db, err := database.Connect(cfg, slog.Default())
	if err != nil {
		t.Fatalf("connect to database for questionnaire cleanup: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			t.Errorf("close cleanup database connection: %v", err)
		}
	}()

	kegiatanName := fmt.Sprintf("API Test Evaluation %d", time.Now().UnixNano())
	kegiatanDate := time.Now().AddDate(0, 0, 30)
	kegiatanTanggal := kegiatanDate.Format("2006-01-02")
	questionnaireTitle := fmt.Sprintf("Evaluasi %s, %s", kegiatanName, kegiatanDate.Format("02-01-2006"))
	annualYear := 2100
	annualTitle := fmt.Sprintf("Evaluasi Tahunan %d", annualYear)
	var kegiatanID string

	// Always clean up the questionnaire before deleting its Kegiatan.
	defer func() {
		result := db.Exec(
			"UPDATE kuesioner_evaluasi SET deleted_at = NOW() WHERE judul_form = ? AND deleted_at IS NULL",
			questionnaireTitle,
		)
		if result.Error != nil {
			t.Errorf("cleanup questionnaire: %v", result.Error)
		}

		if kegiatanID != "" {
			cleanupStatus, cleanupBody := admin.request(
				t,
				http.MethodDelete,
				"/kegiatan/"+kegiatanID,
				nil,
				true,
			)
			if cleanupStatus != http.StatusOK {
				t.Errorf("cleanup temporary Kegiatan: expected 200, got %d: %s", cleanupStatus, cleanupBody)
			} else {
				t.Log("Temporary Kegiatan cleanup: PASS")
			}
		}
		result = db.Exec(
			"UPDATE kuesioner_evaluasi SET deleted_at = NOW() WHERE judul_form = ? AND deleted_at IS NULL",
			annualTitle,
		)
		if result.Error != nil {
			t.Errorf("cleanup annual questionnaire: %v", result.Error)
		}
	}()

	// Unauthenticated users cannot access questionnaire management.
	status, body = admin.request(t, http.MethodGet, "/kuesioner", nil, false)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated questionnaire list: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated questionnaire list rejected: PASS")
	}

	status, body = admin.request(t, http.MethodPost, "/kuesioner", map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"judul_form":      questionnaireTitle,
	}, false)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated questionnaire creation: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated questionnaire creation rejected: PASS")
	}

	// Invalid questionnaire filter.
	status, body = admin.request(t, http.MethodGet, "/kuesioner?jenis=INVALID", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid questionnaire filter: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid questionnaire filter rejected: PASS")
	}

	// Create a temporary Kegiatan for the weekly questionnaire.
	kegiatanPayload := map[string]any{
		"nama_kegiatan":  kegiatanName,
		"tanggal":        kegiatanTanggal,
		"waktu_mulai":    "08:00",
		"waktu_selesai":  "09:00",
		"tema":           "Weekly Evaluation Test",
		"lokasi":         "Testing Room",
		"perlu_evaluasi": true,
	}

	status, body = admin.request(t, http.MethodPost, "/kegiatan", kegiatanPayload, true)
	if status != http.StatusCreated {
		t.Fatalf("create temporary Kegiatan: expected 201, got %d: %s", status, body)
	}

	kegiatanID = findID(body)
	if kegiatanID == "" {
		t.Fatalf("could not extract temporary Kegiatan ID: %s", body)
	}
	t.Log("Create temporary Kegiatan: PASS")

	kegiatanIDNumber, err := strconv.ParseUint(kegiatanID, 10, 64)
	if err != nil {
		t.Fatalf("invalid temporary Kegiatan ID %q: %v", kegiatanID, err)
	}

	questionnairePayload := map[string]any{
		"jenis_kuesioner": "MINGGUAN",
		"judul_form":      questionnaireTitle,
		"id_kegiatan":     kegiatanIDNumber,
		"tema":            "Weekly Evaluation Test",
		"pembicara":       "API Test",
	}

	status, body = admin.request(t, http.MethodPost, "/kuesioner", questionnairePayload, true)
	if status != http.StatusCreated {
		t.Fatalf("create weekly questionnaire: expected 200, got %d: %s", status, body)
	}
	t.Log("Create weekly questionnaire: PASS")

	// Confirm it appears in the weekly and unfiltered lists.
	status, body = admin.request(t, http.MethodGet, "/kuesioner?jenis=MINGGUAN", nil, true)
	if status != http.StatusOK || !strings.Contains(body, questionnaireTitle) {
		t.Fatalf("list weekly questionnaires: expected 200 and created questionnaire, got %d: %s", status, body)
	}
	t.Log("List weekly questionnaires: PASS")

	status, body = admin.request(t, http.MethodGet, "/kuesioner", nil, true)
	if status != http.StatusOK || !strings.Contains(body, questionnaireTitle) {
		t.Errorf("list all questionnaires: expected 200 and created questionnaire, got %d: %s", status, body)
	} else {
		t.Log("List all questionnaires: PASS")
	}

	// Create an annual questionnaire.
	annualPayload := map[string]any{
		"jenis_kuesioner": "TAHUNAN",
		"judul_form":      annualTitle,
		"tahun":           annualYear,
		"dibuka_pada":     fmt.Sprintf("%d-01-01 10:00:00", annualYear),
		"ditutup_pada":    fmt.Sprintf("%d-01-02 23:59:00", annualYear),
	}

	status, body = admin.request(t, http.MethodPost, "/kuesioner", annualPayload, true)
	if status != http.StatusCreated {
		t.Fatalf("create annual questionnaire: expected 201, got %d: %s", status, body)
	}
	t.Log("Create annual questionnaire: PASS")

	// Retrieve the generated questionnaire ID from MySQL.
	var questionnaireID uint64
	result := db.Raw(
		"SELECT id_kuesioner FROM kuesioner_evaluasi WHERE judul_form = ? AND deleted_at IS NULL",
		questionnaireTitle,
	).Scan(&questionnaireID)

	if result.Error != nil {
		t.Fatalf("retrieve created questionnaire ID: %v", result.Error)
	}
	if result.RowsAffected == 0 || questionnaireID == 0 {
		t.Fatalf("created questionnaire ID not found")
	}

	questionnairePath := fmt.Sprintf("/kuesioner/%d/jadwal", questionnaireID)

	// Update its schedule with a valid opening and closing time.
	opening := time.Now().AddDate(0, 0, 2).Truncate(time.Second)
	closing := opening.Add(2 * time.Hour)

	schedulePayload := map[string]string{
		"dibuka_pada":  opening.Format("2006-01-02 15:04:05"),
		"ditutup_pada": closing.Format("2006-01-02 15:04:05"),
	}

	status, body = admin.request(t, http.MethodPut, questionnairePath, schedulePayload, true)
	if status != http.StatusOK {
		t.Errorf("update questionnaire schedule: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Update weekly questionnaire schedule: PASS")
	}
	var initialOpening, initialClosing sql.NullTime

	initialResult := db.Raw(
		"SELECT dibuka_pada, ditutup_pada FROM kuesioner_evaluasi WHERE id_kuesioner = ?",
		questionnaireID,
	).Row().Scan(&initialOpening, &initialClosing)

	if initialResult != nil {
		t.Errorf("retrieve initially updated schedule: %v", initialResult)
	} else if !initialOpening.Valid || !initialClosing.Valid {
		t.Errorf("initial schedule contains NULL: opening=%v closing=%v",
			initialOpening, initialClosing)
	} else if !initialOpening.Time.Equal(opening) || !initialClosing.Time.Equal(closing) {
		t.Errorf("initial schedule mismatch: opening=%v closing=%v",
			initialOpening.Time, initialClosing.Time)
	} else {
		t.Log("Initial schedule persisted correctly: PASS")
	}

	// Updating only the opening time must preserve the closing time.
	updatedOpening := opening.Add(30 * time.Minute)

	status, body = admin.request(t, http.MethodPut, questionnairePath, map[string]string{
		"dibuka_pada": updatedOpening.Format("2006-01-02 15:04:05"),
	}, true)

	if status != http.StatusOK {
		t.Errorf("partial questionnaire schedule update: expected 200, got %d: %s", status, body)
	} else {
		var persistedOpening, persistedClosing sql.NullTime

		result := db.Raw(
			"SELECT dibuka_pada, ditutup_pada FROM kuesioner_evaluasi WHERE id_kuesioner = ?",
			questionnaireID,
		).Row().Scan(&persistedOpening, &persistedClosing)

		if result != nil {
			t.Errorf("retrieve partially updated schedule: %v", result)
		} else if !persistedOpening.Valid || !persistedClosing.Valid {
			t.Errorf("schedule contains NULL: opening=%v closing=%v", persistedOpening, persistedClosing)
		} else if !persistedOpening.Time.Equal(updatedOpening) ||
			!persistedClosing.Time.Equal(closing) {
			t.Errorf("partial update did not preserve schedule: opening=%v closing=%v",
				persistedOpening.Time, persistedClosing.Time)
		} else {
			t.Log("Partial questionnaire schedule update: PASS")
		}
	}

	// Closing before opening must be rejected.
	invalidSchedule := map[string]string{
		"dibuka_pada":  closing.Format("2006-01-02 15:04:05"),
		"ditutup_pada": opening.Format("2006-01-02 15:04:05"),
	}
	status, body = admin.request(t, http.MethodPut, questionnairePath, invalidSchedule, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("invalid questionnaire schedule: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Invalid questionnaire schedule rejected: PASS")
	}

	// Quick action: open questionnaire.
	status, body = admin.request(t, http.MethodPut, questionnairePath, map[string]string{
		"status": "BUKA",
	}, true)

	if status != http.StatusOK {
		t.Errorf("open questionnaire: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Open questionnaire: PASS")
	}

	// Quick action: close questionnaire.
	status, body = admin.request(t, http.MethodPut, questionnairePath, map[string]string{
		"status": "TUTUP",
	}, true)

	if status != http.StatusOK {
		t.Errorf("close questionnaire: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Close questionnaire: PASS")
	}

	// Nonexistent questionnaire ID.
	status, body = admin.request(t, http.MethodPut, "/kuesioner/999999999/jadwal", schedulePayload, true)
	if status != http.StatusNotFound {
		t.Errorf("nonexistent questionnaire: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Nonexistent questionnaire rejected: PASS")
	}

	// Malformed questionnaire ID.
	status, body = admin.request(t, http.MethodPut, "/kuesioner/invalid/jadwal", schedulePayload, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid questionnaire ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid questionnaire ID rejected: PASS")
	}

	// Confirm the generated ID is numeric.
	if _, err := strconv.ParseUint(questionID, 10, 64); err != nil {
		t.Errorf("created question ID is not numeric: %q", questionID)
	}
}
