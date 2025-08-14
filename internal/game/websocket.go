// websocket.go

package game

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// 写入超时时间
	writeWait = 10 * time.Second

	// 读取超时时间
	pongWait = 60 * time.Second

	// 发送 ping 的间隔时间
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Message 消息结构
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// handleWSConnection 处理WebSocket连接
func (s *GameServer) handleWSConnection(w http.ResponseWriter, r *http.Request) {
	// 获取认证信息
	playerID := r.URL.Query().Get("player_id")
	token := r.URL.Query().Get("token")

	// 验证认证信息
	// TODO: 实现真正的认证逻辑
	if playerID == "" || token == "" {
		http.Error(w, "未授权", http.StatusUnauthorized)
		return
	}

	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 创建玩家连接
	playerConn := &PlayerConnection{
		ID:         uuid.New().String(),
		PlayerID:   parseInt64(playerID),
		LastActive: time.Now(),
		Send:       make(chan []byte, 256),
		Receive:    make(chan []byte, 256),
		IsAlive:    true,
	}

	// 添加到连接列表
	s.connMutex.Lock()
	s.connections[playerConn.ID] = playerConn
	s.connMutex.Unlock()

	log.Printf("玩家 %s 已连接", playerID)

	// 启动读写协程
	go s.readPump(conn, playerConn)
	go s.writePump(conn, playerConn)
}

// readPump 从WebSocket读取数据
func (s *GameServer) readPump(conn *websocket.Conn, player *PlayerConnection) {
	defer func() {
		s.closeConnection(player)
		conn.Close()
	}()

	// 设置读取参数
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		player.LastActive = time.Now()

		// 处理接收到的消息
		s.handleMessage(player, message)
	}
}

// writePump 向WebSocket写入数据
func (s *GameServer) writePump(conn *websocket.Conn, player *PlayerConnection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-player.Send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道已关闭
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加队列中的其他消息
			n := len(player.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-player.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// closeConnection 关闭玩家连接
func (s *GameServer) closeConnection(player *PlayerConnection) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// 检查连接是否已关闭
	if _, ok := s.connections[player.ID]; !ok {
		return
	}

	// 如果玩家在房间中，从房间移除
	if player.Room != nil {
		player.Room.RemovePlayer(player.ID)
		player.Room = nil
	}

	// 关闭发送通道
	close(player.Send)

	// 从连接列表移除
	delete(s.connections, player.ID)

	log.Printf("玩家 %d 已断开连接", player.PlayerID)
}

// handleMessage 处理接收到的消息
func (s *GameServer) handleMessage(player *PlayerConnection, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("解析消息失败: %v", err)
		return
	}

	switch msg.Type {
	case "join_room":
		s.handleJoinRoom(player, msg.Payload)
	case "create_room":
		s.handleCreateRoom(player, msg.Payload)
	case "leave_room":
		s.handleLeaveRoom(player)
	case "ready":
		s.handlePlayerReady(player, true)
	case "unready":
		s.handlePlayerReady(player, false)
	case "player_input":
		s.handlePlayerInput(player, msg.Payload)
	default:
		log.Printf("未知消息类型: %s", msg.Type)
	}
}

// handleJoinRoom 处理加入房间请求
func (s *GameServer) handleJoinRoom(player *PlayerConnection, payload json.RawMessage) {
	// TODO: 实现加入房间逻辑
}

// handleCreateRoom 处理创建房间请求
func (s *GameServer) handleCreateRoom(player *PlayerConnection, payload json.RawMessage) {
	// TODO: 实现创建房间逻辑
}

// handleLeaveRoom 处理离开房间请求
func (s *GameServer) handleLeaveRoom(player *PlayerConnection) {
	if player.Room != nil {
		player.Room.RemovePlayer(player.ID)
		player.Room = nil

		// 发送离开房间确认
		s.sendMessage(player, Message{
			Type: "leave_room_confirm",
		})
	}
}

// handlePlayerReady 处理玩家准备/取消准备
func (s *GameServer) handlePlayerReady(player *PlayerConnection, ready bool) {
	// TODO: 实现玩家准备逻辑
}

// handlePlayerInput 处理玩家输入
func (s *GameServer) handlePlayerInput(player *PlayerConnection, payload json.RawMessage) {
	// TODO: 实现玩家输入处理逻辑
}

// sendMessage 向玩家发送消息
func (s *GameServer) sendMessage(player *PlayerConnection, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	select {
	case player.Send <- data:
		// 消息已发送到通道
	default:
		// 通道已满，关闭连接
		s.closeConnection(player)
	}
}

// broadcastMessage 向所有玩家广播消息
func (s *GameServer) broadcastMessage(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return
	}

	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	for _, player := range s.connections {
		select {
		case player.Send <- data:
			// 消息已发送到通道
		default:
			// 通道已满，关闭连接
			go s.closeConnection(player)
		}
	}
}

// 辅助函数

// parseInt64 将字符串转换为int64
func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
