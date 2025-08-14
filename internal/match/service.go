// service.go

package match

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/jacl-coder/PixelStorm-Server/config"
	"github.com/jacl-coder/PixelStorm-Server/internal/game"
	"github.com/jacl-coder/PixelStorm-Server/internal/models"
)

// MatchRequest 匹配请求
type MatchRequest struct {
	PlayerID    int64
	CharacterID int
	GameMode    models.GameMode
	Timestamp   time.Time
	SessionID   string
}

// MatchService 匹配服务
type MatchService struct {
	// 匹配队列，按游戏模式分类
	queues      map[models.GameMode][]*MatchRequest
	queuesMutex sync.RWMutex

	// 游戏服务器引用
	gameServer *game.GameServer

	// 匹配配置
	config *config.Config

	// HTTP服务器
	httpServer *http.Server
	handler    *MatchHandler

	// 控制通道
	shutdown  chan struct{}
	isRunning bool
}

// NewMatchService 创建匹配服务
func NewMatchService(cfg *config.Config, gameServer *game.GameServer) *MatchService {
	service := &MatchService{
		queues:     make(map[models.GameMode][]*MatchRequest),
		gameServer: gameServer,
		config:     cfg,
		shutdown:   make(chan struct{}),
	}

	// 创建处理器
	service.handler = NewMatchHandler(service)

	return service
}

// Start 启动匹配服务
func (s *MatchService) Start() error {
	if s.isRunning {
		return fmt.Errorf("匹配服务已经在运行")
	}

	log.Println("匹配服务启动")
	s.isRunning = true

	// 创建HTTP服务器
	mux := http.NewServeMux()
	s.handler.RegisterHandlers(mux)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Server.MatchPort),
		Handler: mux,
	}

	// 启动HTTP服务器
	go func() {
		log.Printf("匹配服务HTTP服务器启动，监听端口: %d", s.config.Server.MatchPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("匹配服务HTTP服务器错误: %v", err)
		}
	}()

	// 启动匹配循环
	go s.matchLoop()

	return nil
}

// Stop 停止匹配服务
func (s *MatchService) Stop() {
	if !s.isRunning {
		return
	}

	close(s.shutdown)
	s.isRunning = false

	// 关闭HTTP服务器
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}

	log.Println("匹配服务已停止")
}

// AddToQueue 添加玩家到匹配队列
func (s *MatchService) AddToQueue(playerID int64, characterID int, gameMode models.GameMode, sessionID string) {
	s.queuesMutex.Lock()
	defer s.queuesMutex.Unlock()

	// 创建匹配请求
	request := &MatchRequest{
		PlayerID:    playerID,
		CharacterID: characterID,
		GameMode:    gameMode,
		Timestamp:   time.Now(),
		SessionID:   sessionID,
	}

	// 检查该模式的队列是否存在
	if _, ok := s.queues[gameMode]; !ok {
		s.queues[gameMode] = make([]*MatchRequest, 0)
	}

	// 添加到队列
	s.queues[gameMode] = append(s.queues[gameMode], request)
	log.Printf("玩家 %d 加入 %s 模式的匹配队列", playerID, gameMode)
}

// RemoveFromQueue 从匹配队列移除玩家
func (s *MatchService) RemoveFromQueue(playerID int64, gameMode models.GameMode) bool {
	s.queuesMutex.Lock()
	defer s.queuesMutex.Unlock()

	// 检查该模式的队列是否存在
	queue, ok := s.queues[gameMode]
	if !ok {
		return false
	}

	// 查找并移除玩家
	for i, req := range queue {
		if req.PlayerID == playerID {
			// 移除该玩家
			s.queues[gameMode] = append(queue[:i], queue[i+1:]...)
			log.Printf("玩家 %d 离开 %s 模式的匹配队列", playerID, gameMode)
			return true
		}
	}

	return false
}

// GetQueueLength 获取队列长度
func (s *MatchService) GetQueueLength(gameMode models.GameMode) int {
	s.queuesMutex.RLock()
	defer s.queuesMutex.RUnlock()

	if queue, ok := s.queues[gameMode]; ok {
		return len(queue)
	}
	return 0
}

// GetAllQueueLengths 获取所有队列长度
func (s *MatchService) GetAllQueueLengths() map[models.GameMode]int {
	s.queuesMutex.RLock()
	defer s.queuesMutex.RUnlock()

	result := make(map[models.GameMode]int)
	for mode, queue := range s.queues {
		result[mode] = len(queue)
	}
	return result
}

// matchLoop 匹配循环
func (s *MatchService) matchLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.processMatching()
		case <-s.shutdown:
			return
		}
	}
}

// processMatching 处理匹配
func (s *MatchService) processMatching() {
	s.queuesMutex.Lock()
	defer s.queuesMutex.Unlock()

	// 为每种游戏模式进行匹配
	for mode, queue := range s.queues {
		// 根据游戏模式获取需要的玩家数量
		playersNeeded := getPlayersNeededForMode(mode)

		// 如果队列中的玩家不足，跳过
		if len(queue) < playersNeeded {
			continue
		}

		// 按照加入时间排序（先进先出）
		// 这里使用简单的时间排序，实际可能需要更复杂的匹配算法
		// 例如考虑玩家等级、技能水平等

		// 创建房间
		roomName := fmt.Sprintf("%s-%s", mode, time.Now().Format("150405"))
		room, err := s.gameServer.CreateRoom(roomName, mode, playersNeeded, 1) // 使用默认地图ID 1
		if err != nil {
			log.Printf("创建房间失败: %v", err)
			continue
		}

		// 将前N个玩家加入房间
		matchedPlayers := queue[:playersNeeded]
		s.queues[mode] = queue[playersNeeded:] // 更新队列

		// 通知这些玩家已匹配成功
		for _, player := range matchedPlayers {
			// 在实际实现中，这里会通过WebSocket通知玩家
			// 并提供房间信息让玩家加入
			log.Printf("玩家 %d 匹配成功，房间ID: %s", player.PlayerID, room.ID)

			// TODO: 通过会话ID找到玩家连接，并发送匹配成功消息
		}
	}
}

// getPlayersNeededForMode 根据游戏模式获取需要的玩家数量
func getPlayersNeededForMode(mode models.GameMode) int {
	switch mode {
	case models.DeathMatch:
		return 4 // 死亡竞赛需要4人
	case models.TeamDeathMatch:
		return 6 // 团队死亡竞赛需要6人（3v3）
	case models.CapturePoint:
		return 8 // 据点占领需要8人（4v4）
	case models.FlagCapture:
		return 6 // 夺旗模式需要6人（3v3）
	default:
		return 4 // 默认需要4人
	}
}
