// Package memory provides memory functionality for GoClaw.
package memory

import (
	"context"
)

// NoneMemoryBackend is a memory backend that does nothing.
type NoneMemoryBackend struct{}

// NewNoneMemoryBackend creates a new NoneMemoryBackend.
func NewNoneMemoryBackend() *NoneMemoryBackend {
	return &NoneMemoryBackend{}
}

// Recall retrieves relevant memory entries based on a query.
func (b *NoneMemoryBackend) Recall(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error) {
	return []MemoryEntry{}, nil
}

// Store saves a memory entry.
func (b *NoneMemoryBackend) Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error {
	return nil
}

// Get retrieves a specific memory entry by key.
func (b *NoneMemoryBackend) Get(ctx context.Context, key string) (*MemoryEntry, error) {
	return nil, ErrNotFound
}

// Delete removes a memory entry.
func (b *NoneMemoryBackend) Delete(ctx context.Context, key string) error {
	return nil
}

// Forget removes a memory entry by key.
func (b *NoneMemoryBackend) Forget(ctx context.Context, key string) error {
	return nil
}

// Clear removes all memory entries.
func (b *NoneMemoryBackend) Clear(ctx context.Context) error {
	return nil
}

// List returns all memory entries.
func (b *NoneMemoryBackend) List(ctx context.Context, category *string) ([]MemoryEntry, error) {
	return []MemoryEntry{}, nil
}

// Count returns the number of memory entries.
func (b *NoneMemoryBackend) Count(ctx context.Context, category *string) (int, error) {
	return 0, nil
}

// Compact compacts the memory to save space.
func (b *NoneMemoryBackend) Compact(ctx context.Context) error {
	return nil
}

// Search searches memory entries.
func (b *NoneMemoryBackend) Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error) {
	return []MemoryEntry{}, nil
}

// Close closes the memory backend.
func (b *NoneMemoryBackend) Close() error {
	return nil
}

// Export exports memory entries to a file.
func (b *NoneMemoryBackend) Export(ctx context.Context, path string) error {
	return nil
}

// Import imports memory entries from a file.
func (b *NoneMemoryBackend) Import(ctx context.Context, path string) error {
	return nil
}
