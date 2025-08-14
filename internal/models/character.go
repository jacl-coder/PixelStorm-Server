package models

// Character 角色模型
type Character struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// 基础属性
	MaxHP       int     `json:"max_hp"`
	Speed       float64 `json:"speed"`
	BaseAttack  int     `json:"base_attack"`
	BaseDefense int     `json:"base_defense"`

	// 角色特性
	SpecialAbility string `json:"special_ability"`

	// 技能列表
	Skills []Skill `json:"skills"`

	// 角色元数据
	Difficulty int    `json:"difficulty"`  // 难度等级 1-5
	Role       string `json:"role"`        // 角色定位，如"攻击手"、"辅助"等
	Unlockable bool   `json:"unlockable"`  // 是否可解锁（有些角色可能是默认的）
	UnlockCost int    `json:"unlock_cost"` // 解锁花费
}

// PlayerCharacter 玩家拥有的角色
type PlayerCharacter struct {
	PlayerID    int64 `json:"player_id"`
	CharacterID int   `json:"character_id"`
	Level       int   `json:"level"`
	Exp         int   `json:"exp"`
	Unlocked    bool  `json:"unlocked"`
	UsageCount  int   `json:"usage_count"` // 使用次数
	WinCount    int   `json:"win_count"`   // 胜利次数
	KillCount   int   `json:"kill_count"`  // 击杀数
	DeathCount  int   `json:"death_count"` // 死亡数
}

// PlayerDefaultCharacter 玩家默认角色
type PlayerDefaultCharacter struct {
	PlayerID    int64 `json:"player_id"`
	CharacterID int   `json:"character_id"`
}

// CharacterUnlockRequirement 角色解锁条件
type CharacterUnlockRequirement struct {
	CharacterID     int   `json:"character_id"`
	RequiredLevel   int   `json:"required_level"`   // 需要的玩家等级
	RequiredCoins   int64 `json:"required_coins"`   // 需要的金币
	RequiredGems    int64 `json:"required_gems"`    // 需要的宝石
	RequiredMatches int   `json:"required_matches"` // 需要的对局数
}

// PlayerCharacterInfo 玩家角色信息
type PlayerCharacterInfo struct {
	Characters      []Character `json:"characters"`       // 玩家拥有的角色列表
	DefaultCharacter *Character `json:"default_character"` // 默认角色
}

// 注意：表结构定义已移至 pkg/db/schema.go 统一管理
