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

		store.CleanupBefore("", cleanupBefore)

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

		store.CleanupBefore("", cleanupBefore)

		exists, err = store.KeyExists("", randomKey)
		if err != nil {
			t.Error(err)
		}

		if !exists {
			t.Error("key should exist")
		}
	}
}
