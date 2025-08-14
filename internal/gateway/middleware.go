package gateway

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter 请求频率限制器
type RateLimiter struct {
	clients map[string]*ClientInfo
	mutex   sync.RWMutex
	
	// 配置
	RequestsPerMinute int
	BurstSize         int
	CleanupInterval   time.Duration
}

// ClientInfo 客户端信息
type ClientInfo struct {
	Requests  []time.Time
	LastSeen  time.Time
}

// NewRateLimiter 创建新的频率限制器
func NewRateLimiter(requestsPerMinute, burstSize int) *RateLimiter {
	rl := &RateLimiter{
		clients:           make(map[string]*ClientInfo),
		RequestsPerMinute: requestsPerMinute,
		BurstSize:         burstSize,
		CleanupInterval:   5 * time.Minute,
	}
	
	// 启动清理协程
	go rl.cleanup()
	
	return rl
}

// Middleware 频率限制中间件
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端IP
		clientIP := rl.getClientIP(r)
		
		// 检查频率限制
		if !rl.allowRequest(clientIP) {
			rl.sendRateLimitError(w)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// allowRequest 检查是否允许请求
func (rl *RateLimiter) allowRequest(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// 获取或创建客户端信息
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &ClientInfo{
			Requests: make([]time.Time, 0),
			LastSeen: now,
		}
		rl.clients[clientIP] = client
	}
	
	// 更新最后访问时间
	client.LastSeen = now
	
	// 清理过期的请求记录
	cutoff := now.Add(-time.Minute)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range client.Requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.Requests = validRequests
	
	// 检查是否超过限制
	if len(client.Requests) >= rl.RequestsPerMinute {
		return false
	}
	
	// 记录当前请求
	client.Requests = append(client.Requests, now)
	
	return true
}

// getClientIP 获取客户端IP
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}
	
	// 检查X-Real-IP头
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// 使用RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	
	return ip
}

// sendRateLimitError 发送频率限制错误响应
func (rl *RateLimiter) sendRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	
	response := map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("请求过于频繁，每分钟最多允许 %d 次请求", rl.RequestsPerMinute),
		"code":    "RATE_LIMIT_EXCEEDED",
	}
	
	json.NewEncoder(w).Encode(response)
}

// cleanup 清理过期的客户端信息
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		
		cutoff := time.Now().Add(-10 * time.Minute) // 10分钟未访问的客户端
		for ip, client := range rl.clients {
			if client.LastSeen.Before(cutoff) {
				delete(rl.clients, ip)
			}
		}
		
		rl.mutex.Unlock()
	}
}

// SecurityMiddleware 安全头中间件
type SecurityMiddleware struct{}

// NewSecurityMiddleware 创建安全中间件
func NewSecurityMiddleware() *SecurityMiddleware {
	return &SecurityMiddleware{}
}

// Middleware 安全头中间件
func (sm *SecurityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置安全头
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// 移除服务器信息
		w.Header().Set("Server", "PixelStorm")
		
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware() *CORSMiddleware {
	return &CORSMiddleware{
		AllowedOrigins: []string{"*"}, // 生产环境应该限制具体域名
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
	}
}

// Middleware CORS中间件
func (cm *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*") // 生产环境应该更严格
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware 日志中间件
type LoggingMiddleware struct{}

// NewLoggingMiddleware 创建日志中间件
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{}
}

// Middleware 日志中间件
func (lm *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 创建响应记录器
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// 处理请求
		next.ServeHTTP(recorder, r)
		
		// 记录日志
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s %d %v\n",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			recorder.statusCode,
			duration,
		)
	})
}

// responseRecorder 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 记录状态码
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
