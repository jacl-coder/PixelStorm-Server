// stats.go

package gateway

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jacl-coder/PixelStorm-Server/internal/models"
	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

// StatsHandler 战绩处理器
type StatsHandler struct {
	redisLeaderboard *models.RedisLeaderboard
	useRedis         bool
}

// NewStatsHandler 创建战绩处理器
func NewStatsHandler() *StatsHandler {
	useRedis := db.RedisClient != nil
	var redisLeaderboard *models.RedisLeaderboard

	if useRedis {
		redisLeaderboard = models.NewRedisLeaderboard()
	}

	return &StatsHandler{
		redisLeaderboard: redisLeaderboard,
		useRedis:         useRedis,
	}
}

// RegisterHandlers 注册HTTP处理器
func (h *StatsHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/stats/player/", h.handlePlayerStats)
	mux.HandleFunc("/stats/matches/", h.handlePlayerMatches)
	mux.HandleFunc("/stats/leaderboard", h.handleLeaderboard)
	mux.HandleFunc("/stats/leaderboard/refresh", h.handleRefreshLeaderboard)
}

// StatsResponse 战绩响应
type StatsResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PlayerMatchesResponse 玩家对局响应
type PlayerMatchesResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message"`
	Data    *PlayerMatchesData          `json:"data"`
}

// PlayerMatchesData 玩家对局数据
type PlayerMatchesData struct {
	Matches []models.PlayerMatchRecord `json:"matches"`
	Total   int                        `json:"total"`
	Page    int                        `json:"page"`
	Limit   int                        `json:"limit"`
}

// LeaderboardResponse 排行榜响应
type LeaderboardResponse struct {
	Success bool                      `json:"success"`
	Message string                    `json:"message"`
	Data    []models.LeaderboardEntry `json:"data"`
}

// handlePlayerStats 处理玩家战绩查询
func (h *StatsHandler) handlePlayerStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 提取玩家ID
	path := strings.TrimPrefix(r.URL.Path, "/stats/player/")
	playerID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	// 查询玩家战绩统计
	stats, err := h.getPlayerStats(playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendErrorResponse(w, "玩家不存在", http.StatusNotFound)
			return
		}
		log.Printf("查询玩家战绩失败: %v", err)
		h.sendErrorResponse(w, "查询玩家战绩失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", stats)
}

// handlePlayerMatches 处理玩家对局历史查询
func (h *StatsHandler) handlePlayerMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 提取玩家ID
	path := strings.TrimPrefix(r.URL.Path, "/stats/matches/")
	playerID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	// 解析查询参数
	query := r.URL.Query()
	limit := 10 // 默认限制
	offset := 0 // 默认偏移

	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// 查询玩家对局历史
	matches, total, err := h.getPlayerMatches(playerID, limit, offset)
	if err != nil {
		log.Printf("查询玩家对局历史失败: %v", err)
		h.sendErrorResponse(w, "查询对局历史失败", http.StatusInternalServerError)
		return
	}

	// 构建响应数据
	data := &PlayerMatchesData{
		Matches: matches,
		Total:   total,
		Page:    offset/limit + 1,
		Limit:   limit,
	}

	// 返回成功响应
	h.sendMatchesResponse(w, "查询成功", data)
}

// handleLeaderboard 处理排行榜查询
func (h *StatsHandler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析查询参数
	query := r.URL.Query()
	leaderboardType := query.Get("type")
	if leaderboardType == "" {
		leaderboardType = "score" // 默认按综合得分排序
	}

	limit := 50 // 默认限制
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// 验证排行榜类型
	validTypes := map[string]bool{
		"kills": true,
		"wins":  true,
		"score": true,
		"kda":   true,
	}

	if !validTypes[leaderboardType] {
		h.sendErrorResponse(w, "无效的排行榜类型", http.StatusBadRequest)
		return
	}

	// 查询排行榜
	leaderboard, err := h.getLeaderboard(models.LeaderboardType(leaderboardType), limit)
	if err != nil {
		log.Printf("查询排行榜失败: %v", err)
		h.sendErrorResponse(w, "查询排行榜失败", http.StatusInternalServerError)
		return
	}

	log.Printf("排行榜查询结果: 类型=%s, 数量=%d", leaderboardType, len(leaderboard))

	// 返回成功响应
	h.sendLeaderboardResponse(w, "查询成功", leaderboard)
}

// handleRefreshLeaderboard 处理排行榜刷新
func (h *StatsHandler) handleRefreshLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendErrorResponse(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	if !h.useRedis {
		h.sendErrorResponse(w, "Redis未启用，无需刷新", http.StatusBadRequest)
		return
	}

	// 刷新排行榜
	if err := h.redisLeaderboard.RefreshLeaderboard(); err != nil {
		log.Printf("刷新排行榜失败: %v", err)
		h.sendErrorResponse(w, "刷新排行榜失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "排行榜刷新成功", nil)
}

// sendSuccessResponse 发送成功响应
func (h *StatsHandler) sendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	resp := StatsResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// sendMatchesResponse 发送对局响应
func (h *StatsHandler) sendMatchesResponse(w http.ResponseWriter, message string, data *PlayerMatchesData) {
	resp := PlayerMatchesResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// sendLeaderboardResponse 发送排行榜响应
func (h *StatsHandler) sendLeaderboardResponse(w http.ResponseWriter, message string, data []models.LeaderboardEntry) {
	resp := LeaderboardResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// sendErrorResponse 发送错误响应
func (h *StatsHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	resp := StatsResponse{
		Success: false,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码错误响应失败: %v", err)
	}
}

// 数据库查询方法

// getPlayerStats 获取玩家战绩统计
func (h *StatsHandler) getPlayerStats(playerID int64) (*models.PlayerStats, error) {
	query := `
		SELECT
			p.id as player_id,
			p.total_matches,
			p.total_wins,
			(p.total_matches - p.total_wins) as losses,
			CASE WHEN p.total_matches > 0 THEN (p.total_wins * 100.0 / p.total_matches) ELSE 0 END as win_rate,
			p.total_kills,
			p.total_deaths,
			COALESCE(SUM(pmr.assists), 0) as total_assists,
			CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + COALESCE(SUM(pmr.assists), 0)) * 1.0 / p.total_deaths)
				 ELSE (p.total_kills + COALESCE(SUM(pmr.assists), 0)) END as kda,
			CASE WHEN p.total_matches > 0 THEN (COALESCE(SUM(pmr.score), 0) * 1.0 / p.total_matches) ELSE 0 END as average_score,
			COALESCE(SUM(CASE WHEN pmr.mvp = true THEN 1 ELSE 0 END), 0) as total_mvp,
			COALESCE(SUM(pmr.play_time), 0) as play_time
		FROM players p
		LEFT JOIN player_match_records pmr ON p.id = pmr.player_id
		WHERE p.id = $1
		GROUP BY p.id, p.total_matches, p.total_wins, p.total_kills, p.total_deaths
	`

	var stats models.PlayerStats
	err := db.DB.QueryRow(query, playerID).Scan(
		&stats.PlayerID, &stats.TotalMatches, &stats.TotalWins, &stats.Losses,
		&stats.WinRate, &stats.TotalKills, &stats.TotalDeaths, &stats.TotalAssists,
		&stats.KDA, &stats.AverageScore, &stats.TotalMVP, &stats.PlayTime,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// getPlayerMatches 获取玩家对局历史
func (h *StatsHandler) getPlayerMatches(playerID int64, limit, offset int) ([]models.PlayerMatchRecord, int, error) {
	// 先查询总数
	countQuery := `
		SELECT COUNT(*) FROM player_match_records
		WHERE player_id = $1
	`

	var total int
	err := db.DB.QueryRow(countQuery, playerID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询对局总数失败: %w", err)
	}

	// 查询对局记录
	query := `
		SELECT pmr.match_id, pmr.player_id, pmr.character_id, pmr.team, pmr.score,
		       pmr.kills, pmr.deaths, pmr.assists, pmr.exp_gained, pmr.coins_gained,
		       pmr.mvp, pmr.play_time, pmr.join_time, pmr.leave_time
		FROM player_match_records pmr
		WHERE pmr.player_id = $1
		ORDER BY pmr.join_time DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.DB.Query(query, playerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询对局记录失败: %w", err)
	}
	defer rows.Close()

	var matches []models.PlayerMatchRecord
	for rows.Next() {
		var match models.PlayerMatchRecord
		err := rows.Scan(
			&match.MatchID, &match.PlayerID, &match.CharacterID, &match.Team,
			&match.Score, &match.Kills, &match.Deaths, &match.Assists,
			&match.ExpGained, &match.CoinsGained, &match.MVP,
			&match.PlayTime, &match.JoinTime, &match.LeaveTime,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描对局记录失败: %w", err)
		}
		matches = append(matches, match)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历对局记录失败: %w", err)
	}

	return matches, total, nil
}

// getLeaderboard 获取排行榜
func (h *StatsHandler) getLeaderboard(leaderboardType models.LeaderboardType, limit int) ([]models.LeaderboardEntry, error) {
	// 优先使用Redis
	if h.useRedis {
		entries, err := h.redisLeaderboard.GetLeaderboard(leaderboardType, limit)
		if err == nil && len(entries) > 0 {
			return entries, nil
		}

		// Redis失败或无数据时，刷新排行榜并重试
		log.Printf("Redis排行榜查询失败或无数据，刷新排行榜: %v", err)
		if refreshErr := h.redisLeaderboard.RefreshLeaderboard(); refreshErr == nil {
			if entries, err := h.redisLeaderboard.GetLeaderboard(leaderboardType, limit); err == nil {
				return entries, nil
			}
		}

		log.Printf("Redis排行榜刷新失败，回退到数据库查询")
	}

	// 回退到数据库查询
	return h.getLeaderboardFromDB(leaderboardType, limit)
}

// getLeaderboardFromDB 从数据库获取排行榜
func (h *StatsHandler) getLeaderboardFromDB(leaderboardType models.LeaderboardType, limit int) ([]models.LeaderboardEntry, error) {
	var orderBy string

	switch leaderboardType {
	case models.LeaderboardKills:
		orderBy = "p.total_kills DESC"
	case models.LeaderboardWins:
		orderBy = "p.total_wins DESC"
	case models.LeaderboardKDA:
		orderBy = "CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + p.total_assists) * 1.0 / p.total_deaths) ELSE (p.total_kills + p.total_assists) END DESC"
	case models.LeaderboardScore:
		orderBy = "(p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) DESC"
	default:
		orderBy = "(p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) DESC"
	}

	query := fmt.Sprintf(`
		SELECT
			p.id AS player_id,
			p.username,
			p.level,
			p.total_kills,
			p.total_wins,
			CASE WHEN p.total_matches > 0 THEN (p.total_wins * 100.0 / p.total_matches) ELSE 0 END AS win_rate,
			CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + p.total_assists) * 1.0 / p.total_deaths)
				 ELSE (p.total_kills + p.total_assists) END AS kda,
			(p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) AS score,
			ROW_NUMBER() OVER (ORDER BY %s) as rank
		FROM players p
		ORDER BY %s
		LIMIT $1
	`, orderBy, orderBy)

	rows, err := db.DB.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询排行榜失败: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.PlayerID, &entry.Username, &entry.Level, &entry.TotalKills,
			&entry.TotalWins, &entry.WinRate, &entry.KDA, &entry.Score, &entry.Rank,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描排行榜数据失败: %w", err)
		}
		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历排行榜数据失败: %w", err)
	}

	return entries, nil
}
