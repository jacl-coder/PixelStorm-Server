package game

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jacl-coder/PixelStorm-Server/config"
	"github.com/jacl-coder/PixelStorm-Server/internal/models"
)

// GameServer 游戏服务器
type GameServer struct {
	config      *config.Config
	rooms       map[string]*Room
	roomsMutex  sync.RWMutex
	httpServer  *http.Server
	connections map[string]*PlayerConnection
	connMutex   sync.RWMutex

	// 关闭信号
	shutdown  chan struct{}
	isRunning bool
}

// PlayerConnection 玩家连接
type PlayerConnection struct {
	ID         string
	PlayerID   int64
	Room       *Room
	LastActive time.Time

	// 通信通道
	Send    chan []byte
	Receive chan []byte

	// 连接状态
	IsAlive bool
	conn    net.Conn
}

// NewGameServer 创建新的游戏服务器
func NewGameServer(cfg *config.Config) *GameServer {
	return &GameServer{
		config:      cfg,
		rooms:       make(map[string]*Room),
		connections: make(map[string]*PlayerConnection),
		shutdown:    make(chan struct{}),
	}
}

// Start 启动游戏服务器
func (s *GameServer) Start() error {
	if s.isRunning {
		return fmt.Errorf("服务器已经在运行")
	}

	// 初始化HTTP服务器
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Server.GamePort),
		Handler: s.createHandler(),
	}

	// 启动HTTP服务器
	go func() {
		log.Printf("游戏服务器启动，监听端口: %d", s.config.Server.GamePort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器错误: %v", err)
		}
	}()

	// 启动房间管理
	go s.roomManager()

	s.isRunning = true
	return nil
}

// Stop 停止游戏服务器
func (s *GameServer) Stop() error {
	if !s.isRunning {
		return nil
	}

	// 发送关闭信号
	close(s.shutdown)

	// 关闭所有房间
	s.roomsMutex.Lock()
	for _, room := range s.rooms {
		room.Stop()
	}
	s.roomsMutex.Unlock()

	// 关闭所有连接
	s.connMutex.Lock()
	for _, conn := range s.connections {
		close(conn.Send)
		if conn.conn != nil {
			conn.conn.Close()
		}
	}
	s.connMutex.Unlock()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP服务器关闭错误: %w", err)
	}

	s.isRunning = false
	log.Println("游戏服务器已停止")
	return nil
}

// createHandler 创建HTTP处理器
func (s *GameServer) createHandler() http.Handler {
	mux := http.NewServeMux()

	// WebSocket 连接端点
	mux.HandleFunc("/ws", s.handleWSConnection)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux
}

// roomManager 房间管理器
func (s *GameServer) roomManager() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupRooms()
		case <-s.shutdown:
			return
		}
	}
}

// cleanupRooms 清理空闲房间
func (s *GameServer) cleanupRooms() {
	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	for id, room := range s.rooms {
		if room.ShouldCleanup() {
			log.Printf("清理空闲房间: %s", id)
			room.Stop()
			delete(s.rooms, id)
		}
	}
}

// CreateRoom 创建游戏房间
func (s *GameServer) CreateRoom(name string, mode models.GameMode, maxPlayers int, mapID int) (*Room, error) {
	room := NewRoom(name, mode, maxPlayers, mapID)

	s.roomsMutex.Lock()
	defer s.roomsMutex.Unlock()

	s.rooms[room.ID] = room

	// 启动房间
	go room.Start()

	log.Printf("创建房间: %s, 模式: %s, 最大玩家数: %d", room.ID, mode, maxPlayers)
	return room, nil
}

// GetRoom 获取房间
func (s *GameServer) GetRoom(roomID string) (*Room, bool) {
	s.roomsMutex.RLock()
	defer s.roomsMutex.RUnlock()

	room, exists := s.rooms[roomID]
	return room, exists
}

// ListRooms 列出所有房间
func (s *GameServer) ListRooms() []*Room {
	s.roomsMutex.RLock()
	defer s.roomsMutex.RUnlock()

	rooms := make([]*Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}
