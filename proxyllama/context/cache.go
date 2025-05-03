package context

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Cache entry with TTL support
type cacheEntry struct {
	context   *ConversationContext
	expiresAt time.Time
}

// Cache key combines userID and conversationID for unique identification
type cacheKey struct {
	userID         string
	conversationID int
}

// ConversationCache provides a thread-safe in-memory cache for conversation contexts
type ConversationCache struct {
	entries         map[cacheKey]cacheEntry
	mutex           sync.RWMutex
	ttl             time.Duration
	janitorInterval time.Duration
	stopJanitor     chan bool
}

// Global cache instance with default TTL of 30 minutes
var (
	Cache           *ConversationCache
	defaultTTL      = 30 * time.Minute
	janitorCycleTTL = 5 * time.Minute // Run cleanup every 5 minutes
)

// InitCache initializes the conversation cache with the specified TTL
func InitCache(ttl time.Duration) *ConversationCache {
	if ttl == 0 {
		ttl = defaultTTL
	}

	cache := &ConversationCache{
		entries:         make(map[cacheKey]cacheEntry),
		ttl:             ttl,
		janitorInterval: janitorCycleTTL,
		stopJanitor:     make(chan bool),
	}

	// Start the janitor to clean expired entries
	go cache.startJanitor()

	Cache = cache
	log.Printf("ConversationCache initialized with TTL of %v", ttl)
	return cache
}

// startJanitor periodically cleans up expired cache entries
func (c *ConversationCache) startJanitor() {
	ticker := time.NewTicker(c.janitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanExpired()
		case <-c.stopJanitor:
			return
		}
	}
}

// Stop stops the janitor goroutine
func (c *ConversationCache) Stop() {
	c.stopJanitor <- true
}

// cleanExpired removes expired entries from the cache
func (c *ConversationCache) cleanExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	var expiredKeys []cacheKey

	// Find all expired keys
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			expiredKeys = append(expiredKeys, k)
		}
	}

	// Remove expired entries
	for _, k := range expiredKeys {
		delete(c.entries, k)
	}

	if len(expiredKeys) > 0 {
		log.Printf("Removed %d expired conversations from cache", len(expiredKeys))
	}
}

// Get retrieves a conversation context from the cache
func (c *ConversationCache) Get(userID string, conversationID int) (*ConversationContext, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	key := cacheKey{userID: userID, conversationID: conversationID}
	entry, found := c.entries[key]

	// Check if entry exists and is not expired
	if !found {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// We'll let the janitor clean this up later
		return nil, false
	}

	return entry.context, true
}

// Set adds or updates a conversation context in the cache
func (c *ConversationCache) Set(convContext *ConversationContext) {
	if convContext == nil {
		log.Printf("Attempted to cache nil conversation context")
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := cacheKey{
		userID:         convContext.UserID,
		conversationID: convContext.ConversationID,
	}

	c.entries[key] = cacheEntry{
		context:   convContext,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Remove removes a conversation context from the cache
func (c *ConversationCache) Remove(userID string, conversationID int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := cacheKey{userID: userID, conversationID: conversationID}
	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *ConversationCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[cacheKey]cacheEntry)
	log.Printf("ConversationCache cleared")
}

// Size returns the current number of entries in the cache
func (c *ConversationCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.entries)
}

// GetCache returns the global cache instance, initializing it if necessary
func GetCache() *ConversationCache {
	if Cache == nil {
		InitCache(defaultTTL)
	}
	return Cache
}

// GetCachedConversation retrieves a conversation from cache if available,
// otherwise retrieves from storage and caches the result
func GetCachedConversation(userID string, model string, conversationID int) (*ConversationContext, error) {
	cache := GetCache()

	// Try to get from cache first
	if cachedContext, found := cache.Get(userID, conversationID); found {
		return cachedContext, nil
	}

	// Not in cache, fetch from storage
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conversationIDPtr := &conversationID
	convContext, err := GetOrCreateConversation(ctx, userID, model, conversationIDPtr)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve conversation: %w", err)
	}

	// Cache the result for future use
	cache.Set(convContext)

	return convContext, nil
}

// InvalidateConversation removes a conversation from the cache, forcing it to be reloaded on next access
func InvalidateConversation(userID string, conversationID int) {
	GetCache().Remove(userID, conversationID)
}
