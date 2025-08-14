// stats.go

package models

import (
	"time"
)

// MatchRecord 对局记录
type MatchRecord struct {
	ID          string    `json:"id"`
	GameMode    GameMode  `json:"game_mode"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	WinningTeam int       `json:"winning_team"`
	MapID       int       `json:"map_id"`
	Duration    int       `json:"duration"` // 对局时长(秒)
}

// PlayerMatchRecord 玩家对局记录
type PlayerMatchRecord struct {
	MatchID     string    `json:"match_id"`
	PlayerID    int64     `json:"player_id"`
	CharacterID int       `json:"character_id"`
	Team        int       `json:"team"`
	Score       int       `json:"score"`
	Kills       int       `json:"kills"`
	Deaths      int       `json:"deaths"`
	Assists     int       `json:"assists"`
	ExpGained   int       `json:"exp_gained"`
	CoinsGained int       `json:"coins_gained"`
	MVP         bool      `json:"mvp"`        // 是否为MVP
	PlayTime    int       `json:"play_time"`  // 游戏时长(秒)
	JoinTime    time.Time `json:"join_time"`  // 加入时间
	LeaveTime   time.Time `json:"leave_time"` // 离开时间
}

// PlayerStats 玩家战绩统计
type PlayerStats struct {
	PlayerID     int64   `json:"player_id"`
	TotalMatches int     `json:"total_matches"`
	TotalWins    int     `json:"total_wins"`
	Losses       int     `json:"losses"`
	WinRate      float64 `json:"win_rate"`
	TotalKills   int     `json:"total_kills"`
	TotalDeaths  int     `json:"total_deaths"`
	TotalAssists int     `json:"total_assists"`
	KDA          float64 `json:"kda"`           // (击杀+助攻)/死亡
	AverageScore float64 `json:"average_score"` // 平均得分
	TotalMVP     int     `json:"total_mvp"`     // MVP次数
	PlayTime     int     `json:"play_time"`     // 总游戏时长(秒)
}

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
	PlayerID   int64   `json:"player_id"`
	Username   string  `json:"username"`
	Level      int     `json:"level"`
	TotalKills int     `json:"total_kills"`
	TotalWins  int     `json:"total_wins"`
	WinRate    float64 `json:"win_rate"`
	KDA        float64 `json:"kda"`
	Score      float64 `json:"score"` // 综合评分
	Rank       int     `json:"rank"`  // 排名
}

// LeaderboardType 排行榜类型
type LeaderboardType string

const (
	// LeaderboardKills 击杀排行榜
	LeaderboardKills LeaderboardType = "kills"
	// LeaderboardWins 胜场排行榜
	LeaderboardWins LeaderboardType = "wins"
	// LeaderboardScore 综合得分排行榜
	LeaderboardScore LeaderboardType = "score"
	// LeaderboardKDA KDA排行榜
	LeaderboardKDA LeaderboardType = "kda"
)

// 注意：表结构定义已移至 pkg/db/schema.go 统一管理
