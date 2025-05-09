package context

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"proxyllama/config"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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

// CacheProvider defines the interface for conversation caching
type CacheProvider interface {
	Get(userID string, conversationID int) (*ConversationContext, bool)
	Set(convContext *ConversationContext)
	Remove(userID string, conversationID int)
	Clear()
	Size() int
	Stop()
}

// InMemoryCache provides a thread-safe in-memory cache for conversation contexts
type InMemoryCache struct {
	entries         map[cacheKey]cacheEntry
	mutex           sync.RWMutex
	ttl             time.Duration
	janitorInterval time.Duration
	stopJanitor     chan bool
}

// RedisCache provides Redis-based caching for conversation contexts
type RedisCache struct {
	client     *redis.Client
	ttl        time.Duration
	keyPrefix  string
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// Global cache instance with default TTL of 30 minutes
var (
	Cache           CacheProvider
	defaultTTL      = 30 * time.Minute
	janitorCycleTTL = 5 * time.Minute // Run cleanup every 5 minutes
)

// InitCache initializes the conversation cache with the specified TTL
func InitCache(ttl time.Duration) CacheProvider {
	if ttl == 0 {
		ttl = defaultTTL
	}

	// Check if Redis is enabled in the config
	conf := config.GetConfig()

	if conf.Redis.Enabled {
		log.Printf("Initializing Redis cache with TTL of %v", ttl)
		cache, err := NewRedisCache(ttl)
		if err != nil {
			log.Printf("Error initializing Redis cache: %v, falling back to in-memory cache", err)
		} else {
			Cache = cache
			return cache
		}
	}

	// Fall back to in-memory cache if Redis is not enabled or failed
	log.Printf("Initializing in-memory cache with TTL of %v", ttl)
	cache := NewInMemoryCache(ttl)
	Cache = cache
	return cache
}

// === In-Memory Cache Implementation ===

// NewInMemoryCache creates a new in-memory cache with the specified TTL
func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		entries:         make(map[cacheKey]cacheEntry),
		ttl:             ttl,
		janitorInterval: janitorCycleTTL,
		stopJanitor:     make(chan bool),
	}

	// Start the janitor to clean expired entries
	go cache.startJanitor()

	log.Printf("InMemoryCache initialized with TTL of %v", ttl)
	return cache
}

// startJanitor periodically cleans up expired cache entries
func (c *InMemoryCache) startJanitor() {
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
func (c *InMemoryCache) Stop() {
	c.stopJanitor <- true
}

// cleanExpired removes expired entries from the cache
func (c *InMemoryCache) cleanExpired() {
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
		log.Printf("Removed %d expired conversations from in-memory cache", len(expiredKeys))
	}
}

// Get retrieves a conversation context from the cache
func (c *InMemoryCache) Get(userID string, conversationID int) (*ConversationContext, bool) {
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
func (c *InMemoryCache) Set(convContext *ConversationContext) {
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
func (c *InMemoryCache) Remove(userID string, conversationID int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := cacheKey{userID: userID, conversationID: conversationID}
	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *InMemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[cacheKey]cacheEntry)
	log.Printf("InMemoryCache cleared")
}

// Size returns the current number of entries in the cache
func (c *InMemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.entries)
}

// === Redis Cache Implementation ===

// NewRedisCache creates a new Redis-based cache
func NewRedisCache(ttl time.Duration) (*RedisCache, error) {
	conf := config.GetConfig()

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", conf.Redis.Host, conf.Redis.Port),
		Password:     conf.Redis.Password,
		DB:           conf.Redis.DB,
		PoolSize:     conf.Redis.PoolSize,
		MinIdleConns: conf.Redis.MinIdleConnections,
		ReadTimeout:  conf.Redis.ConnectTimeout,
		WriteTimeout: conf.Redis.ConnectTimeout,
		DialTimeout:  conf.Redis.ConnectTimeout,
	})

	// Use provided TTL or config TTL
	if ttl == 0 {
		ttl = conf.Redis.ConversationTTL
	}

	// Create background context for Redis operations
	ctx, cancel := context.WithCancel(context.Background())

	// Test the connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		cancel() // Cancel the context if connection fails
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	cache := &RedisCache{
		client:     rdb,
		ttl:        ttl,
		keyPrefix:  "proxyllama:conversation:",
		ctx:        ctx,
		cancelFunc: cancel,
	}

	log.Printf("RedisCache initialized with TTL of %v at %s:%d", ttl, conf.Redis.Host, conf.Redis.Port)

	// Run a health check routine
	go cache.startHealthCheck()

	return cache, nil
}

// startHealthCheck periodically checks the Redis connection
func (c *RedisCache) startHealthCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := c.client.Ping(c.ctx).Result()
			if err != nil {
				log.Printf("Redis health check failed: %v", err)
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// createRedisKey generates a Redis key for a conversation
func (c *RedisCache) createRedisKey(userID string, conversationID int) string {
	return fmt.Sprintf("%s%s:%d", c.keyPrefix, userID, conversationID)
}

// Get retrieves a conversation context from Redis
func (c *RedisCache) Get(userID string, conversationID int) (*ConversationContext, bool) {
	key := c.createRedisKey(userID, conversationID)

	// Get the data from Redis
	data, err := c.client.Get(c.ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Error retrieving from Redis cache: %v", err)
		}
		return nil, false
	}

	// Deserialize the conversation context
	var convContext ConversationContext
	if err := json.Unmarshal(data, &convContext); err != nil {
		log.Printf("Error deserializing conversation context from Redis: %v", err)
		return nil, false
	}

	log.Printf("Retrieved conversation %d for user %s from Redis cache", conversationID, userID)
	return &convContext, true
}

// Set adds or updates a conversation context in Redis
func (c *RedisCache) Set(convContext *ConversationContext) {
	if convContext == nil {
		log.Printf("Attempted to cache nil conversation context")
		return
	}

	// Serialize the conversation context
	data, err := json.Marshal(convContext)
	if err != nil {
		log.Printf("Error serializing conversation context for Redis: %v", err)
		return
	}

	// Create the Redis key
	key := c.createRedisKey(convContext.UserID, convContext.ConversationID)

	// Set the data in Redis with expiration
	err = c.client.Set(c.ctx, key, data, c.ttl).Err()
	if err != nil {
		log.Printf("Error setting conversation context in Redis: %v", err)
		return
	}

	log.Printf("Saved conversation %d for user %s to Redis cache", convContext.ConversationID, convContext.UserID)
}

// Remove removes a conversation context from Redis
func (c *RedisCache) Remove(userID string, conversationID int) {
	key := c.createRedisKey(userID, conversationID)

	err := c.client.Del(c.ctx, key).Err()
	if err != nil && err != redis.Nil {
		log.Printf("Error removing conversation context from Redis: %v", err)
	}
}

// Clear removes all entries with the cache prefix
func (c *RedisCache) Clear() {
	// Find all keys with our prefix
	iter := c.client.Scan(c.ctx, 0, c.keyPrefix+"*", 100).Iterator()

	var keysToDelete []string
	for iter.Next(c.ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
	}

	if err := iter.Err(); err != nil {
		log.Printf("Error scanning Redis keys: %v", err)
		return
	}

	// Delete all found keys if there are any
	if len(keysToDelete) > 0 {
		err := c.client.Del(c.ctx, keysToDelete...).Err()
		if err != nil {
			log.Printf("Error deleting keys from Redis: %v", err)
		} else {
			log.Printf("Cleared %d entries from Redis cache", len(keysToDelete))
		}
	}
}

// Size returns an approximate count of entries in the Redis cache
func (c *RedisCache) Size() int {
	// Count keys with our prefix
	count, err := c.client.Keys(c.ctx, c.keyPrefix+"*").Result()
	if err != nil {
		log.Printf("Error counting Redis cache entries: %v", err)
		return 0
	}
	return len(count)
}

// Stop closes the Redis client connection
func (c *RedisCache) Stop() {
	c.cancelFunc()
	if err := c.client.Close(); err != nil {
		log.Printf("Error closing Redis client: %v", err)
	}
}

// GetCache returns the global cache instance, initializing it if necessary
func GetCache() CacheProvider {
	if Cache == nil {
		conf := config.GetConfig()
		InitCache(conf.Redis.ConversationTTL)
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
