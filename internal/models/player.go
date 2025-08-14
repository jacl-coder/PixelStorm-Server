// plays.go

package models

import (
	"time"
)

// Player 玩家模型
type Player struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // 不序列化密码
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 游戏相关属性
	Level int   `json:"level"`
	Exp   int64 `json:"exp"`
	Coins int64 `json:"coins"`
	Gems  int64 `json:"gems"`

	// 战斗数据统计
	TotalKills   int `json:"total_kills"`
	TotalDeaths  int `json:"total_deaths"`
	TotalAssists int `json:"total_assists"`
	TotalMatches int `json:"total_matches"`
	TotalWins    int `json:"total_wins"`
}

// PlayerSession 玩家会话信息
type PlayerSession struct {
	PlayerID  int64  `json:"player_id"`
	SessionID string `json:"session_id"`
	RoomID    string `json:"room_id,omitempty"`
}

// 注意：表结构定义已移至 pkg/db/schema.go 统一管理
