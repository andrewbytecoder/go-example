package cacheaside

import (
	"context"
	"fmt"
	"log"
	"time"

	"cachestrategysdemo/internal/cache"
	"cachestrategysdemo/internal/database"
	"cachestrategysdemo/internal/models"
)

// 使用总结： 先从cache获取，如果获取不到再从db获取，如果db获取不到 返回nil 如果db能获取到，则更新cache

// GetUser retrieves user info using Cache-Aside strategy.
// It requires cache and database interfaces, along with config.
func GetUser(ctx context.Context, userID string, db database.Database, memCache cache.Cache, keyPrefix string, ttl time.Duration) (*models.User, error) {
	cacheKey := keyPrefix + userID

	// 1. Check cache first
	if cachedVal, found := memCache.Get(cacheKey); found {
		if user, ok := cachedVal.(*models.User); ok {
			log.Println("[Cache-Aside] Cache Hit for user:", userID)
			return user, nil
		}
		// 这里出现说明缓存别修改过，不应该出现这种类型
		memCache.Delete(cacheKey) // Remove bad data
	}

	// 2. Cache Miss
	log.Println("[Cache-Aside] Cache Miss for user:", userID)

	// 3. Fetch from Database
	user, err := db.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from DB: %w", err)
	}
	if user == nil {
		// Handle not found (optional: cache 'not found' with a short TTL)
		return nil, nil // Or return a specific "not found" error
	}

	// 4. Store data into cache
	memCache.Set(cacheKey, user, ttl)
	log.Println("[Cache-Aside] User stored in cache:", userID)

	// 5. Return data
	return user, nil
}
