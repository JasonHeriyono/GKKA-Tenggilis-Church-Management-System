package group

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"sim-g-backend/models"
	"context"
)

func setupGroupHandler(t *testing.T, repo *fakeGroupRepository) (*gin.Engine, *Handler) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := NewHandler(NewService(repo))
	router := gin.New()

	RegisterRoutes(router.Group("/gkki/api"), handler)

	return router, handler
}

func performGroupRequest(
	router http.Handler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	return recorder
}

func TestGroupHandlerList(t *testing.T) {
	repo := &fakeGroupRepository{
		listFn: func(_ context.Context, filter ListQuery) ([]models.Komunitas, int64, error) {
			return []models.Komunitas{
				{ID: 1, Nama: "Pemuda"},
			}, 1, nil
		},
	}

	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router, http.MethodGet, "/gkki/api/komunitas", "",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerGetInvalidID(t *testing.T) {
	repo := &fakeGroupRepository{}
	router, _ := setupGroupHandler(t, repo)

	tests := []struct {
		name string
		id   string
	}{
		{name: "zero", id: "0"},
		{name: "text", id: "abc"},
		{name: "negative", id: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performGroupRequest(
				router,
				http.MethodGet,
				"/gkki/api/komunitas/"+tt.id,
				"",
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestGroupHandlerCreateInvalidJSON(t *testing.T) {
	repo := &fakeGroupRepository{}
	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodPost,
		"/gkki/api/komunitas",
		`{"nama":`,
	)

	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf("expected a 4xx status, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerCreateMissingNama(t *testing.T) {
	repo := &fakeGroupRepository{}
	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodPost,
		"/gkki/api/komunitas",
		`{"deskripsi":"Komunitas pengujian"}`,
	)

	if recorder.Code < 400 || recorder.Code >= 500 {
		t.Fatalf("expected a 4xx status, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerGetServiceError(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(_ context.Context, _ uint) (*models.Komunitas, error) {
			return nil, errors.New("database unavailable")
		},
	}

	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodGet,
		"/gkki/api/komunitas/1",
		"",
	)

	if recorder.Code < 400 {
		t.Fatalf("expected an error status, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerInvalidIDsAcrossEndpoints(t *testing.T) {
	repo := &fakeGroupRepository{}
	router, _ := setupGroupHandler(t, repo)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "update with invalid community ID",
			method: http.MethodPut,
			path:   "/gkki/api/komunitas/abc",
		},
		{
			name:   "delete with invalid community ID",
			method: http.MethodDelete,
			path:   "/gkki/api/komunitas/abc",
		},
		{
			name:   "add member with invalid community ID",
			method: http.MethodPost,
			path:   "/gkki/api/komunitas/abc/anggota",
		},
		{
			name:   "set leader with invalid community ID",
			method: http.MethodPatch,
			path:   "/gkki/api/komunitas/abc/anggota/1",
		},
		{
			name:   "set leader with invalid user ID",
			method: http.MethodPatch,
			path:   "/gkki/api/komunitas/1/anggota/abc",
		},
		{
			name:   "remove member with invalid community ID",
			method: http.MethodDelete,
			path:   "/gkki/api/komunitas/abc/anggota/1",
		},
		{
			name:   "remove member with invalid user ID",
			method: http.MethodDelete,
			path:   "/gkki/api/komunitas/1/anggota/abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performGroupRequest(
				router,
				tt.method,
				tt.path,
				`{}`,
			)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d: %s",
					http.StatusBadRequest,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestGroupHandlerInvalidMemberRequests(t *testing.T) {
	repo := &fakeGroupRepository{}
	router, _ := setupGroupHandler(t, repo)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "add member without user ID",
			method: http.MethodPost,
			path:   "/gkki/api/komunitas/1/anggota",
			body:   `{"is_ketua":false}`,
		},
		{
			name:   "add member with malformed JSON",
			method: http.MethodPost,
			path:   "/gkki/api/komunitas/1/anggota",
			body:   `{"id_user":`,
		},
		{
			name:   "set leader without is_ketua",
			method: http.MethodPatch,
			path:   "/gkki/api/komunitas/1/anggota/2",
			body:   `{}`,
		},
		{
			name:   "set leader with malformed JSON",
			method: http.MethodPatch,
			path:   "/gkki/api/komunitas/1/anggota/2",
			body:   `{"is_ketua":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performGroupRequest(
				router,
				tt.method,
				tt.path,
				tt.body,
			)

			if recorder.Code < 400 || recorder.Code >= 500 {
				t.Fatalf(
					"expected a 4xx status, got %d: %s",
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestGroupHandlerCreateSuccess(t *testing.T) {
	repo := &fakeGroupRepository{
		createFn: func(_ context.Context, community *models.Komunitas) error {
			community.ID = 10
			return nil
		},
	}

	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodPost,
		"/gkki/api/komunitas",
		`{"nama":"Komunitas Uji","deskripsi":"Automated test"}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusCreated, recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerUpdateSuccess(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(_ context.Context, id uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: id, Nama: "Komunitas Lama"}, nil
		},
		updateFn: func(_ context.Context, community *models.Komunitas) error {
			if community.Nama != "Komunitas Baru" {
				t.Errorf("expected updated name, got %q", community.Nama)
			}
			return nil
		},
	}

	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodPut,
		"/gkki/api/komunitas/1",
		`{"nama":"Komunitas Baru","deskripsi":"Updated by test"}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerDeleteSuccess(t *testing.T) {
	deleted := false

	repo := &fakeGroupRepository{
		findByIDFn: func(_ context.Context, id uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: id, Nama: "Komunitas Uji"}, nil
		},
		deleteFn: func(_ context.Context, id uint) error {
			if id != 1 {
				t.Errorf("expected community ID 1, got %d", id)
			}
			deleted = true
			return nil
		},
	}

	router, _ := setupGroupHandler(t, repo)

	recorder := performGroupRequest(
		router,
		http.MethodDelete,
		"/gkki/api/komunitas/1",
		"",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusOK, recorder.Code, recorder.Body.String())
	}

	if !deleted {
		t.Fatal("expected repository delete function to be called")
	}
}

func TestGroupHandlerAddMemberSuccess(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(_ context.Context, id uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: id, Nama: "Pemuda"}, nil
		},
		userExistsFn: func(_ context.Context, id uint) (bool, error) {
			return id == 2, nil
		},
		addAnggotaFn: func(_ context.Context, anggota *models.UserKomunitas) error {
			if anggota.KomunitasID != 1 || anggota.UserID != 2 {
				t.Errorf("unexpected member IDs: community=%d user=%d",
					anggota.KomunitasID, anggota.UserID)
			}
			return nil
		},
		listAnggotaFn: func(_ context.Context, _ uint) ([]models.UserKomunitas, error) {
			return []models.UserKomunitas{
				{UserID: 2, User: models.User{KodeJemaat: "J002", Nama: "Test Member"}},
			}, nil
		},
	}

	router, _ := setupGroupHandler(t, repo)
	recorder := performGroupRequest(
		router, http.MethodPost, "/gkki/api/komunitas/1/anggota",
		`{"id_user":2,"is_ketua":false}`,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGroupHandlerSetLeaderSuccess(t *testing.T) {
	leaderUpdated := false

	repo := &fakeGroupRepository{
		setKetuaFn: func(_ context.Context, communityID, userID uint, isLeader bool) error {
			if communityID != 1 || userID != 2 || !isLeader {
				t.Errorf("unexpected leader update: community=%d user=%d isLeader=%t",
					communityID, userID, isLeader)
			}
			leaderUpdated = true
			return nil
		},
		findByIDFn: func(_ context.Context, id uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: id, Nama: "Pemuda"}, nil
		},
		listAnggotaFn: func(_ context.Context, _ uint) ([]models.UserKomunitas, error) {
			return []models.UserKomunitas{
				{UserID: 2, IsKetua: true, User: models.User{KodeJemaat: "J002", Nama: "Test Member"}},
			}, nil
		},
	}

	router, _ := setupGroupHandler(t, repo)
	recorder := performGroupRequest(
		router, http.MethodPatch, "/gkki/api/komunitas/1/anggota/2",
		`{"is_ketua":true}`,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !leaderUpdated {
		t.Fatal("expected SetKetua repository method to be called")
	}
}

func TestGroupHandlerRemoveMemberSuccess(t *testing.T) {
	memberRemoved := false

	repo := &fakeGroupRepository{
		removeAnggotaFn: func(_ context.Context, communityID, userID uint) error {
			if communityID != 1 || userID != 2 {
				t.Errorf("unexpected IDs: community=%d user=%d", communityID, userID)
			}
			memberRemoved = true
			return nil
		},
	}

	router, _ := setupGroupHandler(t, repo)
	recorder := performGroupRequest(
		router, http.MethodDelete, "/gkki/api/komunitas/1/anggota/2", "",
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !memberRemoved {
		t.Fatal("expected RemoveAnggota repository method to be called")
	}
}