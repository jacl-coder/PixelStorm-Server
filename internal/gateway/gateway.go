package gateway

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/jacl-coder/PixelStorm-Server/config"
)

// ServiceType 服务类型
type ServiceType string

const (
	// ServiceGame 游戏服务
	ServiceGame ServiceType = "game"
	// ServiceMatch 匹配服务
	ServiceMatch ServiceType = "match"
	// ServiceAuth 认证服务
	ServiceAuth ServiceType = "auth"
)

// ServiceInstance 服务实例
type ServiceInstance struct {
	ID        string
	Type      ServiceType
	URL       *url.URL
	Health    bool
	LastCheck time.Time
}

// Gateway API网关
type Gateway struct {
	config     *config.Config
	services   map[ServiceType][]*ServiceInstance
	mutex      sync.RWMutex
	httpServer *http.Server
	isRunning  bool
	shutdown   chan struct{}
}

// NewGateway 创建新的网关
func NewGateway(cfg *config.Config) *Gateway {
	return &Gateway{
		config:   cfg,
		services: make(map[ServiceType][]*ServiceInstance),
		shutdown: make(chan struct{}),
	}
}

// Start 启动网关
func (g *Gateway) Start() error {
	if g.isRunning {
		return fmt.Errorf("网关已经在运行")
	}

	// 初始化HTTP服务器
	g.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.config.Server.GatewayPort),
		Handler: g.createHandler(),
	}

	// 注册内部服务
	g.registerInternalServices()

	// 启动健康检查
	go g.healthCheck()

	// 启动HTTP服务器
	go func() {
		log.Printf("API网关启动，监听端口: %d", g.config.Server.GatewayPort)
		if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器错误: %v", err)
		}
	}()

	g.isRunning = true
	return nil
}

// Stop 停止网关
func (g *Gateway) Stop() error {
	if !g.isRunning {
		return nil
	}

	close(g.shutdown)
	g.isRunning = false
	log.Println("API网关已停止")
	return nil
}

// RegisterService 注册服务
func (g *Gateway) RegisterService(serviceType ServiceType, serviceURL string) error {
	parsedURL, err := url.Parse(serviceURL)
	if err != nil {
		return fmt.Errorf("无效的服务URL: %w", err)
	}

	instance := &ServiceInstance{
		ID:        fmt.Sprintf("%s-%d", serviceType, time.Now().UnixNano()),
		Type:      serviceType,
		URL:       parsedURL,
		Health:    true,
		LastCheck: time.Now(),
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, ok := g.services[serviceType]; !ok {
		g.services[serviceType] = make([]*ServiceInstance, 0)
	}
	g.services[serviceType] = append(g.services[serviceType], instance)
	log.Printf("注册服务: %s, URL: %s", serviceType, serviceURL)

	return nil
}

// UnregisterService 注销服务
func (g *Gateway) UnregisterService(serviceType ServiceType, serviceID string) bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	instances, ok := g.services[serviceType]
	if !ok {
		return false
	}

	for i, instance := range instances {
		if instance.ID == serviceID {
			g.services[serviceType] = append(instances[:i], instances[i+1:]...)
			log.Printf("注销服务: %s, ID: %s", serviceType, serviceID)
			return true
		}
	}

	return false
}

// createHandler 创建HTTP处理器
func (g *Gateway) createHandler() http.Handler {
	mux := http.NewServeMux()

	// 创建各种处理器
	authHandler := NewAuthHandler()
	characterHandler := NewCharacterHandler()
	profileHandler := NewProfileHandler()
	statsHandler := NewStatsHandler()

	// 注册认证相关路由
	authHandler.RegisterHandlers(mux)

	// 注册角色相关路由
	characterHandler.RegisterHandlers(mux)

	// 注册玩家资料相关路由
	profileHandler.RegisterHandlers(mux)

	// 注册战绩相关路由
	statsHandler.RegisterHandlers(mux)

	// 其他服务的API路由（转发到对应服务）
	mux.HandleFunc("/game/", g.handleGameRequest)
	mux.HandleFunc("/match/", g.handleMatchRequest)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 服务发现端点
	mux.HandleFunc("/services", g.handleServiceDiscovery)

	// 应用中间件
	handler := g.applyMiddleware(mux)

	return handler
}

// applyMiddleware 应用中间件
func (g *Gateway) applyMiddleware(handler http.Handler) http.Handler {
	// 创建中间件
	loggingMiddleware := NewLoggingMiddleware()
	securityMiddleware := NewSecurityMiddleware()
	corsMiddleware := NewCORSMiddleware()
	rateLimiter := NewRateLimiter(60, 10) // 每分钟60次请求，突发10次
	cacheMiddleware := NewCacheMiddleware()

	// 按顺序应用中间件（从外到内）
	handler = loggingMiddleware.Middleware(handler)
	handler = securityMiddleware.Middleware(handler)
	handler = corsMiddleware.Middleware(handler)
	handler = rateLimiter.Middleware(handler)
	handler = cacheMiddleware.Middleware(handler)

	return handler
}

// handleGameRequest 处理游戏服务请求
func (g *Gateway) handleGameRequest(w http.ResponseWriter, r *http.Request) {
	g.forwardRequest(w, r, ServiceGame)
}

// handleMatchRequest 处理匹配服务请求
func (g *Gateway) handleMatchRequest(w http.ResponseWriter, r *http.Request) {
	g.forwardRequest(w, r, ServiceMatch)
}

// forwardRequest 转发请求到指定服务
func (g *Gateway) forwardRequest(w http.ResponseWriter, r *http.Request, serviceType ServiceType) {

	// 验证认证
	if !g.validateAuth(r) && serviceType != ServiceAuth {
		http.Error(w, "未授权", http.StatusUnauthorized)
		return
	}

	// 获取服务实例
	instance := g.getServiceInstance(serviceType)
	if instance == nil {
		http.Error(w, "服务不可用", http.StatusServiceUnavailable)
		return
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(instance.URL)

	// 修改请求
	r.URL.Host = instance.URL.Host
	r.URL.Scheme = instance.URL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Origin-Host", instance.URL.Host)
	r.Host = instance.URL.Host

	// 转发请求
	proxy.ServeHTTP(w, r)
}

// handleServiceDiscovery 处理服务发现请求
func (g *Gateway) handleServiceDiscovery(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现服务发现API
	http.Error(w, "未实现", http.StatusNotImplemented)
}

// validateAuth 验证认证
func (g *Gateway) validateAuth(r *http.Request) bool {
	// 获取认证令牌
	token := r.Header.Get("Authorization")
	if token == "" {
		// 尝试从查询参数获取
		token = r.URL.Query().Get("token")
		if token == "" {
			return false
		}
	}

	// TODO: 实现真正的令牌验证
	// 这里简单地检查令牌是否存在
	return token != ""
}

// getServiceInstance 获取服务实例
func (g *Gateway) getServiceInstance(serviceType ServiceType) *ServiceInstance {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	instances, ok := g.services[serviceType]
	if !ok || len(instances) == 0 {
		return nil
	}

	// 简单的负载均衡：轮询
	// 在实际应用中，可能需要更复杂的负载均衡策略
	// 例如考虑服务器负载、响应时间等
	var healthyInstances []*ServiceInstance
	for _, instance := range instances {
		if instance.Health {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil
	}

	// 使用时间戳作为简单的轮询机制
	index := time.Now().UnixNano() % int64(len(healthyInstances))
	return healthyInstances[index]
}

// registerInternalServices 注册内部服务
func (g *Gateway) registerInternalServices() {
	// 注册游戏服务
	gameURL := fmt.Sprintf("http://localhost:%d", g.config.Server.GamePort)
	if err := g.RegisterService(ServiceGame, gameURL); err != nil {
		log.Printf("注册服务失败: %v", err)
	}
	log.Printf("注册服务: %s, URL: %s", ServiceGame, gameURL)

	// 注册匹配服务
	matchURL := fmt.Sprintf("http://localhost:%d", g.config.Server.MatchPort)
	if err := g.RegisterService(ServiceMatch, matchURL); err != nil {
		log.Printf("注册服务失败: %v", err)
	}
	log.Printf("注册服务: %s, URL: %s", ServiceMatch, matchURL)

	// 注册认证服务 (内部实现)
	authURL := fmt.Sprintf("http://localhost:%d", g.config.Server.GatewayPort)
	if err := g.RegisterService(ServiceAuth, authURL); err != nil {
		log.Printf("注册服务失败: %v", err)
	}
	log.Printf("注册服务: %s, URL: %s", ServiceAuth, authURL)
}

// healthCheck 健康检查
func (g *Gateway) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.checkServicesHealth()
		case <-g.shutdown:
			return
		}
	}
}

// checkServicesHealth 检查服务健康状态
func (g *Gateway) checkServicesHealth() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for serviceType, instances := range g.services {
		for _, instance := range instances {
			// 发送健康检查请求
			healthURL := *instance.URL
			healthURL.Path = "/health"

			client := http.Client{
				Timeout: 2 * time.Second,
			}

			resp, err := client.Get(healthURL.String())

			// 更新健康状态
			instance.LastCheck = time.Now()
			if err != nil || resp.StatusCode != http.StatusOK {
				if instance.Health {
					log.Printf("服务不健康: %s, ID: %s", serviceType, instance.ID)
					instance.Health = false
				}
			} else {
				if !instance.Health {
					log.Printf("服务恢复健康: %s, ID: %s", serviceType, instance.ID)
					instance.Health = true
				}
				resp.Body.Close()
			}
		}
	}
}
