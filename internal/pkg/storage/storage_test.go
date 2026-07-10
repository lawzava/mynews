//nolint:gosec // allow weak generators on tests
package storage_test

import (
	"math/rand"
	"mynews/internal/pkg/storage"
	"strconv"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	t.Parallel()

	store := storage.New()

	rGen := rand.New(rand.NewSource(time.Now().Unix()))

	for range 1000 {
		randomKey := strconv.Itoa(rGen.Int())

		exists, err := store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if exists {
			t.Error("key should not exist")
		}

		err = store.PutKey("", randomKey)
		if err != nil {
			t.Error(err)
		}

		exists, err = store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if !exists {
			t.Error("key should exist")
		}
	}
}

//nolint:funlen,cyclop // allow for statements excession
func TestStorageCleanup(t *testing.T) {
	t.Parallel()

	store := storage.New()

	rGen := rand.New(rand.NewSource(time.Now().Unix()))

	for range 1000 {
		randomKey := strconv.Itoa(rGen.Int())

		cleanupBefore := time.Now()

		exists, err := store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if exists {
			t.Error("key should not exist")
		}

		err = store.PutKey("", randomKey)
		if err != nil {
			t.Error(err)
		}

		exists, err = store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if !exists {
			t.Error("key should exist")
		}

		store.CleanupAllBefore(cleanupBefore)

		exists, err = store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if !exists {
			t.Error("key should exist")
		}

		cleanupBefore = time.Now()

		err = store.PutKey("", randomKey)
		if err != nil {
			t.Error(err)
		}

		store.CleanupAllBefore(cleanupBefore)

		exists, err = store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if !exists {
			t.Error("key should exist")
		}
	}
}

// TestStorageCleanupRemovesStaleKeys covers the cutoff boundary: a key last seen
// before the cutoff is pruned while a key stored after the cutoff survives, even
// when the two keys live in different app buckets of the same cleanup pass.
func TestStorageCleanupRemovesStaleKeys(t *testing.T) {
	t.Parallel()

	store := storage.New()

	err := store.PutKey("app-a", "stale")
	if err != nil {
		t.Fatalf("putting stale key: %v", err)
	}

	time.Sleep(time.Millisecond)

	cutoff := time.Now()

	err = store.PutKey("app-b", "fresh")
	if err != nil {
		t.Fatalf("putting fresh key: %v", err)
	}

	store.CleanupAllBefore(cutoff)

	staleExists, err := store.KeyExists("app-a", "stale")
	if err != nil {
		t.Fatalf("checking stale key: %v", err)
	}

	if staleExists {
		t.Error("stale key should have been pruned")
	}

	freshExists, err := store.KeyExists("app-b", "fresh")
	if err != nil {
		t.Fatalf("checking fresh key: %v", err)
	}

	if !freshExists {
		t.Error("a key seen after the cutoff must survive cleanup")
	}
}

func TestStorageCleanupAllBefore(t *testing.T) {
	t.Parallel()

	store := storage.New()

	err := store.PutKey("app-a", "old")
	if err != nil {
		t.Fatalf("putting key: %v", err)
	}

	err = store.PutKey("app-b", "also-old")
	if err != nil {
		t.Fatalf("putting key: %v", err)
	}

	// Cutoff in the future: every stored key predates it and must be pruned
	// across all apps.
	store.CleanupAllBefore(time.Now().Add(time.Hour))

	for _, want := range []struct{ app, key string }{{"app-a", "old"}, {"app-b", "also-old"}} {
		exists, existsErr := store.KeyExists(want.app, want.key)
		if existsErr != nil {
			t.Fatalf("checking %s/%s: %v", want.app, want.key, existsErr)
		}

		if exists {
			t.Errorf("key %s/%s should have been pruned", want.app, want.key)
		}
	}
}
