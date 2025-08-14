package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

// RedisLeaderboard Redis排行榜管理器
type RedisLeaderboard struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisLeaderboard 创建Redis排行榜管理器
func NewRedisLeaderboard() *RedisLeaderboard {
	return &RedisLeaderboard{
		client: db.RedisClient,
		ctx:    context.Background(),
	}
}

// 排行榜Redis键名
const (
	LeaderboardKillsKey = "leaderboard:kills"
	LeaderboardWinsKey  = "leaderboard:wins"
	LeaderboardScoreKey = "leaderboard:score"
	LeaderboardKDAKey   = "leaderboard:kda"
	
	// 玩家详细信息键前缀
	PlayerInfoPrefix = "player:info:"
	
	// 排行榜缓存时间
	LeaderboardCacheTTL = 5 * time.Minute
)

// UpdatePlayerScore 更新玩家分数
func (rl *RedisLeaderboard) UpdatePlayerScore(playerID int64, scoreType LeaderboardType, score float64) error {
	key := rl.getLeaderboardKey(scoreType)
	return rl.client.ZAdd(rl.ctx, key, &redis.Z{
		Score:  score,
		Member: playerID,
	}).Err()
}

// UpdatePlayerInfo 更新玩家信息
func (rl *RedisLeaderboard) UpdatePlayerInfo(player *LeaderboardEntry) error {
	key := fmt.Sprintf("%s%d", PlayerInfoPrefix, player.PlayerID)
	
	data, err := json.Marshal(player)
	if err != nil {
		return err
	}
	
	return rl.client.Set(rl.ctx, key, data, LeaderboardCacheTTL).Err()
}

// GetLeaderboard 获取排行榜
func (rl *RedisLeaderboard) GetLeaderboard(scoreType LeaderboardType, limit int) ([]LeaderboardEntry, error) {
	key := rl.getLeaderboardKey(scoreType)
	
	// 从Redis获取排行榜（按分数降序）
	members, err := rl.client.ZRevRangeWithScores(rl.ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	
	var entries []LeaderboardEntry
	for i, member := range members {
		playerID, err := strconv.ParseInt(member.Member.(string), 10, 64)
		if err != nil {
			continue
		}
		
		// 获取玩家详细信息
		playerInfo, err := rl.getPlayerInfo(playerID)
		if err != nil {
			// 如果Redis中没有玩家信息，从数据库获取
			playerInfo, err = rl.getPlayerInfoFromDB(playerID)
			if err != nil {
				continue
			}
			// 缓存到Redis
			rl.UpdatePlayerInfo(playerInfo)
		}
		
		// 更新分数和排名
		playerInfo.Score = member.Score
		playerInfo.Rank = i + 1
		
		entries = append(entries, *playerInfo)
	}
	
	return entries, nil
}

// GetPlayerRank 获取玩家排名
func (rl *RedisLeaderboard) GetPlayerRank(playerID int64, scoreType LeaderboardType) (int, error) {
	key := rl.getLeaderboardKey(scoreType)
	
	rank, err := rl.client.ZRevRank(rl.ctx, key, strconv.FormatInt(playerID, 10)).Result()
	if err != nil {
		if err == redis.Nil {
			return -1, nil // 玩家不在排行榜中
		}
		return -1, err
	}
	
	return int(rank) + 1, nil // Redis排名从0开始，转换为从1开始
}

// RefreshLeaderboard 刷新排行榜（从数据库重新加载）
func (rl *RedisLeaderboard) RefreshLeaderboard() error {
	// 查询数据库获取最新数据
	query := `
		SELECT
			p.id AS player_id,
			p.username,
			p.level,
			p.total_kills,
			p.total_wins,
			CASE WHEN p.total_matches > 0 THEN (p.total_wins * 100.0 / p.total_matches) ELSE 0 END AS win_rate,
			CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + p.total_assists) * 1.0 / p.total_deaths)
				 ELSE (p.total_kills + p.total_assists) END AS kda,
			(p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) AS score
		FROM players p
		WHERE 1=1
		ORDER BY score DESC
		LIMIT 1000
	`
	
	rows, err := db.DB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	// 清空现有排行榜
	keys := []string{
		LeaderboardKillsKey,
		LeaderboardWinsKey,
		LeaderboardScoreKey,
		LeaderboardKDAKey,
	}
	
	for _, key := range keys {
		rl.client.Del(rl.ctx, key)
	}
	
	// 重新填充排行榜
	for rows.Next() {
		var entry LeaderboardEntry
		err := rows.Scan(
			&entry.PlayerID, &entry.Username, &entry.Level,
			&entry.TotalKills, &entry.TotalWins, &entry.WinRate,
			&entry.KDA, &entry.Score,
		)
		if err != nil {
			continue
		}
		
		// 更新各种排行榜
		rl.UpdatePlayerScore(entry.PlayerID, LeaderboardKills, float64(entry.TotalKills))
		rl.UpdatePlayerScore(entry.PlayerID, LeaderboardWins, float64(entry.TotalWins))
		rl.UpdatePlayerScore(entry.PlayerID, LeaderboardScore, entry.Score)
		rl.UpdatePlayerScore(entry.PlayerID, LeaderboardKDA, entry.KDA)
		
		// 缓存玩家信息
		rl.UpdatePlayerInfo(&entry)
	}
	
	return nil
}

// getLeaderboardKey 获取排行榜键名
func (rl *RedisLeaderboard) getLeaderboardKey(scoreType LeaderboardType) string {
	switch scoreType {
	case LeaderboardKills:
		return LeaderboardKillsKey
	case LeaderboardWins:
		return LeaderboardWinsKey
	case LeaderboardKDA:
		return LeaderboardKDAKey
	case LeaderboardScore:
		return LeaderboardScoreKey
	default:
		return LeaderboardScoreKey
	}
}

// getPlayerInfo 从Redis获取玩家信息
func (rl *RedisLeaderboard) getPlayerInfo(playerID int64) (*LeaderboardEntry, error) {
	key := fmt.Sprintf("%s%d", PlayerInfoPrefix, playerID)
	
	data, err := rl.client.Get(rl.ctx, key).Result()
	if err != nil {
		return nil, err
	}
	
	var entry LeaderboardEntry
	err = json.Unmarshal([]byte(data), &entry)
	if err != nil {
		return nil, err
	}
	
	return &entry, nil
}

// getPlayerInfoFromDB 从数据库获取玩家信息
func (rl *RedisLeaderboard) getPlayerInfoFromDB(playerID int64) (*LeaderboardEntry, error) {
	query := `
		SELECT
			p.id AS player_id,
			p.username,
			p.level,
			p.total_kills,
			p.total_wins,
			CASE WHEN p.total_matches > 0 THEN (p.total_wins * 100.0 / p.total_matches) ELSE 0 END AS win_rate,
			CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + p.total_assists) * 1.0 / p.total_deaths)
				 ELSE (p.total_kills + p.total_assists) END AS kda,
			(p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) AS score
		FROM players p
		WHERE p.id = $1
	`
	
	var entry LeaderboardEntry
	err := db.DB.QueryRow(query, playerID).Scan(
		&entry.PlayerID, &entry.Username, &entry.Level,
		&entry.TotalKills, &entry.TotalWins, &entry.WinRate,
		&entry.KDA, &entry.Score,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &entry, nil
}

// SetLeaderboardTTL 设置排行榜过期时间
func (rl *RedisLeaderboard) SetLeaderboardTTL(ttl time.Duration) error {
	keys := []string{
		LeaderboardKillsKey,
		LeaderboardWinsKey,
		LeaderboardScoreKey,
		LeaderboardKDAKey,
	}
	
	for _, key := range keys {
		if err := rl.client.Expire(rl.ctx, key, ttl).Err(); err != nil {
			return err
		}
	}
	
	return nil
}
