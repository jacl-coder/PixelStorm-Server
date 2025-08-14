package game

import (
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jacl-coder/PixelStorm-Server/internal/models"
	"github.com/jacl-coder/PixelStorm-Server/internal/protocol"
)

// 碰撞检测常量
const (
	// 碰撞检测精度
	collisionPrecision = 0.1

	// 玩家碰撞半径
	playerRadius = 20.0

	// 投射物碰撞半径
	projectileRadius = 10.0
)

// detectCollisions 检测碰撞
func (r *Room) detectCollisions() {
	r.entityMutex.Lock()
	defer r.entityMutex.Unlock()

	// 获取所有实体
	entities := make([]models.Entity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}

	// 检测碰撞
	collisions := make([]models.CollisionInfo, 0)
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			entityA := entities[i]
			entityB := entities[j]

			// 检查是否是投射物和玩家
			var projectile *models.ProjectileEntity
			var player *models.PlayerEntity
			var isCollision bool

			// 确定哪个是投射物，哪个是玩家
			if entityA.GetType() == models.EntityProjectile && entityB.GetType() == models.EntityPlayer {
				projectile = entityA.(*models.ProjectileEntity)
				player = entityB.(*models.PlayerEntity)
				isCollision = true
			} else if entityB.GetType() == models.EntityProjectile && entityA.GetType() == models.EntityPlayer {
				projectile = entityB.(*models.ProjectileEntity)
				player = entityA.(*models.PlayerEntity)
				isCollision = true
			}

			// 如果是投射物和玩家，检查碰撞
			if isCollision && player.IsAlive {
				// 检查投射物是否已经击中该玩家
				hasHit := false
				for _, hitID := range projectile.HitEntities {
					if hitID == player.ID {
						hasHit = true
						break
					}
				}

				// 如果已经击中，跳过
				if hasHit {
					continue
				}

				// 检查是否是友军
				isFriendlyFire := false
				// 获取投射物所有者
				var ownerEntity models.Entity
				if projectile.OwnerID != "" {
					ownerEntity = r.entities[projectile.OwnerID]
				}

				// 如果所有者是玩家，检查是否是友军
				if ownerEntity != nil && ownerEntity.GetType() == models.EntityPlayer {
					ownerPlayer := ownerEntity.(*models.PlayerEntity)
					// 如果是同一队伍且不允许友军伤害，则跳过
					if ownerPlayer.Team == player.Team && ownerPlayer.Team != models.TeamNone && !r.FriendlyFire {
						isFriendlyFire = true
					}
				}

				// 如果是友军伤害且不允许友军伤害，跳过
				if isFriendlyFire {
					continue
				}

				// 检查距离
				posA := projectile.GetPosition()
				posB := player.GetPosition()
				dx := posA.X - posB.X
				dy := posA.Y - posB.Y
				distance := math.Sqrt(dx*dx + dy*dy)

				// 如果距离小于两者半径之和，则发生碰撞
				if distance < projectileRadius+playerRadius {
					// 记录碰撞
					collision := models.CollisionInfo{
						EntityA:  projectile.ID,
						EntityB:  player.ID,
						Position: models.Vector2D{X: (posA.X + posB.X) / 2, Y: (posA.Y + posB.Y) / 2},
						Normal:   models.Vector2D{X: dx / distance, Y: dy / distance},
						Time:     time.Now(),
					}
					collisions = append(collisions, collision)

					// 处理碰撞
					r.handleCollision(projectile, player)
				}
			}
		}
	}

	// 广播碰撞事件
	if len(collisions) > 0 {
		r.broadcastCollisions(collisions)
	}
}

// handleCollision 处理碰撞
func (r *Room) handleCollision(projectile *models.ProjectileEntity, player *models.PlayerEntity) {
	// 将玩家添加到投射物的命中列表
	projectile.HitEntities = append(projectile.HitEntities, player.ID)

	// 计算伤害
	damage := projectile.Damage

	// 应用伤害
	player.Health -= damage
	if player.Health <= 0 {
		player.Health = 0
		player.IsAlive = false
		player.RespawnTime = 5 // 5秒后重生

		// 更新击杀统计
		if projectile.OwnerID != "" {
			// 获取投射物所有者
			ownerEntity := r.entities[projectile.OwnerID]
			if ownerEntity != nil && ownerEntity.GetType() == models.EntityPlayer {
				ownerPlayer := ownerEntity.(*models.PlayerEntity)

				// 更新玩家分数
				r.playerMutex.Lock()
				for _, ps := range r.players {
					if ps.Entity.ID == ownerPlayer.ID {
						ps.Entity.Kills++
						r.scores[ownerPlayer.PlayerID]++
						break
					}
				}
				r.playerMutex.Unlock()

				// 更新被击杀玩家的死亡次数
				r.playerMutex.Lock()
				for _, ps := range r.players {
					if ps.Entity.ID == player.ID {
						ps.Entity.Deaths++
						break
					}
				}
				r.playerMutex.Unlock()

				// 广播击杀事件
				r.broadcastKill(ownerPlayer.PlayerID, player.PlayerID)
			}
		}
	}
}

// CreateProjectile 创建投射物
func (r *Room) CreateProjectile(owner *models.PlayerEntity, skillID int, direction models.Vector2D, damage int, speed float64, lifetime float64) *models.ProjectileEntity {
	// 创建投射物
	projectile := &models.ProjectileEntity{
		BaseEntity: models.BaseEntity{
			ID:        uuid.New().String(),
			Type:      models.EntityProjectile,
			Position:  owner.Position,
			Rotation:  math.Atan2(direction.Y, direction.X) * 180 / math.Pi,
			Velocity:  models.Vector2D{X: direction.X * speed, Y: direction.Y * speed},
			CreatedAt: time.Now(),
		},
		OwnerID:     owner.ID,
		SkillID:     skillID,
		Damage:      damage,
		LifeTime:    lifetime,
		HitEntities: []string{},
	}

	// 添加到实体列表
	r.entityMutex.Lock()
	r.entities[projectile.ID] = projectile
	r.entityMutex.Unlock()

	return projectile
}

// UseSkill 使用技能
func (r *Room) UseSkill(player *models.PlayerEntity, skillID int, targetPos models.Vector2D) error {
	// 检查技能冷却
	if cooldown, ok := player.SkillCooldowns[skillID]; ok && cooldown > 0 {
		return nil // 技能冷却中
	}

	// 计算方向
	playerPos := player.GetPosition()
	dx := targetPos.X - playerPos.X
	dy := targetPos.Y - playerPos.Y
	length := math.Sqrt(dx*dx + dy*dy)

	// 归一化方向向量
	if length > 0 {
		dx /= length
		dy /= length
	}

	direction := models.Vector2D{X: dx, Y: dy}

	// 根据技能ID创建不同的投射物
	switch skillID {
	case 1: // 普通射击
		r.CreateProjectile(player, skillID, direction, 10, 500, 2.0)
		player.SkillCooldowns[skillID] = 0.5 // 0.5秒冷却
	case 2: // 散射
		for i := -1; i <= 1; i++ {
			angle := float64(i) * 15 * math.Pi / 180 // 每个投射物相差15度
			rotatedDir := rotateVector(direction, angle)
			r.CreateProjectile(player, skillID, rotatedDir, 8, 450, 1.5)
		}
		player.SkillCooldowns[skillID] = 3.0 // 3秒冷却
	case 3: // 穿透弹
		projectile := r.CreateProjectile(player, skillID, direction, 15, 400, 3.0)
		projectile.HitEntities = make([]string, 0) // 可以穿透多个目标
		player.SkillCooldowns[skillID] = 5.0       // 5秒冷却
	}

	return nil
}

// broadcastCollisions 广播碰撞事件
func (r *Room) broadcastCollisions(collisions []models.CollisionInfo) {
	// 转换为协议消息
	events := make([]*protocol.CollisionEvent, 0, len(collisions))
	for _, collision := range collisions {
		events = append(events, &protocol.CollisionEvent{
			EntityA:  collision.EntityA,
			EntityB:  collision.EntityB,
			Position: &protocol.Vector2D{X: float32(collision.Position.X), Y: float32(collision.Position.Y)},
			Damage:   int32(getDamageForCollision(collision, r.entities)),
		})
	}

	// 构建游戏帧消息
	frame := &protocol.GameFrame{
		FrameId:       r.frameID,
		Timestamp:     time.Now().UnixNano() / int64(time.Millisecond),
		Collisions:    events,
		RemainingTime: int32(r.TimeLimit - int(time.Since(r.StartedAt).Seconds())),
	}

	// 将分数添加到帧
	frame.Scores = make(map[int64]int32)
	for playerID, score := range r.scores {
		frame.Scores[playerID] = int32(score)
	}

	// 序列化
	data, err := json.Marshal(frame)
	if err != nil {
		log.Printf("序列化碰撞事件失败: %v", err)
		return
	}

	// 广播给房间内所有玩家
	r.playerMutex.RLock()
	defer r.playerMutex.RUnlock()

	for _, player := range r.players {
		if player.Connection != nil {
			select {
			case player.Connection.Send <- data:
				// 消息已发送
			default:
				// 通道已满，跳过
			}
		}
	}
}

// broadcastKill 广播击杀事件
func (r *Room) broadcastKill(killerID, victimID int64) {
	// TODO: 实现击杀事件广播
}

// 辅助函数

// getDamageForCollision 获取碰撞伤害
func getDamageForCollision(collision models.CollisionInfo, entities map[string]models.Entity) int {
	// 获取投射物
	var projectile *models.ProjectileEntity
	entityA := entities[collision.EntityA]
	entityB := entities[collision.EntityB]

	if entityA != nil && entityA.GetType() == models.EntityProjectile {
		projectile = entityA.(*models.ProjectileEntity)
	} else if entityB != nil && entityB.GetType() == models.EntityProjectile {
		projectile = entityB.(*models.ProjectileEntity)
	}

	if projectile != nil {
		return projectile.Damage
	}
	return 0
}

// rotateVector 旋转向量
func rotateVector(v models.Vector2D, angle float64) models.Vector2D {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return models.Vector2D{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
	}
}
