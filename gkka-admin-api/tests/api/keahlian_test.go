package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func loginAPI(t *testing.T, identifier, password string) *apiClient {
	t.Helper()

	baseURL := strings.TrimRight(os.Getenv("SIM_G_API_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	api := &apiClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}

	status, body := api.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": identifier,
			"password":   password,
		}, false)

	if status != http.StatusOK {
		t.Fatalf("login failed: HTTP %d: %s", status, body)
	}

	api.token = findString(body, "access_token", "accessToken", "token")
	if api.token == "" {
		t.Fatalf("login response has no access token: %s", body)
	}

	return api
}

func TestKeahlianAPIIntegration(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run Docker API integration tests")
	}

	adminID := os.Getenv("SIM_G_TEST_IDENTIFIER")
	adminPassword := os.Getenv("SIM_G_TEST_PASSWORD")
	if adminID == "" || adminPassword == "" {
		t.Fatal("set SIM_G_TEST_IDENTIFIER and SIM_G_TEST_PASSWORD")
	}

	api := loginAPI(t, adminID, adminPassword)
	t.Log("Super Admin login: PASS")

	// Listing skills is public.
	status, body := api.request(t, http.MethodGet, "/keahlian", nil, false)
	if status != http.StatusOK {
		t.Errorf("public GET /keahlian: expected 200, got %d: %s",
			status, body)
	} else {
		t.Log("Public skill listing: PASS")
	}

	// List existing skills.
	status, body = api.request(t, http.MethodGet, "/keahlian", nil, true)
	if status != http.StatusOK {
		t.Fatalf("list skills: expected 200, got %d: %s", status, body)
	}
	t.Log("List skills: PASS")

	// Invalid IDs should be rejected.
	status, body = api.request(t, http.MethodGet, "/keahlian/abc", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid skill ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid skill ID rejected: PASS")
	}

	// Missing required name should be rejected.
	status, body = api.request(t, http.MethodPost, "/keahlian",
		map[string]any{}, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("missing skill name: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Missing skill name rejected: PASS")
	}

	// Create a uniquely named skill for lifecycle tests.
	name := fmt.Sprintf("API Test Skill %d", time.Now().UnixNano())
	status, body = api.request(t, http.MethodPost, "/keahlian",
		map[string]string{"nama": name}, true)
	if status != http.StatusCreated {
		t.Fatalf("create skill: expected 201, got %d: %s", status, body)
	}

	skillID := findID(body)
	if skillID == "" {
		t.Fatalf("created skill has no ID in response: %s", body)
	}
	t.Logf("Create skill: PASS (ID %s)", skillID)

	deleted := false
	defer func() {
		if deleted {
			return
		}
		cleanupStatus, cleanupBody := api.request(t, http.MethodDelete,
			"/keahlian/"+skillID, nil, true)
		if cleanupStatus != http.StatusOK {
			t.Errorf("cleanup skill %s failed: HTTP %d: %s",
				skillID, cleanupStatus, cleanupBody)
		} else {
			t.Log("Cleanup: temporary skill deleted")
		}
	}()

	// Retrieve the new skill.
	status, body = api.request(t, http.MethodGet,
		"/keahlian/"+skillID, nil, true)
	if status != http.StatusOK {
		t.Fatalf("get skill: expected 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, name) {
		t.Errorf("skill detail does not contain %q: %s", name, body)
	} else {
		t.Log("Get skill details: PASS")
	}

	// Update the new skill.
	updatedName := name + " Updated"
	status, body = api.request(t, http.MethodPut,
		"/keahlian/"+skillID,
		map[string]string{"nama": updatedName}, true)
	if status != http.StatusOK {
		t.Fatalf("update skill: expected 200, got %d: %s", status, body)
	}
	t.Log("Update skill: PASS")

	// A missing skill should return 404.
	status, body = api.request(t, http.MethodGet,
		"/keahlian/4294967295", nil, true)
	if status != http.StatusNotFound {
		t.Errorf("nonexistent skill: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Nonexistent skill returns 404: PASS")
	}

	// Member-assignment tests use the existing test user.
	userID := os.Getenv("SIM_G_TEST_USER_ID")
	if userID == "" {
		t.Log("Membership tests skipped: set SIM_G_TEST_USER_ID")
		return
	}

	userID = strings.TrimSpace(userID)
	if _, err := fmt.Sscan(userID, new(uint)); err != nil {
		t.Fatalf("SIM_G_TEST_USER_ID must be numeric: %q", userID)
	}

	// Assign the temporary skill to the test user.
	status, body = api.request(t, http.MethodPost,
		"/keahlian/"+skillID+"/jemaat",
		map[string]any{"id_user": mustUint(t, userID)}, true)
	if status != http.StatusOK {
		t.Fatalf("assign skill: expected 200, got %d: %s", status, body)
	}
	t.Log("Assign skill to member: PASS")

	// Confirm the skill appears in the user's skill list.
	status, body = api.request(t, http.MethodGet,
		"/users/"+userID+"/keahlian", nil, true)
	if status != http.StatusOK {
		t.Fatalf("list member skills: expected 200, got %d: %s", status, body)
	}

	var result struct {
		Data struct {
			Keahlian []struct {
				ID uint `json:"id"`
			} `json:"keahlian"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatalf("decode member skills: %v; body: %s", err, body)
	}

	found := false
	for _, item := range result.Data.Keahlian {
		if fmt.Sprint(item.ID) == skillID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("assigned skill %s not found in member skill list: %s",
			skillID, body)
	} else {
		t.Log("List member skills: PASS")
	}

	// Remove the temporary assignment.
	status, body = api.request(t, http.MethodDelete,
		"/keahlian/"+skillID+"/jemaat/"+userID, nil, true)
	if status != http.StatusOK {
		t.Fatalf("remove member skill: expected 200, got %d: %s", status, body)
	}
	t.Log("Remove member skill: PASS")

	// Explicitly delete the temporary skill.
	status, body = api.request(t, http.MethodDelete,
		"/keahlian/"+skillID, nil, true)
	if status != http.StatusOK {
		t.Fatalf("delete skill: expected 200, got %d: %s", status, body)
	}
	deleted = true
	t.Log("Delete skill: PASS")

	status, body = api.request(t, http.MethodGet,
		"/keahlian/"+skillID, nil, true)
	if status != http.StatusNotFound {
		t.Errorf("deleted skill: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Deleted skill returns 404: PASS")
	}
}

func TestKeahlianJemaatPermissions(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run Docker API integration tests")
	}

	identifier := os.Getenv("SIM_G_JEMAAT_IDENTIFIER")
	password := os.Getenv("SIM_G_JEMAAT_PASSWORD")
	if identifier == "" || password == "" {
		t.Skip("set SIM_G_JEMAAT_IDENTIFIER and SIM_G_JEMAAT_PASSWORD to run role tests")
	}

	api := loginAPI(t, identifier, password)
	t.Log("Jemaat login: PASS")

	tests := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodPost, "/keahlian", map[string]string{"nama": "Permission Test"}},
		{http.MethodPut, "/keahlian/1", map[string]string{"nama": "Permission Test"}},
		{http.MethodDelete, "/keahlian/1", nil},
		{http.MethodPost, "/keahlian/1/jemaat", map[string]any{"id_user": 2}},
		{http.MethodDelete, "/keahlian/1/jemaat/2", nil},
		{http.MethodGet, "/users/2/keahlian", nil},
		{http.MethodPut, "/users/2/keahlian", map[string]any{"id_keahlian": []int{}}},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			status, body := api.request(t, tt.method, tt.path, tt.body, true)
			if status != http.StatusForbidden {
				t.Errorf("expected 403 Forbidden, got %d: %s", status, body)
			}
		})
	}
}
