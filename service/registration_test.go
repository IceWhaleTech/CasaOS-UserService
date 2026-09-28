package service

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/IceWhaleTech/CasaOS-UserService/service/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func registrationTestService(t *testing.T) *userService {
	t.Helper()
	registrationMu.Lock()
	registrationKey = ""
	registrationMu.Unlock()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "users.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.UserDBModel{}); err != nil {
		t.Fatal(err)
	}
	return &userService{db: db}
}

func TestRegistrationKeyIsUniqueAndInvalidated(t *testing.T) {
	u := registrationTestService(t)
	first, key, err := u.RegistrationStatus()
	if err != nil || first || key == "" {
		t.Fatalf("first status = (%v, %q, %v)", first, key, err)
	}
	initialized, again, err := u.RegistrationStatus()
	if err != nil || initialized || again != key {
		t.Fatalf("second status did not reuse the single key")
	}
	user, err := u.RegisterInitialUser(key, model.UserDBModel{Username: "owner", Role: "admin"})
	if err != nil || user.Id == 0 {
		t.Fatalf("initial registration failed: %v", err)
	}
	if _, err := u.RegisterInitialUser(key, model.UserDBModel{Username: "second", Role: "admin"}); !errors.Is(err, ErrRegistrationKeyInvalid) {
		t.Fatalf("reused key error = %v", err)
	}
	initialized, again, err = u.RegistrationStatus()
	if err != nil || !initialized || again != "" || u.GetUserCount() != 1 {
		t.Fatalf("post-registration status = (%v, %q, %v), count = %d", initialized, again, err, u.GetUserCount())
	}
}

func TestConcurrentInitialRegistrationCreatesOneUser(t *testing.T) {
	u := registrationTestService(t)
	_, key, err := u.RegistrationStatus()
	if err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := u.RegisterInitialUser(key, model.UserDBModel{Username: "owner", Role: "admin"}); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || u.GetUserCount() != 1 {
		t.Fatalf("registrations = %d, users = %d; want one", successes.Load(), u.GetUserCount())
	}
}
