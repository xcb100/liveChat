package tools

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache 定义缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置缓存值
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	// Del 删除缓存
	Del(ctx context.Context, keys ...string) error
	// Exists 检查缓存是否存在
	Exists(ctx context.Context, keys ...string) (bool, error)
	// Incr 自增1
	Incr(ctx context.Context, key string) (int64, error)
	// Expire 设置过期时间
	Expire(ctx context.Context, key string, expiration time.Duration) error
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	// 缓存数据映射
	cache map[string]memoryCacheItem
	// 互斥锁保证并发安全
	lock sync.RWMutex
	// 清理过期缓存的定时器
	cleanupTimer *time.Ticker
	// 清理间隔
	cleanupInterval time.Duration
}

// memoryCacheItem 内存缓存项
type memoryCacheItem struct {
	value      string
	expiration int64
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	mc := &MemoryCache{
		cache:           make(map[string]memoryCacheItem),
		cleanupInterval: cleanupInterval,
		cleanupTimer:    time.NewTicker(cleanupInterval),
	}

	// 启动清理过期缓存的goroutine
	go mc.cleanupExpired()

	return mc
}

// cleanupExpired 清理过期的缓存项
func (mc *MemoryCache) cleanupExpired() {
	for {
		select {
		case <-mc.cleanupTimer.C:
			mc.lock.Lock()
			now := time.Now().UnixNano()
			for k, v := range mc.cache {
				// 检查是否过期（expiration为0表示永不过期）
				if v.expiration > 0 && now > v.expiration {
					delete(mc.cache, k)
				}
			}
			mc.lock.Unlock()
		}
	}
}

// Get 获取缓存值
func (mc *MemoryCache) Get(ctx context.Context, key string) (string, error) {
	mc.lock.RLock()
	defer mc.lock.RUnlock()

	item, exists := mc.cache[key]
	if !exists {
		return "", redis.Nil // 使用与Redis相同的错误类型
	}

	// 检查是否过期
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return "", redis.Nil
	}

	return item.value, nil
}

// Set 设置缓存值
func (mc *MemoryCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	var expireAt int64
	if expiration > 0 {
		expireAt = time.Now().Add(expiration).UnixNano()
	}

	mc.cache[key] = memoryCacheItem{
		value:      value,
		expiration: expireAt,
	}

	return nil
}

// Del 删除缓存
func (mc *MemoryCache) Del(ctx context.Context, keys ...string) error {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	for _, key := range keys {
		delete(mc.cache, key)
	}

	return nil
}

// Exists 检查缓存是否存在
func (mc *MemoryCache) Exists(ctx context.Context, keys ...string) (bool, error) {
	mc.lock.RLock()
	defer mc.lock.RUnlock()

	now := time.Now().UnixNano()
	for _, key := range keys {
		item, exists := mc.cache[key]
		if !exists {
			return false, nil
		}
		// 检查是否过期
		if item.expiration > 0 && now > item.expiration {
			return false, nil
		}
	}

	return true, nil
}

// Incr 自增1
func (mc *MemoryCache) Incr(ctx context.Context, key string) (int64, error) {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	item, exists := mc.cache[key]
	var value int64

	if exists && (item.expiration == 0 || time.Now().UnixNano() <= item.expiration) {
		// 解析现有值
		v, err := strconv.ParseInt(item.value, 10, 64)
		if err != nil {
			return 0, err
		}
		value = v + 1
	} else {
		value = 1
	}

	// 更新缓存
	mc.cache[key] = memoryCacheItem{
		value:      strconv.FormatInt(value, 10),
		expiration: item.expiration, // 保持原过期时间
	}

	return value, nil
}

// Expire 设置过期时间
func (mc *MemoryCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	mc.lock.Lock()
	defer mc.lock.Unlock()

	item, exists := mc.cache[key]
	if !exists {
		return nil
	}

	var expireAt int64
	if expiration > 0 {
		expireAt = time.Now().Add(expiration).UnixNano()
	}

	item.expiration = expireAt
	mc.cache[key] = item

	return nil
}

// MultiLevelCache 多级缓存实现
type MultiLevelCache struct {
	// 第一级缓存：Redis
	redisCache *RedisCache
	// 第二级缓存：内存
	memoryCache *MemoryCache
	// 是否使用内存缓存（Redis不可用时为true）
	useMemoryCache bool
	// 互斥锁保证并发安全
	lock sync.RWMutex
	// 检查Redis可用性的定时器
	checkTimer *time.Ticker
	// 检查间隔
	checkInterval time.Duration
}

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Get 获取缓存值
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return rc.client.Get(ctx, key).Result()
}

// Set 设置缓存值
func (rc *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return rc.client.Set(ctx, key, value, expiration).Err()
}

// Del 删除缓存
func (rc *RedisCache) Del(ctx context.Context, keys ...string) error {
	return rc.client.Del(ctx, keys...).Err()
}

// Exists 检查缓存是否存在
func (rc *RedisCache) Exists(ctx context.Context, keys ...string) (bool, error) {
	result, err := rc.client.Exists(ctx, keys...).Result()
	return result > 0, err
}

// Incr 自增1
func (rc *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return rc.client.Incr(ctx, key).Result()
}

// Expire 设置过期时间
func (rc *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return rc.client.Expire(ctx, key, expiration).Err()
}

// MultiLevelCache 全局多级缓存实例
var MultiLevelCacheInstance *MultiLevelCache

// InitMultiLevelCache 初始化多级缓存
func InitMultiLevelCache(redisConf RedisConfig) error {
	// 初始化Redis客户端
	err := InitRedis(redisConf)
	if err != nil {
		// Redis初始化失败，直接使用内存缓存
		MultiLevelCacheInstance = NewMultiLevelCache(
			nil,
			5*time.Minute,  // 内存缓存清理间隔
			10*time.Second, // Redis可用性检查间隔
		)
		MultiLevelCacheInstance.useMemoryCache = true
		return nil
	}

	// 创建多级缓存实例
	MultiLevelCacheInstance = NewMultiLevelCache(
		RedisClient,
		5*time.Minute,  // 内存缓存清理间隔
		10*time.Second, // Redis可用性检查间隔
	)

	return nil
}

// GetCache 获取多级缓存实例
func GetCache() Cache {
	return MultiLevelCacheInstance
}

// NewMultiLevelCache 创建多级缓存实例
func NewMultiLevelCache(redisClient *redis.Client, memoryCleanupInterval, redisCheckInterval time.Duration) *MultiLevelCache {
	mc := &MultiLevelCache{
		redisCache:     nil,
		memoryCache:    NewMemoryCache(memoryCleanupInterval),
		useMemoryCache: false,
		checkInterval:  redisCheckInterval,
		checkTimer:     time.NewTicker(redisCheckInterval),
	}
	if redisClient != nil {
		mc.redisCache = NewRedisCache(redisClient)
	}
	if redisClient == nil {
		mc.useMemoryCache = true
	}

	// 启动检查Redis可用性的goroutine
	go mc.checkRedisAvailability()

	return mc
}

// checkRedisAvailability 检查Redis可用性
func (mc *MultiLevelCache) checkRedisAvailability() {
	for {
		select {
		case <-mc.checkTimer.C:
			if mc.redisCache == nil || mc.redisCache.client == nil {
				mc.lock.Lock()
				mc.useMemoryCache = true
				mc.lock.Unlock()
				continue
			}
			// 测试Redis连接
			_, err := mc.redisCache.client.Ping(Ctx).Result()

			mc.lock.Lock()
			if err != nil {
				// Redis不可用，切换到内存缓存
				if !mc.useMemoryCache {
					mc.useMemoryCache = true
					// 可以在这里添加日志记录
				}
			} else {
				// Redis可用，切换回Redis缓存
				if mc.useMemoryCache {
					mc.useMemoryCache = false
					// 可以在这里添加日志记录
				}
			}
			mc.lock.Unlock()
		}
	}
}

// Get 获取缓存值
func (mc *MultiLevelCache) Get(ctx context.Context, key string) (string, error) {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Get(ctx, key)
		}
		// 尝试从Redis获取
		value, err := mc.redisCache.Get(ctx, key)
		if err == nil {
			return value, nil
		}

		// Redis错误（包括不可用），切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 从内存缓存获取
	return mc.memoryCache.Get(ctx, key)
}

// Set 设置缓存值
func (mc *MultiLevelCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Set(ctx, key, value, expiration)
		}
		// 尝试设置到Redis
		err := mc.redisCache.Set(ctx, key, value, expiration)
		if err == nil {
			return nil
		}

		// Redis错误，切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 设置到内存缓存
	return mc.memoryCache.Set(ctx, key, value, expiration)
}

// Del 删除缓存
func (mc *MultiLevelCache) Del(ctx context.Context, keys ...string) error {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Del(ctx, keys...)
		}
		// 尝试从Redis删除
		err := mc.redisCache.Del(ctx, keys...)
		if err == nil {
			return nil
		}

		// Redis错误，切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 从内存缓存删除
	return mc.memoryCache.Del(ctx, keys...)
}

// Exists 检查缓存是否存在
func (mc *MultiLevelCache) Exists(ctx context.Context, keys ...string) (bool, error) {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Exists(ctx, keys...)
		}
		// 尝试从Redis检查
		exists, err := mc.redisCache.Exists(ctx, keys...)
		if err == nil {
			return exists, nil
		}

		// Redis错误，切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 从内存缓存检查
	return mc.memoryCache.Exists(ctx, keys...)
}

// Incr 自增1
func (mc *MultiLevelCache) Incr(ctx context.Context, key string) (int64, error) {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Incr(ctx, key)
		}
		// 尝试从Redis自增
		value, err := mc.redisCache.Incr(ctx, key)
		if err == nil {
			return value, nil
		}

		// Redis错误，切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 从内存缓存自增
	return mc.memoryCache.Incr(ctx, key)
}

// Expire 设置过期时间
func (mc *MultiLevelCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	mc.lock.RLock()
	useMemory := mc.useMemoryCache
	mc.lock.RUnlock()

	if !useMemory {
		if mc.redisCache == nil {
			mc.lock.Lock()
			mc.useMemoryCache = true
			mc.lock.Unlock()
			return mc.memoryCache.Expire(ctx, key, expiration)
		}
		// 尝试从Redis设置过期时间
		err := mc.redisCache.Expire(ctx, key, expiration)
		if err == nil {
			return nil
		}

		// Redis错误，切换到内存缓存
		mc.lock.Lock()
		mc.useMemoryCache = true
		mc.lock.Unlock()
	}

	// 从内存缓存设置过期时间
	return mc.memoryCache.Expire(ctx, key, expiration)
}
