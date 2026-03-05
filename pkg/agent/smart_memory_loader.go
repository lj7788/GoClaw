package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// CachedMemoryEntry represents a cached memory entry with timestamp
type CachedMemoryEntry struct {
	Result     string
	Timestamp  time.Time
	Query      string
	EntryCount int
}

// SmartMemoryLoader is an intelligent memory loader with caching and optimization
type SmartMemoryLoader struct {
	mu                 sync.RWMutex
	cache              map[string]*CachedMemoryEntry
	cacheTTL           time.Duration
	maxCacheSize       int
	minRelevanceScore  float64
	maxEntries         int
	enableCache        bool
	enableDynamicLimit bool
	categoryFilter     *string
}

// SmartMemoryLoaderConfig configures the SmartMemoryLoader
type SmartMemoryLoaderConfig struct {
	CacheTTL           time.Duration
	MaxCacheSize       int
	MinRelevanceScore  float64
	MaxEntries         int
	EnableCache        bool
	EnableDynamicLimit bool
	CategoryFilter     *string
}

// DefaultSmartMemoryLoaderConfig returns default configuration
func DefaultSmartMemoryLoaderConfig() SmartMemoryLoaderConfig {
	return SmartMemoryLoaderConfig{
		CacheTTL:           5 * time.Minute,
		MaxCacheSize:       100,
		MinRelevanceScore:  0.1,
		MaxEntries:         5,
		EnableCache:        true,
		EnableDynamicLimit: true,
		CategoryFilter:     nil,
	}
}

// NewSmartMemoryLoader creates a new SmartMemoryLoader with default config
func NewSmartMemoryLoader() *SmartMemoryLoader {
	return NewSmartMemoryLoaderWithConfig(DefaultSmartMemoryLoaderConfig())
}

// NewSmartMemoryLoaderWithConfig creates a new SmartMemoryLoader with custom config
func NewSmartMemoryLoaderWithConfig(config SmartMemoryLoaderConfig) *SmartMemoryLoader {
	return &SmartMemoryLoader{
		cache:              make(map[string]*CachedMemoryEntry),
		cacheTTL:           config.CacheTTL,
		maxCacheSize:       config.MaxCacheSize,
		minRelevanceScore:  config.MinRelevanceScore,
		maxEntries:         config.MaxEntries,
		enableCache:        config.EnableCache,
		enableDynamicLimit: config.EnableDynamicLimit,
		categoryFilter:     config.CategoryFilter,
	}
}

// LoadMemory loads relevant memory with caching and optimization
func (l *SmartMemoryLoader) LoadMemory(ctx context.Context, memory Memory, query string) (string, error) {
	// Check cache first if enabled
	if l.enableCache {
		if cached, found := l.getFromCache(query); found {
			log.Printf("MemoryLoader: Using cached result for query: %s", query)
			return cached, nil
		}
	}

	// Determine the number of entries to retrieve
	limit := l.determineEntryLimit(query)

	// Retrieve relevant memory entries
	entries, err := memory.Recall(ctx, query, limit, l.categoryFilter)
	if err != nil {
		return "", fmt.Errorf("failed to recall memory: %w", err)
	}

	// Filter by relevance score and category
	filteredEntries := l.filterEntries(entries)

	if len(filteredEntries) == 0 {
		return "No relevant memory found", nil
	}

	// Build memory context
	context := l.buildContext(filteredEntries)

	// Cache the result if enabled
	if l.enableCache {
		l.addToCache(query, context, len(filteredEntries))
	}

	log.Printf("MemoryLoader: Loaded %d entries for query: %s", len(filteredEntries), query)
	return context, nil
}

// getFromCache retrieves a cached result if valid
func (l *SmartMemoryLoader) getFromCache(query string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	cached, exists := l.cache[query]
	if !exists {
		return "", false
	}

	// Check if cache entry is expired
	if time.Since(cached.Timestamp) > l.cacheTTL {
		return "", false
	}

	return cached.Result, true
}

// addToCache adds a result to the cache
func (l *SmartMemoryLoader) addToCache(query, result string, entryCount int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Remove expired entries
	l.cleanExpiredCache()

	// Check cache size limit
	if len(l.cache) >= l.maxCacheSize {
		// Remove oldest entry (simple LRU)
		var oldestKey string
		var oldestTime time.Time
		for key, entry := range l.cache {
			if oldestTime.IsZero() || entry.Timestamp.Before(oldestTime) {
				oldestTime = entry.Timestamp
				oldestKey = key
			}
		}
		if oldestKey != "" {
			delete(l.cache, oldestKey)
		}
	}

	l.cache[query] = &CachedMemoryEntry{
		Result:     result,
		Timestamp:  time.Now(),
		Query:      query,
		EntryCount: entryCount,
	}
}

// cleanExpiredCache removes expired cache entries
func (l *SmartMemoryLoader) cleanExpiredCache() {
	now := time.Now()
	for key, entry := range l.cache {
		if now.Sub(entry.Timestamp) > l.cacheTTL {
			delete(l.cache, key)
		}
	}
}

// determineEntryLimit dynamically determines the number of entries to retrieve
func (l *SmartMemoryLoader) determineEntryLimit(query string) int {
	if !l.enableDynamicLimit {
		return l.maxEntries
	}

	// Simple heuristic: longer queries might need more context
	queryLength := len(strings.Fields(query))
	
	switch {
	case queryLength <= 3:
		return 3 // Short queries need less context
	case queryLength <= 10:
		return 5 // Medium queries need moderate context
	default:
		return 8 // Long queries need more context
	}
}

// filterEntries filters entries by relevance score and other criteria
func (l *SmartMemoryLoader) filterEntries(entries []MemoryEntry) []MemoryEntry {
	var filtered []MemoryEntry

	for _, entry := range entries {
		// Filter by relevance score
		if entry.Score != nil && *entry.Score < l.minRelevanceScore {
			continue
		}

		// Filter by category if specified
		if l.categoryFilter != nil {
			if entry.Category != nil && !strings.Contains(*entry.Category, *l.categoryFilter) {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	return filtered
}

// buildContext builds memory context string from entries
func (l *SmartMemoryLoader) buildContext(entries []MemoryEntry) string {
	var context strings.Builder

	for i, entry := range entries {
		if entry.Score != nil {
			context.WriteString(fmt.Sprintf("%d. [%.2f] %s\n", i+1, *entry.Score, entry.Content))
		} else {
			context.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry.Content))
		}
	}

	return context.String()
}

// ClearCache clears the memory cache
func (l *SmartMemoryLoader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cache = make(map[string]*CachedMemoryEntry)
	log.Printf("MemoryLoader: Cache cleared")
}

// GetCacheStats returns cache statistics
func (l *SmartMemoryLoader) GetCacheStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	totalEntries := 0
	for _, entry := range l.cache {
		totalEntries += entry.EntryCount
	}

	return map[string]interface{}{
		"cache_size":       len(l.cache),
		"max_cache_size":   l.maxCacheSize,
		"total_entries":    totalEntries,
		"cache_ttl":        l.cacheTTL.String(),
		"min_relevance":    l.minRelevanceScore,
		"max_entries":      l.maxEntries,
		"cache_enabled":    l.enableCache,
		"dynamic_limit":    l.enableDynamicLimit,
		"category_filter":  l.categoryFilter,
	}
}

// SetCategoryFilter sets a category filter for memory loading
func (l *SmartMemoryLoader) SetCategoryFilter(category *string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.categoryFilter = category
}

// SetMinRelevanceScore sets the minimum relevance score threshold
func (l *SmartMemoryLoader) SetMinRelevanceScore(score float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minRelevanceScore = score
}