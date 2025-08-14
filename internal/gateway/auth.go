package gateway

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	// 会话缓存，现在支持Redis
	sessions    map[string]SessionInfo
	useRedis    bool
	sessionTTL  time.Duration
}

// SessionInfo 会话信息
type SessionInfo struct {
	PlayerID  int64
	Username  string
	ExpiresAt time.Time
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Token    string `json:"token,omitempty"`
	PlayerID int64  `json:"player_id,omitempty"`
	Username string `json:"username,omitempty"`
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler() *AuthHandler {
	// 检查Redis是否可用
	useRedis := db.RedisClient != nil

	return &AuthHandler{
		sessions:   make(map[string]SessionInfo),
		useRedis:   useRedis,
		sessionTTL: 24 * time.Hour,
	}
}

// RegisterHandlers 注册HTTP处理器
func (h *AuthHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.handleLogin)
	mux.HandleFunc("/auth/register", h.handleRegister)
	mux.HandleFunc("/auth/validate", h.handleValidate)
	mux.HandleFunc("/auth/logout", h.handleLogout)
}

// handleLogin 处理登录请求
func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证用户名和密码
	playerID, err := h.validateCredentials(req.Username, req.Password)
	if err != nil {
		// 返回错误响应
		resp := AuthResponse{
			Success: false,
			Message: "用户名或密码错误",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 生成会话令牌
	token, err := h.generateToken()
	if err != nil {
		http.Error(w, "生成令牌失败", http.StatusInternalServerError)
		return
	}

	// 保存会话信息
	sessionInfo := SessionInfo{
		PlayerID:  playerID,
		Username:  req.Username,
		ExpiresAt: time.Now().Add(h.sessionTTL),
	}
	h.setSession(token, sessionInfo)

	// 返回成功响应
	resp := AuthResponse{
		Success:  true,
		Message:  "登录成功",
		Token:    token,
		PlayerID: playerID,
		Username: req.Username,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleRegister 处理注册请求
func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求格式", http.StatusBadRequest)
		return
	}

	// 验证请求
	if req.Username == "" || req.Password == "" || req.Email == "" {
		http.Error(w, "缺少必要参数", http.StatusBadRequest)
		return
	}

	// 创建用户
	playerID, err := h.createUser(req.Username, req.Password, req.Email)
	if err != nil {
		// 返回错误响应
		resp := AuthResponse{
			Success: false,
			Message: fmt.Sprintf("注册失败: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 生成会话令牌
	token, err := h.generateToken()
	if err != nil {
		http.Error(w, "生成令牌失败", http.StatusInternalServerError)
		return
	}

	// 保存会话信息
	sessionInfo := SessionInfo{
		PlayerID:  playerID,
		Username:  req.Username,
		ExpiresAt: time.Now().Add(h.sessionTTL),
	}
	h.setSession(token, sessionInfo)

	// 返回成功响应
	resp := AuthResponse{
		Success:  true,
		Message:  "注册成功",
		Token:    token,
		PlayerID: playerID,
		Username: req.Username,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleValidate 处理令牌验证请求
func (h *AuthHandler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET方法", http.StatusMethodNotAllowed)
		return
	}

	// 获取令牌
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "未提供令牌", http.StatusBadRequest)
			return
		}
	}

	// 验证令牌
	session, ok := h.getSession(token)
	if !ok || time.Now().After(session.ExpiresAt) {
		// 令牌无效或已过期
		if ok {
			// 删除过期会话
			h.deleteSession(token)
		}
		http.Error(w, "无效或已过期的令牌", http.StatusUnauthorized)
		return
	}

	// 返回成功响应
	resp := AuthResponse{
		Success:  true,
		Message:  "令牌有效",
		PlayerID: session.PlayerID,
		Username: session.Username,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleLogout 处理登出请求
func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 获取令牌
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "未提供令牌", http.StatusBadRequest)
			return
		}
	}

	// 删除会话
	h.deleteSession(token)

	// 返回成功响应
	resp := AuthResponse{
		Success: true,
		Message: "登出成功",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// validateCredentials 验证用户凭据
func (h *AuthHandler) validateCredentials(username, password string) (int64, error) {
	// 计算密码哈希
	hashedPassword := hashPassword(password)

	// 查询数据库
	var playerID int64
	err := db.DB.QueryRow("SELECT id FROM players WHERE username = $1 AND password = $2", username, hashedPassword).Scan(&playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("用户名或密码错误")
		}
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}

	return playerID, nil
}

// createUser 创建用户
func (h *AuthHandler) createUser(username, password, email string) (int64, error) {
	// 检查用户名是否已存在
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM players WHERE username = $1", username).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	err = db.DB.QueryRow("SELECT COUNT(*) FROM players WHERE email = $1", email).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("数据库查询错误: %w", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("邮箱已被使用")
	}

	// 计算密码哈希
	hashedPassword := hashPassword(password)

	// 插入用户
	var playerID int64
	err = db.DB.QueryRow(
		"INSERT INTO players (username, password, email, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) RETURNING id",
		username, hashedPassword, email,
	).Scan(&playerID)
	if err != nil {
		return 0, fmt.Errorf("创建用户失败: %w", err)
	}

	return playerID, nil
}

// generateToken 生成随机令牌
func (h *AuthHandler) generateToken() (string, error) {
	// 生成32字节的随机数
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	// 转换为Base64字符串
	return base64.URLEncoding.EncodeToString(b), nil
}

// hashPassword 计算密码哈希
func hashPassword(password string) string {
	// 使用SHA-256哈希
	// 在实际应用中，应该使用更安全的哈希算法，如bcrypt
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// setSession 设置会话信息
func (h *AuthHandler) setSession(token string, session SessionInfo) {
	if h.useRedis {
		// 使用Redis存储
		sessionKey := "session:" + token
		sessionData := fmt.Sprintf("%d:%s:%d", session.PlayerID, session.Username, session.ExpiresAt.Unix())

		err := db.RedisClient.Set(db.RedisClient.Context(), sessionKey, sessionData, h.sessionTTL).Err()
		if err != nil {
			// Redis失败时回退到内存存储
			h.sessions[token] = session
		}
	} else {
		// 使用内存存储
		h.sessions[token] = session
	}
}

// getSession 获取会话信息
func (h *AuthHandler) getSession(token string) (SessionInfo, bool) {
	if h.useRedis {
		// 从Redis获取
		sessionKey := "session:" + token
		sessionData, err := db.RedisClient.Get(db.RedisClient.Context(), sessionKey).Result()
		if err != nil {
			// Redis失败时尝试内存存储
			session, ok := h.sessions[token]
			return session, ok
		}

		// 解析会话数据
		parts := strings.Split(sessionData, ":")
		if len(parts) != 3 {
			return SessionInfo{}, false
		}

		playerID, _ := strconv.ParseInt(parts[0], 10, 64)
		username := parts[1]
		expiresAt, _ := strconv.ParseInt(parts[2], 10, 64)

		session := SessionInfo{
			PlayerID:  playerID,
			Username:  username,
			ExpiresAt: time.Unix(expiresAt, 0),
		}

		return session, true
	} else {
		// 从内存获取
		session, ok := h.sessions[token]
		return session, ok
	}
}

// deleteSession 删除会话信息
func (h *AuthHandler) deleteSession(token string) {
	if h.useRedis {
		// 从Redis删除
		sessionKey := "session:" + token
		db.RedisClient.Del(db.RedisClient.Context(), sessionKey)
	}

	// 同时从内存删除（如果存在）
	delete(h.sessions, token)
}

// ValidateToken 验证令牌（供其他模块使用）
func (h *AuthHandler) ValidateToken(token string) (int64, string, bool) {
	session, ok := h.getSession(token)
	if !ok || time.Now().After(session.ExpiresAt) {
		if ok {
			h.deleteSession(token)
		}
		return 0, "", false
	}

	return session.PlayerID, session.Username, true
}
