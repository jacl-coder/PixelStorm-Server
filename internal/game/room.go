package game

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jacl-coder/PixelStorm-Server/internal/models"
)

// Room 游戏房间
type Room struct {
	ID         string
	Name       string
	Mode       models.GameMode
	Status     models.RoomStatus
	MaxPlayers int
	CreatedAt  time.Time
	StartedAt  time.Time
	EndedAt    time.Time
	MapID      int

	// 房间设置
	TimeLimit    int  // 时间限制(秒)
	ScoreLimit   int  // 分数限制
	FriendlyFire bool // 友军伤害
	PrivateRoom  bool // 私人房间
	Password     string

	// 玩家管理
	players     map[string]*PlayerState
	playerMutex sync.RWMutex

	// 游戏状态
	entities      map[string]models.Entity
	entityMutex   sync.RWMutex
	frameID       int64
	lastFrameTime time.Time
	scores        map[int64]int // 玩家ID -> 分数

	// 控制通道
	shutdown     chan struct{}
	isRunning    bool
	lastActivity time.Time
}

// PlayerState 玩家游戏状态
type PlayerState struct {
	Connection *PlayerConnection
	Entity     *models.PlayerEntity
	Ready      bool
	LastInput  time.Time
}

// NewRoom 创建新房间
func NewRoom(name string, mode models.GameMode, maxPlayers int, mapID int) *Room {
	roomID := uuid.New().String()
	now := time.Now()

	return &Room{
		ID:           roomID,
		Name:         name,
		Mode:         mode,
		Status:       models.RoomWaiting,
		MaxPlayers:   maxPlayers,
		CreatedAt:    now,
		MapID:        mapID,
		TimeLimit:    300, // 默认5分钟
		ScoreLimit:   20,  // 默认20分
		FriendlyFire: false,
		players:      make(map[string]*PlayerState),
		entities:     make(map[string]models.Entity),
		scores:       make(map[int64]int),
		shutdown:     make(chan struct{}),
		lastActivity: now,
	}
}

// Start 启动房间
func (r *Room) Start() error {
	if r.isRunning {
		return fmt.Errorf("房间已经在运行")
	}

	log.Printf("房间 %s 启动", r.ID)
	r.isRunning = true
	r.lastActivity = time.Now()

	// 游戏循环
	go r.gameLoop()

	return nil
}

// Stop 停止房间
func (r *Room) Stop() {
	if !r.isRunning {
		return
	}

	close(r.shutdown)
	r.isRunning = false
	r.Status = models.RoomEnded
	r.EndedAt = time.Now()

	log.Printf("房间 %s 已停止", r.ID)
}

// AddPlayer 添加玩家到房间
func (r *Room) AddPlayer(conn *PlayerConnection, characterID int) error {
	r.playerMutex.Lock()
	defer r.playerMutex.Unlock()

	if len(r.players) >= r.MaxPlayers {
		return fmt.Errorf("房间已满")
	}

	if r.Status != models.RoomWaiting {
		return fmt.Errorf("游戏已经开始，无法加入")
	}

	// 创建玩家实体
	playerEntity := &models.PlayerEntity{
		BaseEntity: models.BaseEntity{
			ID:        uuid.New().String(),
			Type:      models.EntityPlayer,
			Position:  getRandomSpawnPosition(),
			Rotation:  0,
			Velocity:  models.Vector2D{X: 0, Y: 0},
			CreatedAt: time.Now(),
		},
		PlayerID:       conn.PlayerID,
		CharacterID:    characterID,
		Team:           assignTeam(r),
		Health:         100,
		MaxHealth:      100,
		IsAlive:        true,
		SkillCooldowns: make(map[int]float64),
	}

	// 添加到房间
	playerState := &PlayerState{
		Connection: conn,
		Entity:     playerEntity,
		Ready:      false,
		LastInput:  time.Now(),
	}

	r.players[conn.ID] = playerState

	// 添加到实体列表
	r.entityMutex.Lock()
	r.entities[playerEntity.ID] = playerEntity
	r.entityMutex.Unlock()

	r.lastActivity = time.Now()
	log.Printf("玩家 %d 加入房间 %s", conn.PlayerID, r.ID)

	return nil
}

// RemovePlayer 从房间移除玩家
func (r *Room) RemovePlayer(connID string) {
	r.playerMutex.Lock()
	defer r.playerMutex.Unlock()

	player, exists := r.players[connID]
	if !exists {
		return
	}

	// 从实体列表移除
	if player.Entity != nil {
		r.entityMutex.Lock()
		delete(r.entities, player.Entity.ID)
		r.entityMutex.Unlock()
	}

	delete(r.players, connID)
	r.lastActivity = time.Now()

	log.Printf("玩家已离开房间 %s", r.ID)

	// 如果房间为空，可以标记为可清理
	if len(r.players) == 0 && r.Status != models.RoomEnded {
		log.Printf("房间 %s 已空，等待清理", r.ID)
	}
}

// GetPlayerCount 获取玩家数量
func (r *Room) GetPlayerCount() int {
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()
	return len(r.players)
}

// IsEmpty 检查房间是否为空
func (r *Room) IsEmpty() bool {
	return r.GetPlayerCount() == 0
}

// ShouldCleanup 检查房间是否应该被清理
func (r *Room) ShouldCleanup() bool {
	// 如果房间为空且超过5分钟没有活动，则可以清理
	if r.IsEmpty() {
		return time.Since(r.lastActivity) > 5*time.Minute
	}

	// 如果游戏已结束且超过2分钟，则可以清理
	if r.Status == models.RoomEnded {
		return time.Since(r.EndedAt) > 2*time.Minute
	}

	return false
}

// gameLoop 游戏主循环
func (r *Room) gameLoop() {
	ticker := time.NewTicker(16 * time.Millisecond) // 约60FPS
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.Status == models.RoomPlaying {
				r.update()
			} else if r.Status == models.RoomWaiting {
				r.checkGameStart()
			}
		case <-r.shutdown:
			return
		}
	}
}

// update 更新游戏状态
func (r *Room) update() {
	now := time.Now()
	deltaTime := now.Sub(r.lastFrameTime).Seconds()
	r.lastFrameTime = now
	r.frameID++

	// 更新实体
	r.updateEntities(deltaTime)

	// 检测碰撞
	r.detectCollisions()

	// 检查游戏结束条件
	r.checkGameEnd()

	// 发送游戏状态
	r.broadcastGameState()
}

// updateEntities 更新所有实体
func (r *Room) updateEntities(deltaTime float64) {
	r.entityMutex.Lock()
	defer r.entityMutex.Unlock()

	// 更新所有实体位置
	for id, entity := range r.entities {
		// 根据实体类型进行不同的更新逻辑
		switch e := entity.(type) {
		case *models.PlayerEntity:
			// 玩家实体更新
			if e.IsAlive {
				// 更新位置
				pos := e.GetPosition()
				vel := e.GetVelocity()
				pos.X += vel.X * deltaTime
				pos.Y += vel.Y * deltaTime
				e.Position = pos

				// 更新技能冷却
				for skillID, cooldown := range e.SkillCooldowns {
					if cooldown > 0 {
						e.SkillCooldowns[skillID] = cooldown - deltaTime
						if e.SkillCooldowns[skillID] <= 0 {
							delete(e.SkillCooldowns, skillID)
						}
					}
				}
			} else {
				// 处理重生逻辑
				e.RespawnTime -= int(deltaTime)
				if e.RespawnTime <= 0 {
					e.IsAlive = true
					e.Health = e.MaxHealth
					e.Position = getRandomSpawnPosition()
					e.Velocity = models.Vector2D{X: 0, Y: 0}
				}
			}
		case *models.ProjectileEntity:
			// 投射物实体更新
			pos := e.GetPosition()
			vel := e.GetVelocity()
			pos.X += vel.X * deltaTime
			pos.Y += vel.Y * deltaTime
			e.Position = pos

			// 检查生命周期
			e.LifeTime -= deltaTime
			if e.LifeTime <= 0 {
				delete(r.entities, id)
			}
		}
	}
}

// checkGameStart 检查游戏是否可以开始
func (r *Room) checkGameStart() {
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	// 检查是否有足够的玩家
	if len(r.players) < 2 {
		return
	}

	// 检查所有玩家是否准备就绪
	allReady := true
	for _, player := range r.players {
		if !player.Ready {
			allReady = false
			break
		}
	}

	if allReady {
		r.startGame()
	}
}

// startGame 开始游戏
func (r *Room) startGame() {
	r.Status = models.RoomPlaying
	r.StartedAt = time.Now()
	r.lastFrameTime = time.Now()
	r.frameID = 0

	log.Printf("房间 %s 游戏开始", r.ID)

	// 通知所有玩家游戏开始
	r.broadcastGameStart()
}

// checkGameEnd 检查游戏是否结束
func (r *Room) checkGameEnd() {
	// 检查时间限制
	if time.Since(r.StartedAt).Seconds() >= float64(r.TimeLimit) {
		r.endGame()
		return
	}

	// 检查分数限制
	for _, score := range r.scores {
		if score >= r.ScoreLimit {
			r.endGame()
			return
		}
	}
}

// endGame 结束游戏
func (r *Room) endGame() {
	r.Status = models.RoomEnded
	r.EndedAt = time.Now()

	log.Printf("房间 %s 游戏结束", r.ID)

	// 通知所有玩家游戏结束
	r.broadcastGameEnd()
}

// broadcastGameState 广播游戏状态
func (r *Room) broadcastGameState() {
	// TODO: 实现游戏状态广播
}

// broadcastGameStart 广播游戏开始
func (r *Room) broadcastGameStart() {
	// TODO: 实现游戏开始广播
}

// broadcastGameEnd 广播游戏结束
func (r *Room) broadcastGameEnd() {
	// TODO: 实现游戏结束广播
}

// 辅助函数

// getRandomSpawnPosition 获取随机出生点
func getRandomSpawnPosition() models.Vector2D {
	// 临时实现，返回随机位置
	return models.Vector2D{
		X: rand.Float64() * 1000,
		Y: rand.Float64() * 1000,
	}
}

// assignTeam 分配队伍
func assignTeam(r *Room) models.Team {
	if r.Mode != models.TeamDeathMatch && r.Mode != models.FlagCapture {
		return models.TeamNone
	}

	// 统计当前队伍人数
	redCount := 0
	blueCount := 0

	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	for _, player := range r.players {
		if player.Entity.Team == models.TeamRed {
			redCount++
		} else if player.Entity.Team == models.TeamBlue {
			blueCount++
		}
	}

	// 分配到人数较少的队伍
	if redCount <= blueCount {
		return models.TeamRed
	}
	return models.TeamBlue
}
