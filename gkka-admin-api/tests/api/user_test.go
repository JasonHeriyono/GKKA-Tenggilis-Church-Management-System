package api_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestUserAPIIntegration(t *testing.T) {
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

	// Get the admin's ID for self-protection tests.
	status, body := admin.request(t, http.MethodGet, "/auth/me", nil, true)
	if status != http.StatusOK {
		t.Fatalf("get admin profile: expected 200, got %d: %s", status, body)
	}
	adminUserID := findID(body)
	if adminUserID == "" {
		t.Fatalf("could not find admin ID in profile response: %s", body)
	}

	// List users.
	status, body = admin.request(t, http.MethodGet, "/users", nil, true)
	if status != http.StatusOK {
		t.Fatalf("list users: expected 200, got %d: %s", status, body)
	}
	t.Log("List users: PASS")

	// Invalid IDs should be rejected.
	status, body = admin.request(t, http.MethodGet, "/users/abc", nil, true)
	if status != http.StatusBadRequest {
		t.Errorf("invalid user ID: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Invalid user ID rejected: PASS")
	}

	// A valid but nonexistent ID should return 404.
	status, body = admin.request(t, http.MethodGet, "/users/4294967295", nil, true)
	if status != http.StatusNotFound {
		t.Errorf("nonexistent user: expected 404, got %d: %s", status, body)
	} else {
		t.Log("Nonexistent user rejected: PASS")
	}

	// Invalid create request should be rejected.
	status, body = admin.request(t, http.MethodPost, "/users",
		map[string]any{}, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("empty create request: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Missing user fields rejected: PASS")
	}

	// Create a unique test user.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	kode := "API" + suffix[len(suffix)-12:]
	email := "api.user." + suffix + "@example.com"
	initialPassword := "InitialPass2026!"
	newPassword := "ChangedPass2026!"

	status, body = admin.request(t, http.MethodPost, "/users",
		map[string]string{
			"kode_jemaat": kode,
			"nama":        "API Test User " + suffix,
			"email":       email,
			"role":        "Jemaat",
			"password":    initialPassword,
		}, true)
	if status != http.StatusCreated {
		t.Fatalf("create user: expected 201, got %d: %s", status, body)
	}

	userID := findID(body)
	if userID == "" {
		t.Fatalf("could not find created user ID: %s", body)
	}
	t.Log("Create user: PASS")

	// Duplicate user data should return 409.
	status, body = admin.request(t, http.MethodPost, "/users",
		map[string]string{
			"kode_jemaat": kode,
			"nama":        "Duplicate API Test User",
			"role":        "Jemaat",
		}, true)
	if status != http.StatusConflict {
		t.Errorf("duplicate user: expected 409, got %d: %s", status, body)
	} else {
		t.Log("Duplicate user rejected: PASS")
	}

	// Retrieve the newly created user.
	status, body = admin.request(t, http.MethodGet, "/users/"+userID, nil, true)
	if status != http.StatusOK {
		t.Fatalf("get created user: expected 200, got %d: %s", status, body)
	}
	t.Log("Get user details: PASS")

	// Search for the new user.
	status, body = admin.request(t, http.MethodGet, "/users?q="+kode, nil, true)
	if status != http.StatusOK {
		t.Errorf("search users: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Search users: PASS")
	}

	// Filter users by role.
	status, body = admin.request(t, http.MethodGet, "/users?role=Jemaat", nil, true)
	if status != http.StatusOK {
		t.Errorf("filter users by role: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Filter users by role: PASS")
	}

	// Invalid role should be rejected.
	status, body = admin.request(t, http.MethodPatch, "/users/"+userID+"/role",
		map[string]string{"role": "InvalidRole"}, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("invalid role: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Invalid role rejected: PASS")
	}

	// Change the test user's role.
	status, body = admin.request(t, http.MethodPatch, "/users/"+userID+"/role",
		map[string]string{"role": "Majelis"}, true)
	if status != http.StatusOK {
		t.Fatalf("change role: expected 200, got %d: %s", status, body)
	}
	t.Log("Change user role: PASS")

	// An admin must not be able to change their own role.
	status, body = admin.request(t, http.MethodPatch, "/users/"+adminUserID+"/role",
		map[string]string{"role": "Jemaat"}, true)
	if status != http.StatusBadRequest {
		t.Errorf("change own role: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Self-role change rejected: PASS")
	}

	// An admin must not be able to deactivate their own account.
	status, body = admin.request(t, http.MethodPatch, "/users/"+adminUserID+"/status",
		map[string]bool{"is_aktif": false}, true)
	if status != http.StatusBadRequest {
		t.Errorf("deactivate own account: expected 400, got %d: %s", status, body)
	} else {
		t.Log("Self-deactivation rejected: PASS")
	}

	// Deactivate the test user.
	status, body = admin.request(t, http.MethodPatch, "/users/"+userID+"/status",
		map[string]bool{"is_aktif": false}, true)
	if status != http.StatusOK {
		t.Fatalf("deactivate user: expected 200, got %d: %s", status, body)
	}
	t.Log("Deactivate user: PASS")

	// Inactive users must not be able to log in.
	status, body = admin.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": email,
			"password":   initialPassword,
		}, false)
	if status != http.StatusForbidden {
		t.Errorf("login with inactive account: expected 403, got %d: %s", status, body)
	} else {
		t.Log("Inactive account login rejected: PASS")
	}

	// Reactivate the test user.
	status, body = admin.request(t, http.MethodPatch, "/users/"+userID+"/status",
		map[string]bool{"is_aktif": true}, true)
	if status != http.StatusOK {
		t.Fatalf("reactivate user: expected 200, got %d: %s", status, body)
	}
	t.Log("Reactivate user: PASS")

	// Log in to obtain a refresh token before changing the password.
	status, body = admin.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": email,
			"password":   initialPassword,
		}, false)
	if status != http.StatusOK {
		t.Fatalf("test user login: expected 200, got %d: %s", status, body)
	}
	oldRefreshToken := findString(body, "refresh_token")
	if oldRefreshToken == "" {
		t.Fatalf("login response has no refresh token: %s", body)
	}
	t.Log("Test user login: PASS")

	// Passwords shorter than 8 characters should be rejected.
	status, body = admin.request(t, http.MethodPut, "/users/"+userID+"/password",
		map[string]string{"password": "short"}, true)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("short password: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Short password rejected: PASS")
	}

	// Change the test user's password.
	status, body = admin.request(t, http.MethodPut, "/users/"+userID+"/password",
		map[string]string{"password": newPassword}, true)
	if status != http.StatusOK {
		t.Fatalf("change password: expected 200, got %d: %s", status, body)
	}
	t.Log("Change user password: PASS")

	// Old password should no longer work.
	status, body = admin.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": email,
			"password":   initialPassword,
		}, false)
	if status != http.StatusUnauthorized {
		t.Errorf("old password: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Old password rejected: PASS")
	}

	// New password should work.
	status, body = admin.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": email,
			"password":   newPassword,
		}, false)
	if status != http.StatusOK {
		t.Fatalf("new password login: expected 200, got %d: %s", status, body)
	}
	t.Log("New password accepted: PASS")

	// Password changes should revoke the previous refresh token.
	status, body = admin.request(t, http.MethodPost, "/auth/refresh",
		map[string]string{
			"refresh_token": oldRefreshToken,
		}, false)
	if status != http.StatusUnauthorized {
		t.Errorf("refresh token after password change: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Previous refresh token revoked after password change: PASS")
	}
}
