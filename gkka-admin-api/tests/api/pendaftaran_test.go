package api_test

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestPendaftaranAPIIntegration(t *testing.T) {
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	shortSuffix := suffix[len(suffix)-8:]

	// Admin-only endpoints must reject unauthenticated requests.
	status, body := admin.request(
		t, http.MethodGet, "/pendaftaran", nil, false,
	)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated list: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated access rejected: PASS")
	}

	// Invalid IDs should be rejected.
	status, body = admin.request(
		t, http.MethodGet, "/pendaftaran/abc", nil, true,
	)
	if status != http.StatusBadRequest {
		t.Errorf("invalid registration ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid registration ID rejected: PASS")
	}

	// Create an existing Jemaat without a password for the signup flow.
	existingCode := "P" + shortSuffix
	existingName := "API Signup User " + suffix

	status, body = admin.request(
		t, http.MethodPost, "/users",
		map[string]string{
			"kode_jemaat": existingCode,
			"nama":        existingName,
			"role":        "Jemaat",
		}, true,
	)
	if status != http.StatusCreated {
		t.Fatalf("create existing Jemaat: expected 201, got %d: %s", status, body)
	}
	existingUserID := findID(body)
	if existingUserID == "" {
		t.Fatalf("could not find existing Jemaat ID: %s", body)
	}
	t.Log("Create existing Jemaat for signup: PASS")

	// Existing Jemaat submits a signup request.
	signupEmail := "api.signup." + suffix + "@example.com"
	signupPassword := "SignupPass2026!"

	status, body = admin.request(
		t, http.MethodPost, "/auth/signup",
		map[string]any{
			"kode_jemaat":      existingCode,
			"nama":             existingName,
			"email":            signupEmail,
			"password":         signupPassword,
			"persetujuan_data": true,
		}, false,
	)
	if status != http.StatusCreated {
		t.Fatalf("signup: expected 201, got %d: %s", status, body)
	}
	signupID := findID(body)
	if signupID == "" {
		t.Fatalf("could not find signup registration ID: %s", body)
	}
	t.Log("Existing Jemaat signup: PASS")

	// Create a new registration that will be approved.
	approvedEmail := "api.approved." + suffix + "@example.com"
	approvedPassword := "ApprovedPass2026!"
	approvedName := "API Approved User " + suffix

	status, body = admin.request(
		t, http.MethodPost, "/auth/daftar",
		map[string]any{
			"nama":             approvedName,
			"email":            approvedEmail,
			"no_telepon":       "0812" + shortSuffix,
			"password":         approvedPassword,
			"persetujuan_data": true,
			"jenis_kelamin":    "L",
			"tanggal_lahir":    "2000-01-01",
		}, false,
	)
	if status != http.StatusCreated {
		t.Fatalf("new registration: expected 201, got %d: %s", status, body)
	}
	approvedID := findID(body)
	if approvedID == "" {
		t.Fatalf("could not find new registration ID: %s", body)
	}
	t.Log("New Jemaat registration: PASS")

	// Create a registration that will be rejected.
	rejectedEmail := "api.rejected." + suffix + "@example.com"

	status, body = admin.request(
		t, http.MethodPost, "/auth/daftar",
		map[string]any{
			"nama":             "API Rejected User " + suffix,
			"email":            rejectedEmail,
			"persetujuan_data": true,
			"jenis_kelamin":    "P",
			"tanggal_lahir":    "2001-01-01",
		}, false,
	)
	if status != http.StatusCreated {
		t.Fatalf("registration for rejection: expected 201, got %d: %s", status, body)
	}
	rejectedID := findID(body)
	if rejectedID == "" {
		t.Fatalf("could not find rejected registration ID: %s", body)
	}
	t.Log("Registration for rejection: PASS")

	// Create another registration that will be marked as a duplicate.
	duplicateEmail := "api.duplicate." + suffix + "@example.com"

	status, body = admin.request(
		t, http.MethodPost, "/auth/daftar",
		map[string]any{
			"nama":             "API Duplicate User " + suffix,
			"email":            duplicateEmail,
			"persetujuan_data": true,
			"jenis_kelamin":    "L",
			"tanggal_lahir":    "1999-01-01",
		}, false,
	)
	if status != http.StatusCreated {
		t.Fatalf("registration for duplicate: expected 201, got %d: %s", status, body)
	}
	duplicateID := findID(body)
	if duplicateID == "" {
		t.Fatalf("could not find duplicate registration ID: %s", body)
	}
	t.Log("Registration for duplicate marking: PASS")

	// List pending registrations.
	status, body = admin.request(
		t, http.MethodGet, "/pendaftaran?status=MENUNGGU", nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("list pending registrations: expected 200, got %d: %s", status, body)
	}
	t.Log("List pending registrations: PASS")

	// Filter registrations by type.
	status, body = admin.request(
		t, http.MethodGet, "/pendaftaran?jenis=BARU", nil, true,
	)
	if status != http.StatusOK {
		t.Errorf("filter new registrations: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Filter registrations by type: PASS")
	}

	// Invalid filters should be rejected.
	status, body = admin.request(
		t, http.MethodGet, "/pendaftaran?status=INVALID", nil, true,
	)
	if status != http.StatusBadRequest {
		t.Errorf("invalid status filter: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid registration filter rejected: PASS")
	}

	// Retrieve registration details.
	status, body = admin.request(
		t, http.MethodGet, "/pendaftaran/"+approvedID, nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("get registration details: expected 200, got %d: %s", status, body)
	}
	t.Log("Get registration details: PASS")

	// Approve the existing Jemaat signup.
	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+signupID+"/setujui",
		map[string]string{
			"catatan_admin": "API integration test approval",
		}, true,
	)
	if status != http.StatusOK {
		t.Fatalf("approve existing Jemaat signup: expected 200, got %d: %s", status, body)
	}
	t.Log("Approve existing Jemaat signup: PASS")

	// The existing account should now be usable.
	signupAccount := loginAPI(t, signupEmail, signupPassword)
	if signupAccount.token == "" {
		t.Fatal("signup account login returned an empty token")
	}
	t.Log("Approved existing Jemaat can log in: PASS")

	// Approve the new Jemaat registration.
	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+approvedID+"/setujui",
		map[string]string{
			"catatan_admin": "API integration test approval",
		}, true,
	)
	if status != http.StatusOK {
		t.Fatalf("approve new registration: expected 200, got %d: %s", status, body)
	}
	t.Log("Approve new Jemaat registration: PASS")

	// The newly created account should be usable.
	approvedAccount := loginAPI(t, approvedEmail, approvedPassword)
	if approvedAccount.token == "" {
		t.Fatal("approved account login returned an empty token")
	}
	t.Log("Approved new Jemaat can log in: PASS")

	// A registration that has already been processed cannot be approved again.
	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+approvedID+"/setujui",
		map[string]string{}, true,
	)
	if status != http.StatusConflict {
		t.Errorf("approve already processed registration: expected 409, got %d: %s", status, body)
	} else {
		t.Log("Cannot approve an already processed registration: PASS")
	}

	// Reject another pending registration.
	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+rejectedID+"/tolak",
		map[string]string{
			"catatan_admin": "API integration test rejection",
		}, true,
	)
	if status != http.StatusOK {
		t.Fatalf("reject registration: expected 200, got %d: %s", status, body)
	}
	t.Log("Reject registration: PASS")

	// Mark the last registration as a duplicate of the existing Jemaat.
	existingUserIDNumber, err := strconv.ParseUint(existingUserID, 10, 64)
	if err != nil {
		t.Fatalf("invalid existing user ID %q: %v", existingUserID, err)
	}

	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+duplicateID+"/duplikat",
		map[string]any{
			"id_user":       existingUserIDNumber,
			"catatan_admin": "API integration test duplicate",
		}, true,
	)

	// Processed registrations cannot be processed a second time.
	status, body = admin.request(
		t, http.MethodPost, "/pendaftaran/"+rejectedID+"/tolak",
		map[string]string{
			"catatan_admin": "Second rejection attempt",
		}, true,
	)
	if status != http.StatusConflict {
		t.Errorf("reject already processed registration: expected 409, got %d: %s", status, body)
	} else {
		t.Log("Cannot reject an already processed registration: PASS")
	}

	t.Log("Pendaftaran API integration tests completed")
}
