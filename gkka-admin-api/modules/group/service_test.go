package group

import (
	"context"
	"errors"
	"testing"

	"sim-g-backend/models"
)

type fakeGroupRepository struct {
	listFn              func(context.Context, ListQuery) ([]models.Komunitas, int64, error)
	findByIDFn          func(context.Context, uint) (*models.Komunitas, error)
	createFn            func(context.Context, *models.Komunitas) error
	updateFn            func(context.Context, *models.Komunitas) error
	deleteFn            func(context.Context, uint) error
	userExistsFn        func(context.Context, uint) (bool, error)
	listAnggotaFn       func(context.Context, uint) ([]models.UserKomunitas, error)
	addAnggotaFn        func(context.Context, *models.UserKomunitas) error
	setKetuaFn          func(context.Context, uint, uint, bool) error
	removeAnggotaFn     func(context.Context, uint, uint) error
	findDeletedByNamaFn func(context.Context, string) (*models.Komunitas, error)
	restoreFn           func(context.Context, *models.Komunitas) error
}

func (f *fakeGroupRepository) List(ctx context.Context, q ListQuery) ([]models.Komunitas, int64, error) {
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	return nil, 0, nil
}

func (f *fakeGroupRepository) FindByID(ctx context.Context, id uint) (*models.Komunitas, error) {
	if f.findByIDFn != nil {
		return f.findByIDFn(ctx, id)
	}
	return nil, ErrNotFound
}

func (f *fakeGroupRepository) FindDeletedByNama(ctx context.Context, nama string) (*models.Komunitas, error) {
	if f.findDeletedByNamaFn != nil {
		return f.findDeletedByNamaFn(ctx, nama)
	}
	return nil, ErrNotFound
}

func (f *fakeGroupRepository) Restore(ctx context.Context, k *models.Komunitas) error {
	if f.restoreFn != nil {
		return f.restoreFn(ctx, k)
	}
	return nil
}

func (f *fakeGroupRepository) Create(ctx context.Context, k *models.Komunitas) error {
	if f.createFn != nil {
		return f.createFn(ctx, k)
	}
	return nil
}

func (f *fakeGroupRepository) Update(ctx context.Context, k *models.Komunitas) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, k)
	}
	return nil
}

func (f *fakeGroupRepository) Delete(ctx context.Context, id uint) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, id)
	}
	return nil
}

func (f *fakeGroupRepository) UserExists(ctx context.Context, id uint) (bool, error) {
	if f.userExistsFn != nil {
		return f.userExistsFn(ctx, id)
	}
	return false, nil
}

func (f *fakeGroupRepository) ListAnggota(ctx context.Context, id uint) ([]models.UserKomunitas, error) {
	if f.listAnggotaFn != nil {
		return f.listAnggotaFn(ctx, id)
	}
	return nil, nil
}

func (f *fakeGroupRepository) AddAnggota(ctx context.Context, a *models.UserKomunitas) error {
	if f.addAnggotaFn != nil {
		return f.addAnggotaFn(ctx, a)
	}
	return nil
}

func (f *fakeGroupRepository) SetKetua(ctx context.Context, groupID, userID uint, ketua bool) error {
	if f.setKetuaFn != nil {
		return f.setKetuaFn(ctx, groupID, userID, ketua)
	}
	return nil
}

func (f *fakeGroupRepository) RemoveAnggota(ctx context.Context, groupID, userID uint) error {
	if f.removeAnggotaFn != nil {
		return f.removeAnggotaFn(ctx, groupID, userID)
	}
	return nil
}

func TestGroupServiceList(t *testing.T) {
	repo := &fakeGroupRepository{
		listFn: func(_ context.Context, q ListQuery) ([]models.Komunitas, int64, error) {
			if q.Q != "Pemuda" {
				t.Errorf("expected trimmed search query %q, got %q", "Pemuda", q.Q)
			}
			return []models.Komunitas{
				{ID: 1, Nama: "Komunitas Pemuda"},
			}, 1, nil
		},
	}

	result, err := NewService(repo).List(context.Background(), ListQuery{
		Q: "  Pemuda  ",
	})
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 community, got %d", len(result.Items))
	}
	if result.Items[0].Nama != "Komunitas Pemuda" {
		t.Errorf("unexpected community name: %q", result.Items[0].Nama)
	}
}

func TestGroupServiceListRepositoryError(t *testing.T) {
	repo := &fakeGroupRepository{
		listFn: func(context.Context, ListQuery) ([]models.Komunitas, int64, error) {
			return nil, 0, errors.New("database failure")
		},
	}

	_, err := NewService(repo).List(context.Background(), ListQuery{})
	if err == nil {
		t.Fatal("expected an error when repository List fails")
	}
}

func TestGroupServiceGet(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(_ context.Context, id uint) (*models.Komunitas, error) {
			if id != 7 {
				t.Errorf("expected community ID 7, got %d", id)
			}
			return &models.Komunitas{
				ID:   7,
				Nama: "Komunitas Pemuda",
			}, nil
		},
		listAnggotaFn: func(_ context.Context, id uint) ([]models.UserKomunitas, error) {
			if id != 7 {
				t.Errorf("expected member lookup for community 7, got %d", id)
			}
			return nil, nil
		},
	}

	result, err := NewService(repo).Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if result.ID != 7 || result.Nama != "Komunitas Pemuda" {
		t.Errorf("unexpected community response: %+v", result.Response)
	}
	if len(result.Anggota) != 0 {
		t.Errorf("expected no members, got %d", len(result.Anggota))
	}
}

func TestGroupServiceGetNotFound(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return nil, ErrNotFound
		},
	}

	_, err := NewService(repo).Get(context.Background(), 99)
	if err == nil {
		t.Fatal("expected an error for a nonexistent community")
	}
}

func TestGroupServiceGetMemberListError(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: 1, Nama: "Test"}, nil
		},
		listAnggotaFn: func(context.Context, uint) ([]models.UserKomunitas, error) {
			return nil, errors.New("database failure")
		},
	}

	_, err := NewService(repo).Get(context.Background(), 1)
	if err == nil {
		t.Fatal("expected an error when listing community members fails")
	}
}

func TestGroupServiceCreate(t *testing.T) {
	repo := &fakeGroupRepository{
		createFn: func(_ context.Context, k *models.Komunitas) error {
			if k.Nama != "Komunitas Pemuda" {
				t.Errorf("expected trimmed name, got %q", k.Nama)
			}
			if k.Deskripsi == nil || *k.Deskripsi != "Kegiatan pemuda" {
				t.Errorf("unexpected description: %v", k.Deskripsi)
			}
			k.ID = 10 // simulate database-generated ID
			return nil
		},
	}

	result, err := NewService(repo).Create(context.Background(), SaveRequest{
		Nama:      "  Komunitas Pemuda  ",
		Deskripsi: "  Kegiatan pemuda  ",
	})
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if result.ID != 10 || result.Nama != "Komunitas Pemuda" {
		t.Errorf("unexpected create response: %+v", result)
	}
}

func TestGroupServiceCreateEmptyDescription(t *testing.T) {
	repo := &fakeGroupRepository{
		createFn: func(_ context.Context, k *models.Komunitas) error {
			if k.Deskripsi != nil {
				t.Errorf("expected empty description to become nil, got %v", *k.Deskripsi)
			}
			k.ID = 11
			return nil
		},
	}

	_, err := NewService(repo).Create(context.Background(), SaveRequest{
		Nama:      "Komunitas Baru",
		Deskripsi: "   ",
	})
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
}

func TestGroupServiceCreateDuplicate(t *testing.T) {
	repo := &fakeGroupRepository{
		createFn: func(context.Context, *models.Komunitas) error {
			return ErrDuplicate
		},
	}

	_, err := NewService(repo).Create(context.Background(), SaveRequest{
		Nama: "Komunitas Pemuda",
	})
	if err == nil {
		t.Fatal("expected an error for a duplicate community name")
	}
}

func TestGroupServiceUpdate(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: 3, Nama: "Nama Lama"}, nil
		},
		updateFn: func(_ context.Context, k *models.Komunitas) error {
			if k.Nama != "Nama Baru" {
				t.Errorf("expected updated name, got %q", k.Nama)
			}
			if k.Deskripsi != nil {
				t.Errorf("expected blank description to become nil")
			}
			return nil
		},
	}

	result, err := NewService(repo).Update(context.Background(), 3, SaveRequest{
		Nama:      " Nama Baru ",
		Deskripsi: " ",
	})
	if err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}
	if result.Nama != "Nama Baru" {
		t.Errorf("unexpected updated name: %q", result.Nama)
	}
}

func TestGroupServiceUpdateNotFound(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return nil, ErrNotFound
		},
	}

	_, err := NewService(repo).Update(context.Background(), 99, SaveRequest{
		Nama: "Nama Baru",
	})
	if err == nil {
		t.Fatal("expected an error when updating a nonexistent community")
	}
}

func TestGroupServiceDelete(t *testing.T) {
	called := false
	repo := &fakeGroupRepository{
		deleteFn: func(_ context.Context, id uint) error {
			called = true
			if id != 5 {
				t.Errorf("expected community ID 5, got %d", id)
			}
			return nil
		},
	}

	err := NewService(repo).Delete(context.Background(), 5)
	if err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected repository Delete to be called")
	}
}

func TestGroupServiceDeleteNotFound(t *testing.T) {
	repo := &fakeGroupRepository{
		deleteFn: func(context.Context, uint) error {
			return ErrNotFound
		},
	}

	err := NewService(repo).Delete(context.Background(), 99)
	if err == nil {
		t.Fatal("expected an error when deleting a nonexistent community")
	}
}

func TestGroupServiceAddAnggotaCommunityNotFound(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return nil, ErrNotFound
		},
	}

	_, err := NewService(repo).AddAnggota(context.Background(), 99, AnggotaRequest{
		UserID: 1,
	})
	if err == nil {
		t.Fatal("expected an error when the community does not exist")
	}
}

func TestGroupServiceAddAnggotaUserNotFound(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: 1, Nama: "Test"}, nil
		},
		userExistsFn: func(context.Context, uint) (bool, error) {
			return false, nil
		},
	}

	_, err := NewService(repo).AddAnggota(context.Background(), 1, AnggotaRequest{
		UserID: 99,
	})
	if err == nil {
		t.Fatal("expected an error when the user does not exist")
	}
}

func TestGroupServiceAddAnggotaDuplicate(t *testing.T) {
	repo := &fakeGroupRepository{
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: 1, Nama: "Test"}, nil
		},
		userExistsFn: func(context.Context, uint) (bool, error) {
			return true, nil
		},
		addAnggotaFn: func(context.Context, *models.UserKomunitas) error {
			return ErrDuplicate
		},
	}

	_, err := NewService(repo).AddAnggota(context.Background(), 1, AnggotaRequest{
		UserID: 2,
	})
	if err == nil {
		t.Fatal("expected an error for duplicate membership")
	}
}

func TestGroupServiceSetKetua(t *testing.T) {
	called := false
	repo := &fakeGroupRepository{
		setKetuaFn: func(_ context.Context, groupID, userID uint, isKetua bool) error {
			called = true
			if groupID != 1 || userID != 2 || !isKetua {
				t.Errorf("unexpected SetKetua arguments: group=%d user=%d ketua=%t", groupID, userID, isKetua)
			}
			return nil
		},
		findByIDFn: func(context.Context, uint) (*models.Komunitas, error) {
			return &models.Komunitas{ID: 1, Nama: "Test"}, nil
		},
		listAnggotaFn: func(context.Context, uint) ([]models.UserKomunitas, error) {
			return nil, nil
		},
	}

	_, err := NewService(repo).SetKetua(context.Background(), 1, 2, true)
	if err != nil {
		t.Fatalf("SetKetua returned unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected repository SetKetua to be called")
	}
}

func TestGroupServiceSetKetuaNotMember(t *testing.T) {
	repo := &fakeGroupRepository{
		setKetuaFn: func(context.Context, uint, uint, bool) error {
			return ErrNotFound
		},
	}

	_, err := NewService(repo).SetKetua(context.Background(), 1, 2, true)
	if err == nil {
		t.Fatal("expected an error when the user is not a community member")
	}
}

func TestGroupServiceRemoveAnggota(t *testing.T) {
	repo := &fakeGroupRepository{
		removeAnggotaFn: func(_ context.Context, groupID, userID uint) error {
			if groupID != 1 || userID != 2 {
				t.Errorf("unexpected RemoveAnggota arguments: group=%d user=%d", groupID, userID)
			}
			return nil
		},
	}

	if err := NewService(repo).RemoveAnggota(context.Background(), 1, 2); err != nil {
		t.Fatalf("RemoveAnggota returned unexpected error: %v", err)
	}
}

func TestGroupServiceRemoveAnggotaNotMember(t *testing.T) {
	repo := &fakeGroupRepository{
		removeAnggotaFn: func(context.Context, uint, uint) error {
			return ErrNotFound
		},
	}

	err := NewService(repo).RemoveAnggota(context.Background(), 1, 2)
	if err == nil {
		t.Fatal("expected an error when removing a nonexistent membership")
	}
}
