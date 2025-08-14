// character.go

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

// CharacterHandler 角色处理器
type CharacterHandler struct{}

// NewCharacterHandler 创建角色处理器
func NewCharacterHandler() *CharacterHandler {
	return &CharacterHandler{}
}

// RegisterHandlers 注册HTTP处理器
func (h *CharacterHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/characters", h.handleCharacters)
	mux.HandleFunc("/characters/", h.handleCharacterDetail)
	// 注册具体的角色相关路径
	mux.HandleFunc("/players/characters/", h.handlePlayerCharactersAPI)
	mux.HandleFunc("/players/default-character/", h.handleDefaultCharacterAPI)
}

// CharacterResponse 角色响应
type CharacterResponse struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Data    interface{}         `json:"data"`
}

// PlayerCharacterResponse 玩家角色响应
type PlayerCharacterResponse struct {
	Success bool                          `json:"success"`
	Message string                        `json:"message"`
	Data    *models.PlayerCharacterInfo   `json:"data"`
}

// SetDefaultCharacterRequest 设置默认角色请求
type SetDefaultCharacterRequest struct {
	CharacterID int `json:"character_id"`
}

// handleCharacters 处理角色列表查询
func (h *CharacterHandler) handleCharacters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 查询所有角色
	characters, err := h.getAllCharacters()
	if err != nil {
		log.Printf("查询角色列表失败: %v", err)
		h.sendErrorResponse(w, "查询角色列表失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", characters)
}

// handleCharacterDetail 处理角色详情查询
func (h *CharacterHandler) handleCharacterDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 提取角色ID
	path := strings.TrimPrefix(r.URL.Path, "/characters/")
	characterID, err := strconv.Atoi(path)
	if err != nil {
		h.sendErrorResponse(w, "无效的角色ID", http.StatusBadRequest)
		return
	}

	// 查询角色详情
	character, err := h.getCharacterByID(characterID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendErrorResponse(w, "角色不存在", http.StatusNotFound)
			return
		}
		log.Printf("查询角色详情失败: %v", err)
		h.sendErrorResponse(w, "查询角色详情失败", http.StatusInternalServerError)
		return
	}

	// 查询角色技能
	skills, err := h.getCharacterSkills(characterID)
	if err != nil {
		log.Printf("查询角色技能失败: %v", err)
		// 技能查询失败不影响角色信息返回，只记录日志
	} else {
		character.Skills = skills
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", character)
}

// handlePlayerCharactersAPI 处理玩家角色列表API
func (h *CharacterHandler) handlePlayerCharactersAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 提取玩家ID - 路径格式: /players/characters/{player_id}
	path := strings.TrimPrefix(r.URL.Path, "/players/characters/")
	playerID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	h.handleGetPlayerCharacters(w, r, playerID)
}

// handleDefaultCharacterAPI 处理默认角色API
func (h *CharacterHandler) handleDefaultCharacterAPI(w http.ResponseWriter, r *http.Request) {
	// 提取玩家ID - 路径格式: /players/default-character/{player_id}
	path := strings.TrimPrefix(r.URL.Path, "/players/default-character/")
	playerID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		h.sendErrorResponse(w, "无效的玩家ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetDefaultCharacter(w, r, playerID)
	case http.MethodPost:
		h.handleSetDefaultCharacter(w, r, playerID)
	default:
		h.sendErrorResponse(w, "仅支持GET和POST方法", http.StatusMethodNotAllowed)
	}
}

// 保留原有的处理方法以兼容旧的路径结构（如果需要）
// handlePlayerCharacters 处理玩家角色相关请求（已弃用，保留兼容性）
func (h *CharacterHandler) handlePlayerCharacters(w http.ResponseWriter, r *http.Request) {
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

	switch parts[1] {
	case "characters":
		if r.Method == http.MethodGet {
			h.handleGetPlayerCharacters(w, r, playerID)
		} else {
			h.sendErrorResponse(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		}
	case "default-character":
		if r.Method == http.MethodPost {
			h.handleSetDefaultCharacter(w, r, playerID)
		} else if r.Method == http.MethodGet {
			h.handleGetDefaultCharacter(w, r, playerID)
		} else {
			h.sendErrorResponse(w, "仅支持GET和POST方法", http.StatusMethodNotAllowed)
		}
	default:
		h.sendErrorResponse(w, "未知的请求路径", http.StatusNotFound)
	}
}

// handleGetPlayerCharacters 处理获取玩家角色列表
func (h *CharacterHandler) handleGetPlayerCharacters(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 查询玩家已解锁的角色
	characters, err := h.getPlayerCharacters(playerID)
	if err != nil {
		log.Printf("查询玩家角色失败: %v", err)
		h.sendErrorResponse(w, "查询玩家角色失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", characters)
}

// handleSetDefaultCharacter 处理设置默认角色
func (h *CharacterHandler) handleSetDefaultCharacter(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 解析请求
	var req SetDefaultCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证角色ID
	if req.CharacterID <= 0 {
		h.sendErrorResponse(w, "无效的角色ID", http.StatusBadRequest)
		return
	}

	// 检查玩家是否拥有该角色
	hasCharacter, err := h.checkPlayerHasCharacter(playerID, req.CharacterID)
	if err != nil {
		log.Printf("检查玩家角色失败: %v", err)
		h.sendErrorResponse(w, "检查玩家角色失败", http.StatusInternalServerError)
		return
	}

	if !hasCharacter {
		h.sendErrorResponse(w, "玩家未拥有该角色", http.StatusBadRequest)
		return
	}

	// 设置默认角色
	err = h.setPlayerDefaultCharacter(playerID, req.CharacterID)
	if err != nil {
		log.Printf("设置默认角色失败: %v", err)
		h.sendErrorResponse(w, "设置默认角色失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "设置成功", nil)
}

// handleGetDefaultCharacter 处理获取默认角色
func (h *CharacterHandler) handleGetDefaultCharacter(w http.ResponseWriter, r *http.Request, playerID int64) {
	// 查询玩家默认角色
	characterID, err := h.getPlayerDefaultCharacter(playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.sendErrorResponse(w, "玩家未设置默认角色", http.StatusNotFound)
			return
		}
		log.Printf("查询默认角色失败: %v", err)
		h.sendErrorResponse(w, "查询默认角色失败", http.StatusInternalServerError)
		return
	}

	// 查询角色详情
	character, err := h.getCharacterByID(characterID)
	if err != nil {
		log.Printf("查询角色详情失败: %v", err)
		h.sendErrorResponse(w, "查询角色详情失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	h.sendSuccessResponse(w, "查询成功", character)
}

// sendSuccessResponse 发送成功响应
func (h *CharacterHandler) sendSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	resp := CharacterResponse{
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
func (h *CharacterHandler) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	resp := CharacterResponse{
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

// getAllCharacters 获取所有角色
func (h *CharacterHandler) getAllCharacters() ([]models.Character, error) {
	query := `
		SELECT id, name, description, max_hp, speed, base_attack, base_defense,
		       special_ability, difficulty, role, unlockable, unlock_cost
		FROM characters
		ORDER BY id
	`

	rows, err := db.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	defer rows.Close()

	var characters []models.Character
	for rows.Next() {
		var char models.Character
		err := rows.Scan(
			&char.ID, &char.Name, &char.Description, &char.MaxHP, &char.Speed,
			&char.BaseAttack, &char.BaseDefense, &char.SpecialAbility,
			&char.Difficulty, &char.Role, &char.Unlockable, &char.UnlockCost,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描角色数据失败: %w", err)
		}
		characters = append(characters, char)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历角色数据失败: %w", err)
	}

	return characters, nil
}

// getCharacterByID 根据ID获取角色
func (h *CharacterHandler) getCharacterByID(characterID int) (*models.Character, error) {
	query := `
		SELECT id, name, description, max_hp, speed, base_attack, base_defense,
		       special_ability, difficulty, role, unlockable, unlock_cost
		FROM characters
		WHERE id = $1
	`

	var char models.Character
	err := db.DB.QueryRow(query, characterID).Scan(
		&char.ID, &char.Name, &char.Description, &char.MaxHP, &char.Speed,
		&char.BaseAttack, &char.BaseDefense, &char.SpecialAbility,
		&char.Difficulty, &char.Role, &char.Unlockable, &char.UnlockCost,
	)

	if err != nil {
		return nil, err
	}

	return &char, nil
}

// getCharacterSkills 获取角色技能
func (h *CharacterHandler) getCharacterSkills(characterID int) ([]models.Skill, error) {
	query := `
		SELECT s.id, s.name, s.description, s.type, s.damage, s.cooldown_time,
		       s.range, s.effect_time, s.projectile_speed, s.projectile_count,
		       s.projectile_spread, s.animation_key, s.effect_key
		FROM skills s
		INNER JOIN character_skills cs ON s.id = cs.skill_id
		WHERE cs.character_id = $1
		ORDER BY cs.slot_index, s.id
	`

	rows, err := db.DB.Query(query, characterID)
	if err != nil {
		return nil, fmt.Errorf("查询角色技能失败: %w", err)
	}
	defer rows.Close()

	var skills []models.Skill
	for rows.Next() {
		var skill models.Skill
		var projectileSpeed, projectileSpread sql.NullFloat64
		var projectileCount sql.NullInt64
		var animationKey, effectKey sql.NullString

		err := rows.Scan(
			&skill.ID, &skill.Name, &skill.Description, &skill.Type, &skill.Damage,
			&skill.CooldownTime, &skill.Range, &skill.EffectTime,
			&projectileSpeed, &projectileCount, &projectileSpread,
			&animationKey, &effectKey,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描技能数据失败: %w", err)
		}

		// 处理可空字段
		if projectileSpeed.Valid {
			skill.ProjectileSpeed = projectileSpeed.Float64
		}
		if projectileCount.Valid {
			skill.ProjectileCount = int(projectileCount.Int64)
		}
		if projectileSpread.Valid {
			skill.ProjectileSpread = projectileSpread.Float64
		}
		if animationKey.Valid {
			skill.AnimationKey = animationKey.String
		}
		if effectKey.Valid {
			skill.EffectKey = effectKey.String
		}

		skills = append(skills, skill)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历技能数据失败: %w", err)
	}

	return skills, nil
}

// getPlayerCharacters 获取玩家已解锁的角色
func (h *CharacterHandler) getPlayerCharacters(playerID int64) ([]models.Character, error) {
	query := `
		SELECT c.id, c.name, c.description, c.max_hp, c.speed, c.base_attack,
		       c.base_defense, c.special_ability, c.difficulty, c.role,
		       c.unlockable, c.unlock_cost
		FROM characters c
		INNER JOIN player_characters pc ON c.id = pc.character_id
		WHERE pc.player_id = $1
		ORDER BY c.id
	`

	rows, err := db.DB.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("查询玩家角色失败: %w", err)
	}
	defer rows.Close()

	var characters []models.Character
	for rows.Next() {
		var char models.Character
		err := rows.Scan(
			&char.ID, &char.Name, &char.Description, &char.MaxHP, &char.Speed,
			&char.BaseAttack, &char.BaseDefense, &char.SpecialAbility,
			&char.Difficulty, &char.Role, &char.Unlockable, &char.UnlockCost,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描玩家角色数据失败: %w", err)
		}
		characters = append(characters, char)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历玩家角色数据失败: %w", err)
	}

	return characters, nil
}

// checkPlayerHasCharacter 检查玩家是否拥有指定角色
func (h *CharacterHandler) checkPlayerHasCharacter(playerID int64, characterID int) (bool, error) {
	query := `
		SELECT COUNT(1) FROM player_characters
		WHERE player_id = $1 AND character_id = $2
	`

	var count int
	err := db.DB.QueryRow(query, playerID, characterID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查玩家角色失败: %w", err)
	}

	return count > 0, nil
}

// getPlayerDefaultCharacter 获取玩家默认角色ID
func (h *CharacterHandler) getPlayerDefaultCharacter(playerID int64) (int, error) {
	query := `
		SELECT character_id FROM player_default_characters
		WHERE player_id = $1
	`

	var characterID int
	err := db.DB.QueryRow(query, playerID).Scan(&characterID)
	if err != nil {
		return 0, err
	}

	return characterID, nil
}

// setPlayerDefaultCharacter 设置玩家默认角色
func (h *CharacterHandler) setPlayerDefaultCharacter(playerID int64, characterID int) error {
	// 使用 UPSERT 语法（PostgreSQL）
	query := `
		INSERT INTO player_default_characters (player_id, character_id)
		VALUES ($1, $2)
		ON CONFLICT (player_id)
		DO UPDATE SET character_id = EXCLUDED.character_id
	`

	_, err := db.DB.Exec(query, playerID, characterID)
	if err != nil {
		return fmt.Errorf("设置默认角色失败: %w", err)
	}

	return nil
}
