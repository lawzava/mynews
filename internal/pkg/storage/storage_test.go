//nolint:gosec // allow weak generators on tests
package storage_test

import (
	"fmt"
	"math/rand"
	"mynews/internal/pkg/storage"
	"testing"
	"time"
)

func TestStorage(t *testing.T) {
	t.Parallel()

	store := storage.New()

	rGen := rand.New(rand.NewSource(time.Now().Unix()))

	for i := 0; i < 1000; i++ {
		randomKey := fmt.Sprint(rGen.Int())

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

	for i := 0; i < 1000; i++ {
		randomKey := fmt.Sprint(rGen.Int())

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
