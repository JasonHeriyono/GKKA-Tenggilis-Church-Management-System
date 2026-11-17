package user

import (
	"context"
	"net/http"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"sim-g-backend/helpers/apperror"
	"sim-g-backend/models"
)

type fakeRepo struct {
	users      map[uint]*models.User
	roles      map[string]*models.Role
	created    *models.User
	dupOnNext  bool
	lastRoleID uint
	revoked    []uint
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users: map[uint]*models.User{
			1: {ID: 1, KodeJemaat: "ADMIN001", Nama: "Admin", RoleID: 3, Role: models.Role{ID: 3, Nama: models.RoleSuperAdmin}, IsAktif: true},
			2: {ID: 2, KodeJemaat: "J-002", Nama: "Sari", RoleID: 1, Role: models.Role{ID: 1, Nama: models.RoleJemaat}, IsAktif: true},
		},
		roles: map[string]*models.Role{
			models.RoleJemaat:     {ID: 1, Nama: models.RoleJemaat},
			models.RoleMajelis:    {ID: 2, Nama: models.RoleMajelis},
			models.RoleSuperAdmin: {ID: 3, Nama: models.RoleSuperAdmin},
		},
	}
}

func (r *fakeRepo) List(context.Context, ListQuery) ([]models.User, int64, error) {
	var out []models.User
	for _, u := range r.users {
		out = append(out, *u)
	}
	return out, int64(len(out)), nil
}

func (r *fakeRepo) FindByID(_ context.Context, id uint) (*models.User, error) {
	if u, ok := r.users[id]; ok {
		return u, nil
	}
	return nil, ErrNotFound
}

func (r *fakeRepo) FindRoleByName(_ context.Context, name string) (*models.Role, error) {
	if role, ok := r.roles[name]; ok {
		return role, nil
	}
	return nil, ErrNotFound
}

func (r *fakeRepo) Create(_ context.Context, u *models.User) error {
	if r.dupOnNext {
		return ErrDuplicate
	}
	u.ID = 99
	r.created = u
	r.users[u.ID] = u
	return nil
}

func (r *fakeRepo) UpdateRole(_ context.Context, id, roleID uint) error {
	u, ok := r.users[id]
	if !ok {
		return ErrNotFound
	}
	r.lastRoleID = roleID
	for _, role := range r.roles {
		if role.ID == roleID {
			u.RoleID, u.Role = roleID, *role
		}
	}
	return nil
}

func (r *fakeRepo) SetActive(_ context.Context, id uint, aktif bool) error {
	u, ok := r.users[id]
	if !ok {
		return ErrNotFound
	}
	u.IsAktif = aktif
	if !aktif {
		r.revoked = append(r.revoked, id)
	}
	return nil
}

func (r *fakeRepo) SetPassword(_ context.Context, id uint, hash string) error {
	u, ok := r.users[id]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = &hash
	r.revoked = append(r.revoked, id)
	return nil
}

func status(err error) int {
	if err == nil {
		return 0
	}
	return apperror.From(err).Status
}

func TestCreate_TanpaPassword_AkunBelumAktif(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)
	res, err := svc.Create(context.Background(), CreateRequest{
		KodeJemaat: " J-100 ", Nama: "Dewi", Email: "", NoTelepon: "0812", Role: models.RoleJemaat, TanggalLahir: "2000-02-29",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AccountActivated || res.Role != models.RoleJemaat || res.KodeJemaat != "J-100" {
		t.Errorf("res = %+v", res)
	}
	if res.Email != nil {
		t.Error("email kosong harus disimpan sebagai NULL (kolom unik)")
	}
	if res.TanggalLahir == nil || *res.TanggalLahir != "2000-02-29" {
		t.Errorf("tanggal lahir = %v", res.TanggalLahir)
	}
	if repo.created.PasswordHash != nil {
		t.Error("password_hash harus NULL")
	}
}

func TestCreate_DenganPassword_DisimpanSebagaiHash(t *testing.T) {
	repo := newFakeRepo()
	res, err := NewService(repo).Create(context.Background(), CreateRequest{
		KodeJemaat: "J-101", Nama: "Eko", Role: models.RoleMajelis, Password: "PasswordKuat1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.AccountActivated {
		t.Error("akun dengan password harus aktif")
	}
	h := repo.created.PasswordHash
	if h == nil || *h == "PasswordKuat1" {
		t.Fatal("password tidak boleh disimpan mentah")
	}
	if bcrypt.CompareHashAndPassword([]byte(*h), []byte("PasswordKuat1")) != nil {
		t.Error("hash tidak cocok dengan password")
	}
}

func TestCreate_RoleTidakDikenal(t *testing.T) {
	_, err := NewService(newFakeRepo()).Create(context.Background(), CreateRequest{KodeJemaat: "J-1", Nama: "X", Role: "Usher"})
	if status(err) != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", status(err))
	}
}
