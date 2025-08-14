// entity.go

package models

import (
	"time"
)

// Vector2D 二维向量
type Vector2D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// EntityType 实体类型
type EntityType string

const (
	// EntityPlayer 玩家实体
	EntityPlayer EntityType = "player"
	// EntityProjectile 投射物实体
	EntityProjectile EntityType = "projectile"
	// EntityEffect 特效实体
	EntityEffect EntityType = "effect"
	// EntityObstacle 障碍物实体
	EntityObstacle EntityType = "obstacle"
	// EntityPickup 拾取物实体
	EntityPickup EntityType = "pickup"
)

// Entity 游戏实体基础接口
type Entity interface {
	GetID() string
	GetType() EntityType
	GetPosition() Vector2D
	GetRotation() float64
	GetVelocity() Vector2D
	GetCreatedAt() time.Time
}

// BaseEntity 基础实体结构
type BaseEntity struct {
	ID        string     `json:"id"`
	Type      EntityType `json:"type"`
	Position  Vector2D   `json:"position"`
	Rotation  float64    `json:"rotation"` // 角度(0-360)
	Velocity  Vector2D   `json:"velocity"`
	CreatedAt time.Time  `json:"created_at"`
}

// GetID 获取实体ID
func (e *BaseEntity) GetID() string {
	return e.ID
}

// GetType 获取实体类型
func (e *BaseEntity) GetType() EntityType {
	return e.Type
}

// GetPosition 获取实体位置
func (e *BaseEntity) GetPosition() Vector2D {
	return e.Position
}

// GetRotation 获取实体旋转
func (e *BaseEntity) GetRotation() float64 {
	return e.Rotation
}

// GetVelocity 获取实体速度
func (e *BaseEntity) GetVelocity() Vector2D {
	return e.Velocity
}

// GetCreatedAt 获取实体创建时间
func (e *BaseEntity) GetCreatedAt() time.Time {
	return e.CreatedAt
}

// PlayerEntity 玩家实体
type PlayerEntity struct {
	BaseEntity
	PlayerID       int64 `json:"player_id"`
	CharacterID    int   `json:"character_id"`
	Team           Team  `json:"team"`

	// 战斗属性
	Health      int  `json:"health"`
	MaxHealth   int  `json:"max_health"`
	IsAlive     bool `json:"is_alive"`
	RespawnTime int  `json:"respawn_time,omitempty"`

	// 技能冷却
	SkillCooldowns map[int]float64 `json:"skill_cooldowns,omitempty"`
	
	// 战斗统计
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
}

// ProjectileEntity 投射物实体
type ProjectileEntity struct {
	BaseEntity
	OwnerID     string   `json:"owner_id"`
	SkillID     int      `json:"skill_id"`
	Damage      int      `json:"damage"`
	LifeTime    float64  `json:"life_time"`              // 生命周期(秒)
	HitEntities []string `json:"hit_entities,omitempty"` // 已命中实体
}

// EffectEntity 特效实体
type EffectEntity struct {
	BaseEntity
	EffectType string  `json:"effect_type"`
	Duration   float64 `json:"duration"`
	Radius     float64 `json:"radius,omitempty"`
	OwnerID    string  `json:"owner_id,omitempty"`
}

// CollisionInfo 碰撞信息
type CollisionInfo struct {
	EntityA  string    `json:"entity_a"`
	EntityB  string    `json:"entity_b"`
	Position Vector2D  `json:"position"`
	Normal   Vector2D  `json:"normal"`
	Time     time.Time `json:"time"`
}
