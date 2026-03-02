package memory

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("memory entry not found")

type MemoryCapabilities struct {
	Persistent     bool
	VectorSearch   bool
	SessionSupport bool
}

type MemoryEntry struct {
	ID        string
	Key       string
	Content   string
	Category  *string
	Metadata  map[string]string
	CreatedAt string
	UpdatedAt string
}

type MemoryBackend interface {
	Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error
	Recall(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error)
	Get(ctx context.Context, key string) (*MemoryEntry, error)
	Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)
	Forget(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Close() error
	List(ctx context.Context, category *string) ([]MemoryEntry, error)
	Count(ctx context.Context, category *string) (int, error)
	Compact(ctx context.Context) error
	Export(ctx context.Context, path string) error
	Import(ctx context.Context, path string) error
}

type Registry struct {
	backends map[string]MemoryBackend
}

func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]MemoryBackend),
	}
}

func (r *Registry) Register(name string, backend MemoryBackend) {
	r.backends[name] = backend
}

func (r *Registry) Get(name string) (MemoryBackend, bool) {
	backend, ok := r.backends[name]
	return backend, ok
}

func NewBackend(backendType string, config map[string]string) (MemoryBackend, error) {
	switch backendType {
	case "none":
		return &NoneMemoryBackend{}, nil
	case "sqlite":
		dbPath := ""
		if path, ok := config["path"]; ok {
			dbPath = path
		}
		return NewSQLiteMemoryBackend(dbPath)
	default:
		return nil, errors.New("unknown backend type: " + backendType)
	}
}
