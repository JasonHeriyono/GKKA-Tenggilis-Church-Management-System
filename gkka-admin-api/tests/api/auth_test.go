package api_test

import (
	"net/http"
	"os"
	"testing"
)

func TestAuthAPIIntegration(t *testing.T) {
	if os.Getenv("SIM_G_API_TESTS") != "1" {
		t.Skip("set SIM_G_API_TESTS=1 to run API integration tests")
	}

	adminID := os.Getenv("SIM_G_TEST_IDENTIFIER")
	adminPassword := os.Getenv("SIM_G_TEST_PASSWORD")
	if adminID == "" || adminPassword == "" {
		t.Fatal("set SIM_G_TEST_IDENTIFIER and SIM_G_TEST_PASSWORD")
	}

	// Login using the existing helper.
	api := loginAPI(t, adminID, adminPassword)
	t.Log("Login and access-token issuance: PASS")

	// Incorrect credentials should be rejected.
	status, body := api.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": adminID,
			"password":   "IncorrectPassword123!",
		}, false)
	if status != http.StatusUnauthorized {
		t.Errorf("invalid credentials: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Invalid credentials rejected: PASS")
	}

	// Missing required login fields should be rejected.
	status, body = api.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": adminID,
		}, false)
	if status != http.StatusUnprocessableEntity {
		t.Errorf("missing login password: expected 422, got %d: %s", status, body)
	} else {
		t.Log("Missing login field rejected: PASS")
	}

	// /auth/me requires a valid access token.
	status, body = api.request(t, http.MethodGet, "/auth/me", nil, false)
	if status != http.StatusUnauthorized {
		t.Errorf("unauthenticated /auth/me: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Unauthenticated profile request rejected: PASS")
	}

	validAccessToken := api.token
	api.token = "not-a-valid-access-token"

	status, body = api.request(t, http.MethodGet, "/auth/me", nil, true)
	if status != http.StatusUnauthorized {
		t.Errorf("invalid access token: expected 401, got %d: %s", status, body)
	} else {
		t.Log("Invalid access token rejected: PASS")
	}

	api.token = validAccessToken

	// A valid access token should retrieve the current user's profile.
	status, body = api.request(t, http.MethodGet, "/auth/me", nil, true)
	if status != http.StatusOK {
		t.Fatalf("get current user: expected 200, got %d: %s", status, body)
	}
	t.Log("Current-user profile: PASS")

	// Obtain a refresh token from a separate login.
	status, body = api.request(t, http.MethodPost, "/auth/login",
		map[string]string{
			"identifier": adminID,
			"password":   adminPassword,
		}, false)
	if status != http.StatusOK {
		t.Fatalf("second login for refresh test: expected 200, got %d: %s", status, body)
	}

	refreshToken := findString(body, "refresh_token")
	if refreshToken == "" {
		t.Fatalf("login response has no refresh token: %s", body)
	}
	t.Log("Refresh token issuance: PASS")

	// Refresh should rotate the refresh token and issue a new access token.
	status, body = api.request(t, http.MethodPost, "/auth/refresh",
		map[string]string{
			"refresh_token": refreshToken,
		}, false)
	if status != http.StatusOK {
		t.Fatalf("refresh token: expected 200, got %d: %s", status, body)
	}

	newAccessToken := findString(body, "access_token")
	newRefreshToken := findString(body, "refresh_token")
	if newAccessToken == "" || newRefreshToken == "" {
		t.Fatalf("refresh response is missing tokens: %s", body)
	}
	if newRefreshToken == refreshToken {
		t.Fatal("refresh token was not rotated")
	}
	t.Log("Refresh-token rotation: PASS")

	// The new access token should work.
	api.token = newAccessToken
	status, body = api.request(t, http.MethodGet, "/auth/me", nil, true)
	if status != http.StatusOK {
		t.Errorf("new access token: expected 200, got %d: %s", status, body)
	} else {
		t.Log("New access token works: PASS")
	}

	// Logout should revoke the newly issued refresh token.
	status, body = api.request(t, http.MethodPost, "/auth/logout",
		map[string]string{
			"refresh_token": newRefreshToken,
		}, false)
	if status != http.StatusOK {
		t.Errorf("logout: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Logout: PASS")
	}

	// Logout is documented as idempotent, including unknown tokens.
	status, body = api.request(t, http.MethodPost, "/auth/logout",
		map[string]string{
			"refresh_token": "unknown-refresh-token",
		}, false)
	if status != http.StatusOK {
		t.Errorf("logout with unknown token: expected 200, got %d: %s", status, body)
	} else {
		t.Log("Idempotent logout: PASS")
	}
}
