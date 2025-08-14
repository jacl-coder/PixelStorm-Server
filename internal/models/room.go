package models

import (
	"time"
)

// GameMode 游戏模式
type GameMode string

const (
	// DeathMatch 死亡竞赛模式
	DeathMatch GameMode = "death_match"
	// TeamDeathMatch 团队死亡竞赛
	TeamDeathMatch GameMode = "team_death_match"
	// CapturePoint 据点占领
	CapturePoint GameMode = "capture_point"
	// FlagCapture 夺旗模式
	FlagCapture GameMode = "flag_capture"
)

// RoomStatus 房间状态
type RoomStatus string

const (
	// RoomWaiting 等待中
	RoomWaiting RoomStatus = "waiting"
	// RoomPlaying 游戏中
	RoomPlaying RoomStatus = "playing"
	// RoomEnded 已结束
	RoomEnded RoomStatus = "ended"
)

// Team 队伍
type Team int

const (
	// TeamNone 无队伍
	TeamNone Team = 0
	// TeamRed 红队
	TeamRed Team = 1
	// TeamBlue 蓝队
	TeamBlue Team = 2
)

// Room 游戏房间
type Room struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Mode       GameMode   `json:"mode"`
	Status     RoomStatus `json:"status"`
	MaxPlayers int        `json:"max_players"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  time.Time  `json:"started_at,omitempty"`
	EndedAt    time.Time  `json:"ended_at,omitempty"`
	MapID      int        `json:"map_id"`

	// 房间设置
	TimeLimit    int    `json:"time_limit"`    // 时间限制(秒)
	ScoreLimit   int    `json:"score_limit"`   // 分数限制
	FriendlyFire bool   `json:"friendly_fire"` // 友军伤害
	PrivateRoom  bool   `json:"private_room"`  // 私人房间
	Password     string `json:"-"`             // 房间密码

	// 房间内玩家
	Players []RoomPlayer `json:"players,omitempty"`
}

// RoomPlayer 房间内的玩家
type RoomPlayer struct {
	PlayerID    int64  `json:"player_id"`
	Username    string `json:"username"`
	CharacterID int    `json:"character_id"`
	Team        Team   `json:"team"`
	Ready       bool   `json:"ready"`

	// 游戏中数据
	Score   int `json:"score"`
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
}

// GameMap 游戏地图
type GameMap struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImagePath   string `json:"image_path"`

	// 地图属性
	Width          int        `json:"width"`
	Height         int        `json:"height"`
	MaxPlayers     int        `json:"max_players"`
	SupportedModes []GameMode `json:"supported_modes"`
}

// 注意：表结构定义已移至 pkg/db/schema.go 统一管理
