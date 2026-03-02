package memory

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNoneMemoryBackend(t *testing.T) {
	ctx := context.Background()
	backend := &NoneMemoryBackend{}

	t.Run("Store", func(t *testing.T) {
		err := backend.Store(ctx, "test-key", "test-content", nil, nil)
		if err != nil {
			t.Errorf("NoneMemoryBackend.Store() error = %v", err)
		}
	})

	t.Run("Recall", func(t *testing.T) {
		entries, err := backend.Recall(ctx, "test-query", 10, nil)
		if err != nil {
			t.Errorf("NoneMemoryBackend.Recall() error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("NoneMemoryBackend.Recall() = %v, want empty", entries)
		}
	})

	t.Run("Search", func(t *testing.T) {
		entries, err := backend.Search(ctx, "test-query", 10)
		if err != nil {
			t.Errorf("NoneMemoryBackend.Search() error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("NoneMemoryBackend.Search() = %v, want empty", entries)
		}
	})

	t.Run("Forget", func(t *testing.T) {
		err := backend.Forget(ctx, "test-key")
		if err != nil {
			t.Errorf("NoneMemoryBackend.Forget() error = %v", err)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		err := backend.Clear(ctx)
		if err != nil {
			t.Errorf("NoneMemoryBackend.Clear() error = %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := backend.Close()
		if err != nil {
			t.Errorf("NoneMemoryBackend.Close() error = %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		entries, err := backend.List(ctx, nil)
		if err != nil {
			t.Errorf("NoneMemoryBackend.List() error = %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("NoneMemoryBackend.List() = %v, want empty", entries)
		}
	})

	t.Run("Count", func(t *testing.T) {
		count, err := backend.Count(ctx, nil)
		if err != nil {
			t.Errorf("NoneMemoryBackend.Count() error = %v", err)
		}
		if count != 0 {
			t.Errorf("NoneMemoryBackend.Count() = %d, want 0", count)
		}
	})
}

func TestSQLiteMemoryBackend(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	backend, err := NewSQLiteMemoryBackend(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteMemoryBackend() error = %v", err)
	}
	defer backend.Close()

	t.Run("Store and Recall", func(t *testing.T) {
		err := backend.Store(ctx, "key1", "test content 1", nil, nil)
		if err != nil {
			t.Errorf("Store() error = %v", err)
		}

		entries, err := backend.Recall(ctx, "test", 10, nil)
		if err != nil {
			t.Errorf("Recall() error = %v", err)
		}
		if len(entries) == 0 {
			t.Error("Recall() should return entries")
		}
	})

	t.Run("Store multiple entries", func(t *testing.T) {
		keys := []string{"key2", "key3", "key4"}
		for _, key := range keys {
			err := backend.Store(ctx, key, "content "+key, nil, nil)
			if err != nil {
				t.Errorf("Store(%s) error = %v", key, err)
			}
		}

		count, err := backend.Count(ctx, nil)
		if err != nil {
			t.Errorf("Count() error = %v", err)
		}
		if count < len(keys)+1 {
			t.Errorf("Count() = %d, want at least %d", count, len(keys)+1)
		}
	})

	t.Run("List entries", func(t *testing.T) {
		entries, err := backend.List(ctx, nil)
		if err != nil {
			t.Errorf("List() error = %v", err)
		}
		if len(entries) == 0 {
			t.Error("List() should return entries")
		}
	})

	t.Run("Forget entry", func(t *testing.T) {
		err := backend.Store(ctx, "temp-key", "temp content", nil, nil)
		if err != nil {
			t.Errorf("Store() error = %v", err)
		}

		err = backend.Forget(ctx, "temp-key")
		if err != nil {
			t.Errorf("Forget() error = %v", err)
		}

		entries, err := backend.Recall(ctx, "temp", 10, nil)
		if err != nil {
			t.Errorf("Recall() error = %v", err)
		}
		for _, entry := range entries {
			if entry.Key == "temp-key" {
				t.Error("Forgotten entry should not be recalled")
			}
		}
	})

	t.Run("Clear all", func(t *testing.T) {
		err := backend.Clear(ctx)
		if err != nil {
			t.Errorf("Clear() error = %v", err)
		}

		count, err := backend.Count(ctx, nil)
		if err != nil {
			t.Errorf("Count() error = %v", err)
		}
		if count != 0 {
			t.Errorf("Count() after Clear() = %d, want 0", count)
		}
	})

	t.Run("Default database path", func(t *testing.T) {
		backend, err := NewSQLiteMemoryBackend("")
		if err != nil {
			t.Fatalf("NewSQLiteMemoryBackend() error = %v", err)
		}
		defer backend.Close()

		if backend.dbPath == "" {
			t.Error("Database path should not be empty")
		}
	})
}

func TestHashContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "short content",
			content: "hello",
		},
		{
			name:    "long content",
			content: "this is a longer piece of content to hash",
		},
		{
			name:    "empty content",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashContent(tt.content)
			if hash == "" {
				t.Error("hashContent() should return non-empty hash")
			}
		})
	}
}