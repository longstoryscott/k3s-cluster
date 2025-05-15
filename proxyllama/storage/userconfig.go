package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"proxyllama/config"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

const userConfigKeyPrefix = "proxyllama:userconfig:"

// getUserConfigCacheKey constructs the Redis key for a user's config
func getUserConfigCacheKey(userID string) string {
	return userConfigKeyPrefix + userID
}

// getUserConfigFromCache tries to get user config from Redis
func getUserConfigFromCache(ctx context.Context, userID string) (*config.UserConfig, bool) {
	if !IsStorageCacheEnabled() {
		return nil, false
	}
	key := getUserConfigCacheKey(userID)
	data, err := redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}
	var userConfig config.UserConfig
	if err := json.Unmarshal(data, &userConfig); err != nil {
		return nil, false
	}
	return &userConfig, true
}

// cacheUserConfig stores user config in Redis
func cacheUserConfig(ctx context.Context, userID string, cfg *config.UserConfig) {
	if !IsStorageCacheEnabled() || cfg == nil {
		return
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	ttl := 24 * time.Hour // Cache user config for 24h, adjust as needed
	key := getUserConfigCacheKey(userID)
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file":   filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line":   line,
		"userID": userID,
		"ttl":    ttl,
	}).Info("Caching user config")
	redisClient.Set(ctx, key, data, ttl)
}

// GetUserConfig retrieves user configuration from database
func GetUserConfig(ctx context.Context, userID string) (*config.UserConfig, error) {
	// Ensure user exists
	if err := EnsureUser(ctx, userID); err != nil {
		return nil, err
	}

	// Try to get from cache first
	usrCfg, found := getUserConfigFromCache(ctx, userID)
	if found {
		return usrCfg, nil
	}

	// Parse JSON into config struct
	var usrConfig config.UserConfig
	err := Pool.QueryRow(ctx, GetQuery("user.get_config"), userID).Scan(&usrConfig)
	if err != nil {
		return nil, err
	}

	// Ensure all required fields have values by merging with defaults
	config.MergeWithDefaultConfig(&usrConfig)

	// Cache for future use
	cacheUserConfig(ctx, userID, &usrConfig)

	return &usrConfig, nil
}

// UpdateUserConfig saves user configuration to database
func UpdateUserConfig(ctx context.Context, userID string, cfg *config.UserConfig) error {
	// Convert config to JSON
	configJson, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	// Update in database
	_, err = Pool.Exec(ctx, GetQuery("user.update_config"), configJson, userID)
	if err != nil {
		return err
	}

	// Update the cache
	cacheUserConfig(ctx, userID, cfg)

	return nil
}

// GetUserConfigWithTimeout is a wrapper that adds a timeout to GetUserConfig
func GetUserConfigWithTimeout(userID string) (*config.UserConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return GetUserConfig(ctx, userID)
}
