package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const defaultBaseURL = "http://localhost:8080/gkki/api"

var (
	apiLoginMu   sync.Mutex
	lastAPILogin time.Time
)

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func TestKomunitasAPIIntegration(t *testing.T) {
	// Integration tests require explicit opt-in because they modify real data.
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run Docker API integration tests")
	}

	baseURL := strings.TrimRight(os.Getenv("SIM_G_API_URL"), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	identifier := os.Getenv("SIM_G_TEST_IDENTIFIER")
	password := os.Getenv("SIM_G_TEST_PASSWORD")
	if identifier == "" || password == "" {
		t.Fatal("set SIM_G_TEST_IDENTIFIER and SIM_G_TEST_PASSWORD")
	}

	api := &apiClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	// Authenticate against the real API.
	loginBody := map[string]any{
		"identifier": identifier,
		"password":   password,
	}

	loginStatus, loginResponse := api.request(
		t, http.MethodPost, "/auth/login", loginBody, false,
	)
	if loginStatus != http.StatusOK {
		t.Fatalf("login failed: HTTP %d: %s", loginStatus, loginResponse)
	}

	api.token = findString(loginResponse,
		"access_token", "accessToken", "token",
	)
	if api.token == "" {
		t.Fatalf("login succeeded but no access token was found in response: %s",
			loginResponse)
	}

	t.Log("Login succeeded")

	// Listing communities is public.
	status, body := api.request(
		t, http.MethodGet, "/komunitas", nil, false,
	)
	if status != http.StatusOK {
		t.Errorf(
			"public community list: expected HTTP 200, got %d: %s",
			status, body,
		)
	} else {
		t.Log("Public community listing: PASS")
	}

	// Verify that listing communities works.
	status, body = api.request(
		t, http.MethodGet, "/komunitas", nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("list communities: expected HTTP 200, got %d: %s", status, body)
	}
	t.Log("List communities: PASS")

	// Invalid IDs and malformed request bodies should be rejected.
	status, body = api.request(
		t, http.MethodGet, "/komunitas/abc", nil, true,
	)
	if status != http.StatusBadRequest {
		t.Errorf("invalid community ID: expected HTTP 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid community ID: PASS")
	}

	status, body = api.requestRaw(
		t, http.MethodPost, "/komunitas", `{"nama":`, true,
	)
	if status < 400 || status >= 500 {
		t.Errorf("invalid JSON: expected a 4xx response, got %d: %s", status, body)
	} else {
		t.Log("Invalid JSON: PASS")
	}

	// Use a unique name to avoid collisions with existing communities.
	name := fmt.Sprintf("API Test %d", time.Now().UnixNano())
	createBody := map[string]any{
		"nama":      name,
		"deskripsi": "Temporary community created by automated API integration tests",
	}

	status, body = api.request(
		t, http.MethodPost, "/komunitas", createBody, true,
	)
	if status != http.StatusCreated {
		t.Fatalf("create community: expected HTTP 201, got %d: %s", status, body)
	}

	communityID := findID(body)
	if communityID == "" {
		t.Fatalf("community created but its ID could not be found in response: %s", body)
	}
	t.Logf("Create community: PASS (ID %s)", communityID)

	// A zero user ID should be rejected by request validation.
	status, body = api.request(
		t,
		http.MethodPost,
		"/komunitas/"+communityID+"/anggota",
		map[string]any{
			"id_user":  0,
			"is_ketua": false,
		},
		true,
	)
	if status < 400 || status >= 500 {
		t.Errorf(
			"invalid member request: expected a 4xx response, got %d: %s",
			status, body,
		)
	} else {
		t.Logf("Invalid member request rejected: PASS (HTTP %d)", status)
	}

	// Delete explicitly at the end; clean up automatically if the test fails.
	communityDeleted := false

	defer func() {
		if communityDeleted {
			return
		}

		cleanupStatus, response := api.request(
			t, http.MethodDelete, "/komunitas/"+communityID, nil, true,
		)
		if cleanupStatus != http.StatusOK {
			t.Errorf(
				"cleanup: failed to delete test community %s: HTTP %d: %s",
				communityID, cleanupStatus, response,
			)
		} else {
			t.Log("Cleanup: test community deleted")
		}
	}()

	// Retrieve the newly created community.
	status, body = api.request(
		t, http.MethodGet, "/komunitas/"+communityID, nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("get community: expected HTTP 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, name) {
		t.Errorf("get community response does not contain created name %q: %s", name, body)
	} else {
		t.Log("Get community details: PASS")
	}

	// Update its name and description.
	updatedName := name + " Updated"
	updateBody := map[string]any{
		"nama":      updatedName,
		"deskripsi": "Updated by automated API integration tests",
	}

	status, body = api.request(
		t, http.MethodPut, "/komunitas/"+communityID, updateBody, true,
	)
	if status != http.StatusOK {
		t.Fatalf("update community: expected HTTP 200, got %d: %s", status, body)
	}
	t.Log("Update community: PASS")

	// Membership tests require the ID of an existing test user.
	userID := os.Getenv("SIM_G_TEST_USER_ID")
	if userID == "" {
		t.Log("Membership tests skipped: set SIM_G_TEST_USER_ID to an existing user's ID")
		return
	}
	if _, err := strconv.ParseUint(userID, 10, 32); err != nil {
		t.Fatalf("SIM_G_TEST_USER_ID must be a positive numeric user ID")
	}

	// Add the user to the test community.
	status, body = api.request(
		t, http.MethodPost, "/komunitas/"+communityID+"/anggota",
		map[string]any{"id_user": mustUint(t, userID), "is_ketua": false}, true,
	)
	if status != http.StatusCreated {
		t.Fatalf("add community member: expected HTTP 201, got %d: %s", status, body)
	}
	t.Log("Add community member: PASS")

	// Set the member as leader.
	memberPath := "/komunitas/" + communityID + "/anggota/" + userID
	status, body = api.request(
		t, http.MethodPatch, memberPath,
		map[string]any{"is_ketua": true}, true,
	)
	if status != http.StatusOK {
		t.Fatalf("set community leader: expected HTTP 200, got %d: %s", status, body)
	}
	t.Log("Set community leader: PASS")

	// Remove the member again before deleting the community.
	status, body = api.request(
		t, http.MethodDelete, memberPath, nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("remove community member: expected HTTP 200, got %d: %s", status, body)
	}
	t.Log("Remove community member: PASS")

	// Explicitly test deleting the community.
	status, body = api.request(
		t, http.MethodDelete, "/komunitas/"+communityID, nil, true,
	)
	if status != http.StatusOK {
		t.Fatalf("delete community: expected HTTP 200, got %d: %s", status, body)
	}
	communityDeleted = true
	t.Log("Delete community: PASS")

	// Confirm the deleted community can no longer be retrieved.
	status, body = api.request(
		t, http.MethodGet, "/komunitas/"+communityID, nil, true,
	)
	if status != http.StatusNotFound {
		t.Errorf(
			"get deleted community: expected HTTP 404, got %d: %s",
			status, body,
		)
	} else {
		t.Log("Deleted community is no longer accessible: PASS")
	}
}

func (api *apiClient) request(
	t *testing.T,
	method string,
	path string,
	payload any,
	authenticated bool,
) (int, string) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode request body: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	return api.doRequest(t, method, path, body, authenticated)
}

func (api *apiClient) requestRaw(
	t *testing.T,
	method string,
	path string,
	payload string,
	authenticated bool,
) (int, string) {
	t.Helper()
	return api.doRequest(t, method, path, strings.NewReader(payload), authenticated)
}

func (api *apiClient) doRequest(
	t *testing.T,
	method string,
	path string,
	body io.Reader,
	authenticated bool,
) (int, string) {
	t.Helper()
	// Keep integration-test login requests within the API rate limit.
	if method == http.MethodPost && path == "/auth/login" {
		apiLoginMu.Lock()

		interval := 6250 * time.Millisecond
		if lastAPILogin.IsZero() {
			// Allow the setup script's login attempts to replenish.
			interval = 12500 * time.Millisecond
		}

		if wait := interval - time.Since(lastAPILogin); !lastAPILogin.IsZero() && wait > 0 {
			time.Sleep(wait)
		} else if lastAPILogin.IsZero() {
			time.Sleep(interval)
		}

		lastAPILogin = time.Now()
		apiLoginMu.Unlock()
	}

	req, err := http.NewRequest(method, api.baseURL+path, body)
	if err != nil {
		t.Fatalf("create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+api.token)
	}

	resp, err := api.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	return resp.StatusCode, string(responseBody)
}

// findString searches nested JSON objects for a matching string field.
func findString(raw string, keys ...string) string {
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return ""
	}
	return findStringValue(value, keys...)
}

func findStringValue(value any, keys ...string) string {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range keys {
			if found, ok := v[key]; ok {
				if s, ok := found.(string); ok && s != "" {
					return s
				}
			}
		}
		for _, child := range v {
			if result := findStringValue(child, keys...); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range v {
			if result := findStringValue(child, keys...); result != "" {
				return result
			}
		}
	}
	return ""
}

// findID searches nested JSON for common community ID field names.
func findID(raw string) string {
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return ""
	}
	return findIDValue(value)
}

func findIDValue(value any) string {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range []string{"id_komunitas", "komunitas_id", "id"} {
			if found, ok := v[key]; ok {
				switch n := found.(type) {
				case float64:
					if n > 0 {
						return strconv.FormatInt(int64(n), 10)
					}
				case string:
					if parsed, err := strconv.ParseUint(n, 10, 32); err == nil && parsed > 0 {
						return n
					}
				}
			}
		}
		for _, child := range v {
			if result := findIDValue(child); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range v {
			if result := findIDValue(child); result != "" {
				return result
			}
		}
	}
	return ""
}

func mustUint(t *testing.T, value string) uint {
	t.Helper()
	n, err := strconv.ParseUint(value, 10, 32)
	if err != nil || n == 0 {
		t.Fatalf("invalid user ID %q", value)
	}
	return uint(n)
}
