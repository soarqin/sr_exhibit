package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/soar/sr_exhibit/models"
)

const (
	// DefaultCacheDir is the default cache directory
	DefaultCacheDir = ".cache"
	// DefaultTTL is the default cache expiration time (1 month)
	DefaultTTL = 30 * 24 * time.Hour
	// cacheFileName is the cache file name
	cacheFileName = "players.json"
)

// PlayerCacheItem represents a player cache entry
type PlayerCacheItem struct {
	Data     models.PlayerData `json:"data"`
	CachedAt time.Time         `json:"cached_at"`
}

// PlayerCache handles player data caching
type PlayerCache struct {
	mu      sync.RWMutex
	dir     string
	ttl     time.Duration
	players map[string]*PlayerCacheItem
	dirty   bool // Marks if there are unsaved changes
}

// NewPlayerCache creates a new player cache
func NewPlayerCache(dir string, ttl time.Duration) (*PlayerCache, error) {
	if dir == "" {
		dir = DefaultCacheDir
	}
	if ttl == 0 {
		ttl = DefaultTTL
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &PlayerCache{
		dir:     dir,
		ttl:     ttl,
		players: make(map[string]*PlayerCacheItem),
	}

	// Load existing cache
	if err := cache.Load(); err != nil {
		// Load failure is not fatal, might be first run
		// Just log and return empty cache
		cache.players = make(map[string]*PlayerCacheItem)
	}

	return cache, nil
}

// Load loads cache from file
func (c *PlayerCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.filePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File not existing is normal
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var fileCache struct {
		Players map[string]*PlayerCacheItem `json:"players"`
	}

	if err := json.Unmarshal(data, &fileCache); err != nil {
		return fmt.Errorf("failed to parse cache file: %w", err)
	}

	if fileCache.Players != nil {
		c.players = fileCache.Players
	}

	return nil
}

// Save saves cache to file
func (c *PlayerCache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.dirty {
		return nil
	}

	path := c.filePath()
	fileCache := struct {
		Players map[string]*PlayerCacheItem `json:"players"`
	}{
		Players: c.players,
	}

	data, err := json.MarshalIndent(fileCache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize cache: %w", err)
	}

	// Write to temp file first, then rename (atomic operation)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to save cache file: %w", err)
	}

	c.dirty = false
	return nil
}

// Get retrieves cached player data
// Returns (data, true) if valid cache found
// Returns (nil, false) if cache doesn't exist or has expired
func (c *PlayerCache) Get(playerID string) (*models.PlayerData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.players[playerID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(item.CachedAt) > c.ttl {
		return nil, false
	}

	return &item.Data, true
}

// Set sets cache entry
func (c *PlayerCache) Set(playerID string, data models.PlayerData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.players[playerID] = &PlayerCacheItem{
		Data:     data,
		CachedAt: time.Now(),
	}
	c.dirty = true
}

// CleanExpired removes expired cache entries
func (c *PlayerCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, item := range c.players {
		if now.Sub(item.CachedAt) > c.ttl {
			delete(c.players, id)
			removed++
		}
	}

	if removed > 0 {
		c.dirty = true
	}

	return removed
}

// Clear clears all cache
func (c *PlayerCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.players = make(map[string]*PlayerCacheItem)
	c.dirty = true

	// Delete cache file
	path := c.filePath()
	os.Remove(path)
}

// filePath returns the cache file path
func (c *PlayerCache) filePath() string {
	return filepath.Join(c.dir, cacheFileName)
}

// Stats returns cache statistics
func (c *PlayerCache) Stats() (total, expired int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	for _, item := range c.players {
		total++
		if now.Sub(item.CachedAt) > c.ttl {
			expired++
		}
	}
	return
}

// GetPlayersMap returns a copy of all player data
func (c *PlayerCache) GetPlayersMap() map[string]models.PlayerData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]models.PlayerData, len(c.players))
	for id, item := range c.players {
		result[id] = item.Data
	}
	return result
}
