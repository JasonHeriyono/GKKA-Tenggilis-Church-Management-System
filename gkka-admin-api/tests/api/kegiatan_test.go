package api_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestKegiatanAPIIntegration(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run Docker API integration tests")
	}

	identifier := os.Getenv("SIM_G_TEST_IDENTIFIER")
	password := os.Getenv("SIM_G_TEST_PASSWORD")
	if identifier == "" || password == "" {
		t.Fatal("set SIM_G_TEST_IDENTIFIER and SIM_G_TEST_PASSWORD")
	}

	api := loginAPI(t, identifier, password)
	t.Log("Super Admin login: PASS")

	status, body := api.request(t, http.MethodGet, "/kegiatan", nil, false)
	if status != http.StatusOK {
		t.Fatalf("public list kegiatan: expected 200, got %d: %s", status, body)
	}
	t.Log("Public list kegiatan: PASS")

	status, body = api.request(t, http.MethodGet, "/kegiatan?dari=invalid", nil, false)
	if status != http.StatusBadRequest {
		t.Errorf("invalid date filter: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid date filter rejected: PASS")
	}

	status, body = api.request(t, http.MethodGet, "/kegiatan/abc", nil, false)
	if status != http.StatusBadRequest {
		t.Errorf("invalid kegiatan ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid kegiatan ID rejected: PASS")
	}

	status, body = api.request(t, http.MethodGet, "/kegiatan/4294967295", nil, false)
	if status != http.StatusNotFound {
		t.Errorf("nonexistent kegiatan: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Nonexistent kegiatan returns 404: PASS")
	}

	status, body = api.request(t, http.MethodPost, "/kegiatan", map[string]any{
		"nama_kegiatan": "Unauthorized Test",
	}, false)
	if status != http.StatusUnauthorized && status != http.StatusForbidden {
		t.Errorf("unauthenticated create: expected 401 or 403, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated create rejected: PASS")
	}

	status, body = api.request(t, http.MethodPost, "/kegiatan", map[string]any{}, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("missing required fields: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Missing required fields rejected: PASS")
	}

	name := fmt.Sprintf("API Test Kegiatan %d", time.Now().UnixNano())
	tanggal := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

	payload := map[string]any{
		"nama_kegiatan":  name,
		"tanggal":        tanggal,
		"waktu_mulai":    "08:00",
		"waktu_selesai":  "09:00",
		"tema":           "API Integration Test",
		"lokasi":         "Testing Room",
		"perlu_evaluasi": false,
	}

	status, body = api.request(t, http.MethodPost, "/kegiatan", payload, true)
	if status != http.StatusCreated {
		t.Fatalf("create kegiatan: expected 201, got %d: %s", status, body)
	}

	kegiatanID := findID(body)
	if kegiatanID == "" {
		t.Fatalf("created kegiatan has no ID: %s", body)
	}
	t.Logf("Create kegiatan: PASS (ID %s)", kegiatanID)

	deleted := false
	defer func() {
		if deleted {
			return
		}

		cleanupStatus, cleanupBody := api.request(
			t, http.MethodDelete, "/kegiatan/"+kegiatanID, nil, true,
		)
		if cleanupStatus != http.StatusOK {
			t.Errorf("cleanup kegiatan failed: HTTP %d: %s", cleanupStatus, cleanupBody)
		}
	}()

	status, body = api.request(t, http.MethodGet, "/kegiatan/"+kegiatanID, nil, false)
	if status != http.StatusOK {
		t.Fatalf("get kegiatan: expected 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, name) {
		t.Errorf("kegiatan response does not contain %q: %s", name, body)
	} else {
		t.Log("Get kegiatan details: PASS")
	}

	updatedName := name + " Updated"
	payload["nama_kegiatan"] = updatedName

	status, body = api.request(t, http.MethodPut, "/kegiatan/"+kegiatanID, payload, true)
	if status != http.StatusOK {
		t.Fatalf("update kegiatan: expected 200, got %d: %s", status, body)
	}
	t.Log("Update kegiatan: PASS")

	status, body = api.request(
		t, http.MethodGet, "/kegiatan/"+kegiatanID+"/petugas", nil, true,
	)
	if status != http.StatusOK {
		t.Errorf("list petugas: expected 200, got %d: %s", status, body)
	} else {
		t.Log("List kegiatan petugas: PASS")
	}

	status, body = api.request(t, http.MethodDelete, "/kegiatan/"+kegiatanID, nil, true)
	if status != http.StatusOK {
		t.Fatalf("delete kegiatan: expected 200, got %d: %s", status, body)
	}
	deleted = true
	t.Log("Delete kegiatan: PASS")

	status, body = api.request(t, http.MethodGet, "/kegiatan/"+kegiatanID, nil, false)
	if status != http.StatusNotFound {
		t.Errorf("deleted kegiatan: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Deleted kegiatan returns 404: PASS")
	}
}

func TestScheduleAuthorizationAPI(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run Docker API integration tests")
	}

	api := &apiClient{
		baseURL: strings.TrimRight(os.Getenv("SIM_G_API_URL"), "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}

	if api.baseURL == "" {
		api.baseURL = defaultBaseURL
	}

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "personal schedule requires login",
			method: http.MethodGet,
			path:   "/jadwal/saya",
		},
		{
			name:   "schedule list requires Super Admin",
			method: http.MethodGet,
			path:   "/schedules",
		},
		{
			name:   "schedule import preview requires Super Admin",
			method: http.MethodPost,
			path:   "/schedules/import/preview",
		},
		{
			name:   "schedule import requires Super Admin",
			method: http.MethodPost,
			path:   "/schedules/import",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := api.request(t, tt.method, tt.path, nil, false)

			if status != http.StatusUnauthorized && status != http.StatusForbidden {
				t.Errorf("expected 401 or 403, got %d: %s", status, body)
			}
		})
	}
}
