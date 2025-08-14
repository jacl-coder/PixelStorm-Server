// skill.go

package models

// SkillType 技能类型
type SkillType string

const (
	// ProjectileSkill 投射物技能
	ProjectileSkill SkillType = "projectile"
	// AOESkill 范围伤害技能
	AOESkill SkillType = "aoe"
	// BuffSkill 增益技能
	BuffSkill SkillType = "buff"
	// DebuffSkill 减益技能
	DebuffSkill SkillType = "debuff"
	// MovementSkill 移动技能
	MovementSkill SkillType = "movement"
	// UtilitySkill 功能性技能
	UtilitySkill SkillType = "utility"
)

// Skill 技能模型
type Skill struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        SkillType `json:"type"`

	// 技能属性
	Damage       int     `json:"damage"`
	CooldownTime float64 `json:"cooldown_time"` // 冷却时间(秒)
	Range        float64 `json:"range"`         // 射程/范围
	EffectTime   float64 `json:"effect_time"`   // 效果持续时间(秒)

	// 投射物属性
	ProjectileSpeed  float64 `json:"projectile_speed,omitempty"`
	ProjectileCount  int     `json:"projectile_count,omitempty"`
	ProjectileSpread float64 `json:"projectile_spread,omitempty"` // 散射角度

	// 视觉效果
	AnimationKey string `json:"animation_key"`
	EffectKey    string `json:"effect_key"`
}

// CharacterSkill 角色技能关联
type CharacterSkill struct {
	CharacterID int `json:"character_id"`
	SkillID     int `json:"skill_id"`
	SlotIndex   int `json:"slot_index"` // 技能槽位置
}

// 注意：表结构定义已移至 pkg/db/schema.go 统一管理
