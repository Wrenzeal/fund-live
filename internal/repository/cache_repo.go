// Package repository contains cache implementations.
package repository

import (
	"context"
	"time"

	"github.com/RomaticDOG/fund/internal/domain"
	gocache "github.com/patrickmn/go-cache"
)

// ensure interface compliance
var _ domain.CacheRepository = (*MemoryCacheRepository)(nil)

// MemoryCacheRepository implements CacheRepository using go-cache.
type MemoryCacheRepository struct {
	cache *gocache.Cache
}

// NewMemoryCacheRepository creates a new in-memory cache repository.
func NewMemoryCacheRepository(defaultTTL, cleanupInterval time.Duration) *MemoryCacheRepository {
	return &MemoryCacheRepository{
		cache: gocache.New(defaultTTL, cleanupInterval),
	}
}

// Get retrieves a value from cache.
func (r *MemoryCacheRepository) Get(ctx context.Context, key string) (interface{}, bool) {
	return r.cache.Get(key)
}

// Set stores a value in cache with TTL in seconds.
func (r *MemoryCacheRepository) Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error {
	r.cache.Set(key, value, time.Duration(ttlSeconds)*time.Second)
	return nil
}
