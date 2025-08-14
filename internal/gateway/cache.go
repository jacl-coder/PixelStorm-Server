package gateway

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      []byte
	Headers   map[string]string
	ExpiresAt time.Time
	ETag      string
}

// MemoryCache 内存缓存
type MemoryCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	
	// 配置
	DefaultTTL      time.Duration
	MaxEntries      int
	CleanupInterval time.Duration
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		entries:         make(map[string]*CacheEntry),
		DefaultTTL:      5 * time.Minute,
		MaxEntries:      1000,
		CleanupInterval: 1 * time.Minute,
	}
	
	// 启动清理协程
	go cache.cleanup()
	
	return cache
}

// CacheMiddleware 缓存中间件
type CacheMiddleware struct {
	cache *MemoryCache
	
	// 可缓存的路径模式
	CacheablePaths []string
	// 缓存时间配置
	CacheTTL map[string]time.Duration
}

// NewCacheMiddleware 创建缓存中间件
func NewCacheMiddleware() *CacheMiddleware {
	return &CacheMiddleware{
		cache: NewMemoryCache(),
		CacheablePaths: []string{
			"/characters",
			"/stats/leaderboard",
			"/players/characters/",
			"/players/default-character/",
		},
		CacheTTL: map[string]time.Duration{
			"/characters":        10 * time.Minute, // 角色信息缓存10分钟
			"/stats/leaderboard": 2 * time.Minute,  // 排行榜缓存2分钟
			"/players/":          1 * time.Minute,  // 玩家信息缓存1分钟
		},
	}
}

// Middleware 缓存中间件
func (cm *CacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 只缓存GET请求
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		
		// 检查是否应该缓存
		if !cm.shouldCache(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		
		// 生成缓存键
		cacheKey := cm.generateCacheKey(r)
		
		// 检查缓存
		if entry := cm.cache.Get(cacheKey); entry != nil {
			// 检查ETag
			if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch != "" {
				if ifNoneMatch == entry.ETag {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			
			// 返回缓存的响应
			cm.writeCachedResponse(w, entry)
			return
		}
		
		// 创建响应捕获器
		recorder := &cacheResponseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			headers:        make(map[string]string),
			body:          make([]byte, 0),
		}
		
		// 处理请求
		next.ServeHTTP(recorder, r)
		
		// 如果响应成功，缓存结果
		if recorder.statusCode == http.StatusOK && len(recorder.body) > 0 {
			ttl := cm.getTTL(r.URL.Path)
			etag := cm.generateETag(recorder.body)
			
			entry := &CacheEntry{
				Data:      recorder.body,
				Headers:   recorder.headers,
				ExpiresAt: time.Now().Add(ttl),
				ETag:      etag,
			}
			
			cm.cache.Set(cacheKey, entry)
			
			// 设置ETag头
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(ttl.Seconds())))
		}
	})
}

// shouldCache 检查是否应该缓存
func (cm *CacheMiddleware) shouldCache(path string) bool {
	for _, pattern := range cm.CacheablePaths {
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}
	return false
}

// generateCacheKey 生成缓存键
func (cm *CacheMiddleware) generateCacheKey(r *http.Request) string {
	// 使用路径和查询参数生成键
	key := r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	return key
}

// getTTL 获取缓存时间
func (cm *CacheMiddleware) getTTL(path string) time.Duration {
	for pattern, ttl := range cm.CacheTTL {
		if strings.HasPrefix(path, pattern) {
			return ttl
		}
	}
	return cm.cache.DefaultTTL
}

// generateETag 生成ETag
func (cm *CacheMiddleware) generateETag(data []byte) string {
	hash := md5.Sum(data)
	return fmt.Sprintf(`"%x"`, hash)
}

// writeCachedResponse 写入缓存的响应
func (cm *CacheMiddleware) writeCachedResponse(w http.ResponseWriter, entry *CacheEntry) {
	// 设置头部
	for key, value := range entry.Headers {
		w.Header().Set(key, value)
	}
	
	// 设置缓存相关头部
	w.Header().Set("ETag", entry.ETag)
	w.Header().Set("X-Cache", "HIT")
	
	// 写入响应体
	w.WriteHeader(http.StatusOK)
	w.Write(entry.Data)
}

// Get 获取缓存条目
func (mc *MemoryCache) Get(key string) *CacheEntry {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	entry, exists := mc.entries[key]
	if !exists {
		return nil
	}
	
	// 检查是否过期
	if time.Now().After(entry.ExpiresAt) {
		// 异步删除过期条目
		go func() {
			mc.mutex.Lock()
			delete(mc.entries, key)
			mc.mutex.Unlock()
		}()
		return nil
	}
	
	return entry
}

// Set 设置缓存条目
func (mc *MemoryCache) Set(key string, entry *CacheEntry) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// 检查是否超过最大条目数
	if len(mc.entries) >= mc.MaxEntries {
		// 删除一些过期条目
		mc.evictExpired()
		
		// 如果还是太多，删除最旧的条目
		if len(mc.entries) >= mc.MaxEntries {
			mc.evictOldest()
		}
	}
	
	mc.entries[key] = entry
}

// evictExpired 删除过期条目
func (mc *MemoryCache) evictExpired() {
	now := time.Now()
	for key, entry := range mc.entries {
		if now.After(entry.ExpiresAt) {
			delete(mc.entries, key)
		}
	}
}

// evictOldest 删除最旧的条目
func (mc *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range mc.entries {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}
	
	if oldestKey != "" {
		delete(mc.entries, oldestKey)
	}
}

// cleanup 清理过期条目
func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(mc.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mutex.Lock()
		mc.evictExpired()
		mc.mutex.Unlock()
	}
}

// cacheResponseRecorder 缓存响应记录器
type cacheResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	headers    map[string]string
	body       []byte
}

// WriteHeader 记录状态码
func (crr *cacheResponseRecorder) WriteHeader(code int) {
	crr.statusCode = code
	crr.ResponseWriter.WriteHeader(code)
}

// Write 记录响应体
func (crr *cacheResponseRecorder) Write(data []byte) (int, error) {
	// 记录响应体
	crr.body = append(crr.body, data...)
	
	// 记录重要的头部
	if contentType := crr.ResponseWriter.Header().Get("Content-Type"); contentType != "" {
		crr.headers["Content-Type"] = contentType
	}
	
	// 写入实际响应
	return crr.ResponseWriter.Write(data)
}

// Header 获取头部
func (crr *cacheResponseRecorder) Header() http.Header {
	return crr.ResponseWriter.Header()
}
