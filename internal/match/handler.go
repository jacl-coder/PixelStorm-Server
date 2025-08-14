package match

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jacl-coder/PixelStorm-Server/internal/models"
)

// MatchHandler 匹配处理器
type MatchHandler struct {
	service *MatchService
}

// NewMatchHandler 创建匹配处理器
func NewMatchHandler(service *MatchService) *MatchHandler {
	return &MatchHandler{
		service: service,
	}
}

// RegisterHandlers 注册HTTP处理器
func (h *MatchHandler) RegisterHandlers(mux *http.ServeMux) {
	// 健康检查端点
	mux.HandleFunc("/health", h.handleHealth)

	// 匹配相关端点
	mux.HandleFunc("/match/join", h.handleJoinQueue)
	mux.HandleFunc("/match/leave", h.handleLeaveQueue)
	mux.HandleFunc("/match/status", h.handleMatchStatus)
	mux.HandleFunc("/match/history/", h.handleMatchHistory)
	mux.HandleFunc("/match/preferences/", h.handleMatchPreferences)
}

// handleHealth 处理健康检查请求
func (h *MatchHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 检查服务状态
	if h.service == nil {
		http.Error(w, "服务未初始化", http.StatusServiceUnavailable)
		return
	}

	// 返回健康状态
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// 匹配请求
type joinQueueRequest struct {
	PlayerID    int64           `json:"player_id"`
	CharacterID int             `json:"character_id"`
	GameMode    models.GameMode `json:"game_mode"`
	SessionID   string          `json:"session_id"`
}

// 匹配响应
type matchResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// 匹配状态响应
type matchStatusResponse struct {
	Queues map[models.GameMode]int `json:"queues"`
}

// 匹配历史响应
type matchHistoryResponse struct {
	Success bool                        `json:"success"`
	Message string                      `json:"message"`
	Data    *matchHistoryData           `json:"data"`
}

// 匹配历史数据
type matchHistoryData struct {
	History []matchHistoryEntry `json:"history"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	Limit   int                 `json:"limit"`
}

// 匹配历史条目
type matchHistoryEntry struct {
	MatchID     string              `json:"match_id"`
	GameMode    models.GameMode     `json:"game_mode"`
	JoinTime    string              `json:"join_time"`
	MatchTime   string              `json:"match_time,omitempty"`
	Status      string              `json:"status"` // waiting, matched, cancelled
	WaitTime    int                 `json:"wait_time"` // 等待时间(秒)
}

// 匹配偏好请求
type matchPreferencesRequest struct {
	PreferredModes []models.GameMode `json:"preferred_modes"`
	PreferredMaps  []int             `json:"preferred_maps"`
	MaxWaitTime    int               `json:"max_wait_time"` // 最大等待时间(秒)
	SkillLevel     string            `json:"skill_level"`   // beginner, intermediate, advanced
}

// 匹配偏好响应
type matchPreferencesResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    *matchPreferencesRequest `json:"data"`
}

// handleJoinQueue 处理加入匹配队列请求
func (h *MatchHandler) handleJoinQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req joinQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证请求
	if req.PlayerID <= 0 || req.CharacterID <= 0 || req.GameMode == "" || req.SessionID == "" {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		return
	}

	// 添加到匹配队列
	h.service.AddToQueue(req.PlayerID, req.CharacterID, req.GameMode, req.SessionID)

	// 返回成功响应
	resp := matchResponse{
		Success: true,
		Message: "已加入匹配队列",
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// handleLeaveQueue 处理离开匹配队列请求
func (h *MatchHandler) handleLeaveQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "仅支持POST或DELETE方法", http.StatusMethodNotAllowed)
		return
	}

	// 获取参数
	playerIDStr := r.URL.Query().Get("player_id")
	gameModeStr := r.URL.Query().Get("game_mode")

	if playerIDStr == "" || gameModeStr == "" {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		return
	}

	// 解析参数
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	// 从队列移除
	success := h.service.RemoveFromQueue(playerID, models.GameMode(gameModeStr))

	// 返回响应
	resp := matchResponse{
		Success: success,
		Message: "已离开匹配队列",
	}
	if !success {
		resp.Message = "玩家不在匹配队列中"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// handleMatchStatus 处理获取匹配状态请求
func (h *MatchHandler) handleMatchStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 获取所有队列长度
	queueLengths := h.service.GetAllQueueLengths()

	// 返回响应
	resp := matchStatusResponse{
		Queues: queueLengths,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// handleMatchHistory 处理匹配历史查询
func (h *MatchHandler) handleMatchHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 提取玩家ID
	path := r.URL.Path
	playerIDStr := path[len("/match/history/"):]
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	// 解析查询参数
	query := r.URL.Query()
	limit := 20 // 默认限制
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

	// 查询匹配历史（这里使用模拟数据，实际应从数据库查询）
	history, total := h.getMatchHistory(playerID, limit, offset)

	// 构建响应数据
	data := &matchHistoryData{
		History: history,
		Total:   total,
		Page:    offset/limit + 1,
		Limit:   limit,
	}

	// 返回响应
	resp := matchHistoryResponse{
		Success: true,
		Message: "查询成功",
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// handleMatchPreferences 处理匹配偏好设置
func (h *MatchHandler) handleMatchPreferences(w http.ResponseWriter, r *http.Request) {
	// 提取玩家ID
	path := r.URL.Path
	playerIDStr := path[len("/match/preferences/"):]
	playerID, err := strconv.ParseInt(playerIDStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetMatchPreferences(w, r, playerID)
	case http.MethodPost:
		h.handleSetMatchPreferences(w, r, playerID)
	default:
		http.Error(w, "仅支持GET和POST方法", http.StatusMethodNotAllowed)
	}
}

// handleGetMatchPreferences 获取匹配偏好
func (h *MatchHandler) handleGetMatchPreferences(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 查询玩家匹配偏好（这里使用模拟数据，实际应从数据库查询）
	preferences := h.getMatchPreferences(playerID)

	// 返回响应
	resp := matchPreferencesResponse{
		Success: true,
		Message: "查询成功",
		Data:    preferences,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// handleSetMatchPreferences 设置匹配偏好
func (h *MatchHandler) handleSetMatchPreferences(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 解析请求
	var req matchPreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证请求数据
	if len(req.PreferredModes) == 0 {
		http.Error(w, "至少需要选择一个偏好游戏模式", http.StatusBadRequest)
		return
	}

	if req.MaxWaitTime <= 0 || req.MaxWaitTime > 600 {
		http.Error(w, "最大等待时间必须在1-600秒之间", http.StatusBadRequest)
		return
	}

	// 保存匹配偏好（这里使用模拟保存，实际应保存到数据库）
	err := h.saveMatchPreferences(playerID, &req)
	if err != nil {
		log.Printf("保存匹配偏好失败: %v", err)
		http.Error(w, "保存匹配偏好失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	resp := matchPreferencesResponse{
		Success: true,
		Message: "设置成功",
		Data:    &req,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("编码响应失败: %v", err)
	}
}

// 辅助方法

// getMatchHistory 获取匹配历史（模拟数据）
func (h *MatchHandler) getMatchHistory(playerID int64, limit, offset int) ([]matchHistoryEntry, int) {
	// 这里使用模拟数据，实际应从数据库查询
	// 在真实实现中，应该查询 match_history 表

	allHistory := []matchHistoryEntry{
		{
			MatchID:   "match_001",
			GameMode:  models.DeathMatch,
			JoinTime:  "2024-01-15T10:30:00Z",
			MatchTime: "2024-01-15T10:32:15Z",
			Status:    "matched",
			WaitTime:  135,
		},
		{
			MatchID:   "match_002",
			GameMode:  models.TeamDeathMatch,
			JoinTime:  "2024-01-15T11:15:00Z",
			MatchTime: "2024-01-15T11:16:45Z",
			Status:    "matched",
			WaitTime:  105,
		},
		{
			MatchID:   "match_003",
			GameMode:  models.DeathMatch,
			JoinTime:  "2024-01-15T14:20:00Z",
			MatchTime: "",
			Status:    "cancelled",
			WaitTime:  300,
		},
	}

	total := len(allHistory)

	// 分页处理
	start := offset
	end := offset + limit
	if start >= total {
		return []matchHistoryEntry{}, total
	}
	if end > total {
		end = total
	}

	return allHistory[start:end], total
}

// getMatchPreferences 获取匹配偏好（模拟数据）
func (h *MatchHandler) getMatchPreferences(playerID int64) *matchPreferencesRequest {
	// 这里使用模拟数据，实际应从数据库查询
	// 在真实实现中，应该查询 player_match_preferences 表

	return &matchPreferencesRequest{
		PreferredModes: []models.GameMode{models.DeathMatch, models.TeamDeathMatch},
		PreferredMaps:  []int{1, 2},
		MaxWaitTime:    300,
		SkillLevel:     "intermediate",
	}
}

// saveMatchPreferences 保存匹配偏好（模拟保存）
func (h *MatchHandler) saveMatchPreferences(playerID int64, preferences *matchPreferencesRequest) error {
	// 这里使用模拟保存，实际应保存到数据库
	// 在真实实现中，应该更新 player_match_preferences 表

	log.Printf("保存玩家 %d 的匹配偏好: %+v", playerID, preferences)
	return nil
}
