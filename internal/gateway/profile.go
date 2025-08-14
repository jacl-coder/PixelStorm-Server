package gateway

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jacl-coder/PixelStorm-Server/internal/models"
	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

// ProfileHandler 玩家资料处理器
type ProfileHandler struct{}

// NewProfileHandler 创建玩家资料处理器
func NewProfileHandler() *ProfileHandler {
	return &ProfileHandler{}
}

// RegisterHandlers 注册HTTP处理器
func (h *ProfileHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/players/", h.handlePlayerProfile)
}

// ProfileResponse 资料响应
type ProfileResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

// PlayerProfileInfo 玩家资料信息
type PlayerProfileInfo struct {
	*models.Player
	Statistics *PlayerStatistics `json:"statistics"`
}

// PlayerStatistics 玩家统计信息
type PlayerStatistics struct {
	WinRate     float64 `json:"win_rate"`     // 胜率
	KDA         float64 `json:"kda"`          // KDA比率
	AverageKill float64 `json:"average_kill"` // 平均击杀
	PlayTime    int     `json:"play_time"`    // 总游戏时长(分钟)
}

// handlePlayerProfile 处理玩家资料相关请求
func (h *ProfileHandler) handlePlayerProfile(w http.ResponseWriter, r *http.Request) {
	// 解析URL路径
	path := strings.TrimPrefix(r.URL.Path, "/players/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		h.sendErrorResponse(w, "无效的请求路径", http.StatusBadRequest)
		return
	}

	playerID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	if parts[1] != "profile" {
		h.sendErrorResponse(w, "未知的请求路径", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetPlayerProfile(w, r, playerID)
	case http.MethodPut:
		h.handleUpdatePlayerProfile(w, r, playerID)
	default:
		h.sendErrorResponse(w, "仅支持GET和PUT方法", http.StatusMethodNotAllowed)
	}
}

// handleGetPlayerProfile 处理获取玩家资料
func (h *ProfileHandler) handleGetPlayerProfile(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 查询玩家基本信息
	player, err := h.getPlayerByID(playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendErrorResponse(w, "玩家不存在", http.StatusNotFound)
			return
		}
		log.Printf("查询玩家信息失败: %v", err)
		h.sendErrorResponse(w, "查询玩家信息失败", http.StatusInternalServerError)
		return
	}

	// 查询玩家统计信息
	statistics, err := h.getPlayerStatistics(playerID)
	if err != nil {
		log.Printf("查询玩家统计信息失败: %v", err)
		// 统计信息查询失败不影响基本信息返回
		statistics = &PlayerStatistics{}
	}

	// 构建响应数据
	profileInfo := &PlayerProfileInfo{
		Player:     player,
		Statistics: statistics,
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", profileInfo)
}

// handleUpdatePlayerProfile 处理更新玩家资料
func (h *ProfileHandler) handleUpdatePlayerProfile(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 解析请求
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证请求数据
	if req.Username == "" && req.Email == "" {
		h.sendErrorResponse(w, "至少需要提供一个更新字段", http.StatusBadRequest)
		return
	}

	// 检查玩家是否存在
	exists, err := h.checkPlayerExists(playerID)
	if err != nil {
		log.Printf("检查玩家存在性失败: %v", err)
		h.sendErrorResponse(w, "检查玩家信息失败", http.StatusInternalServerError)
		return
	}

	if !exists {
		h.sendErrorResponse(w, "玩家不存在", http.StatusNotFound)
		return
	}

	// 更新玩家信息
	err = h.updatePlayerProfile(playerID, &req)
	if err != nil {
		log.Printf("更新玩家资料失败: %v", err)
		// 检查是否是唯一约束冲突
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "username") {
				h.sendErrorResponse(w, "用户名已存在", http.StatusConflict)
			} else if strings.Contains(err.Error(), "email") {
				h.sendErrorResponse(w, "邮箱已存在", http.StatusConflict)
			} else {
				h.sendErrorResponse(w, "数据冲突", http.StatusConflict)
			}
			return
		}
		h.sendErrorResponse(w, "更新玩家资料失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "更新成功", nil)
}

// sendSuccessResponse 发送成功响应
func (h *ProfileHandler) sendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	resp := ProfileResponse{
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
func (h *ProfileHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	resp := ProfileResponse{
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

// getPlayerByID 根据ID获取玩家信息
func (h *ProfileHandler) getPlayerByID(playerID int64) (*models.Player, error) {
	query := `
		SELECT id, username, email, created_at, updated_at, level, exp, coins, gems,
		       total_kills, total_deaths, total_assists, total_matches, total_wins
		FROM players
		WHERE id = $1
	`

	var player models.Player
	err := db.DB.QueryRow(query, playerID).Scan(
		&player.ID, &player.Username, &player.Email, &player.CreatedAt, &player.UpdatedAt,
		&player.Level, &player.Exp, &player.Coins, &player.Gems,
		&player.TotalKills, &player.TotalDeaths, &player.TotalAssists, &player.TotalMatches, &player.TotalWins,
	)

	if err != nil {
		return nil, err
	}

	return &player, nil
}

// getPlayerStatistics 获取玩家统计信息
func (h *ProfileHandler) getPlayerStatistics(playerID int64) (*PlayerStatistics, error) {
	query := `
		SELECT
			CASE WHEN total_matches > 0 THEN (total_wins * 100.0 / total_matches) ELSE 0 END as win_rate,
			CASE WHEN total_deaths > 0 THEN (total_kills * 1.0 / total_deaths) ELSE total_kills END as kda,
			CASE WHEN total_matches > 0 THEN (total_kills * 1.0 / total_matches) ELSE 0 END as average_kill,
			COALESCE(SUM(pmr.play_time), 0) / 60 as play_time_minutes
		FROM players p
		LEFT JOIN player_match_records pmr ON p.id = pmr.player_id
		WHERE p.id = $1
		GROUP BY p.id, p.total_matches, p.total_wins, p.total_kills, p.total_deaths
	`
	
	var stats PlayerStatistics
	err := db.DB.QueryRow(query, playerID).Scan(
		&stats.WinRate, &stats.KDA, &stats.AverageKill, &stats.PlayTime,
	)
	
	if err != nil {
		return nil, fmt.Errorf("查询玩家统计信息失败: %w", err)
	}

	return &stats, nil
}

// checkPlayerExists 检查玩家是否存在
func (h *ProfileHandler) checkPlayerExists(playerID int64) (bool, error) {
	query := `SELECT COUNT(1) FROM players WHERE id = $1`

	var count int
	err := db.DB.QueryRow(query, playerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查玩家存在性失败: %w", err)
	}

	return count > 0, nil
}

// updatePlayerProfile 更新玩家资料
func (h *ProfileHandler) updatePlayerProfile(playerID int64, req *UpdateProfileRequest) error {
	// 构建动态更新SQL
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Username != "" {
		setParts = append(setParts, fmt.Sprintf("username = $%d", argIndex))
		args = append(args, req.Username)
		argIndex++
	}

	if req.Email != "" {
		setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
		args = append(args, req.Email)
		argIndex++
	}

	// 添加更新时间
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// 添加WHERE条件
	args = append(args, playerID)

	query := fmt.Sprintf(`
		UPDATE players
		SET %s
		WHERE id = $%d
	`, strings.Join(setParts, ", "), argIndex)

	_, err := db.DB.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("更新玩家资料失败: %w", err)
	}

	return nil
}
