# PixelStorm 游戏大厅功能实现计划

## 概述

本文档详细说明了PixelStorm游戏大厅功能的服务端实现计划，包括登录、匹配、战绩查询和角色选择等核心功能。

## 1. 战绩查询系统

### 1.1 数据模型扩展

在 `internal/models` 目录下创建 `stats.go` 文件，添加以下模型：

```go
// MatchRecord 对局记录
type MatchRecord struct {
    ID          string    `json:"id"`
    GameMode    GameMode  `json:"game_mode"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    WinningTeam int       `json:"winning_team"`
    MapID       int       `json:"map_id"`
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
}

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
    PlayerID   int64  `json:"player_id"`
    Username   string `json:"username"`
    Level      int    `json:"level"`
    TotalKills int    `json:"total_kills"`
    TotalWins  int    `json:"total_wins"`
    Score      int    `json:"score"` // 综合评分
}
```

### 1.2 数据库表创建

添加以下SQL创建语句：

```go
// CreateMatchRecordsTable 创建对局记录表SQL
const CreateMatchRecordsTable = `
CREATE TABLE IF NOT EXISTS match_records (
    id VARCHAR(36) PRIMARY KEY,
    game_mode VARCHAR(20) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    winning_team INT NOT NULL,
    map_id INT NOT NULL
);
`

// CreatePlayerMatchRecordsTable 创建玩家对局记录表SQL
const CreatePlayerMatchRecordsTable = `
CREATE TABLE IF NOT EXISTS player_match_records (
    match_id VARCHAR(36) REFERENCES match_records(id),
    player_id BIGINT REFERENCES players(id),
    character_id INT REFERENCES characters(id),
    team INT NOT NULL,
    score INT NOT NULL,
    kills INT NOT NULL,
    deaths INT NOT NULL,
    assists INT NOT NULL,
    exp_gained INT NOT NULL,
    coins_gained INT NOT NULL,
    PRIMARY KEY (match_id, player_id)
);
`
```

### 1.3 API实现

在 `internal/gateway` 目录下创建 `stats.go` 文件，实现以下API：

1. **个人战绩查询**：
   - 路径：`/stats/player/{player_id}`
   - 方法：GET
   - 功能：查询玩家的总体战绩统计

2. **对局历史查询**：
   - 路径：`/stats/matches/{player_id}`
   - 方法：GET
   - 参数：limit, offset
   - 功能：查询玩家的历史对局记录

3. **排行榜查询**：
   - 路径：`/stats/leaderboard`
   - 方法：GET
   - 参数：type (kills, wins, score), limit
   - 功能：查询不同类型的排行榜

## 2. 角色选择系统

### 2.1 API实现

在 `internal/gateway` 目录下创建 `character.go` 文件，实现以下API：

1. **角色列表查询**：
   - 路径：`/characters`
   - 方法：GET
   - 功能：查询所有可用角色

2. **角色详情查询**：
   - 路径：`/characters/{character_id}`
   - 方法：GET
   - 功能：查询单个角色的详细信息，包括技能

3. **玩家角色查询**：
   - 路径：`/players/{player_id}/characters`
   - 方法：GET
   - 功能：查询玩家已解锁的角色

4. **设置默认角色**：
   - 路径：`/players/{player_id}/default-character`
   - 方法：POST
   - 功能：设置玩家的默认角色

### 2.2 数据模型扩展

在 `internal/models/character.go` 中添加：

```go
// PlayerDefaultCharacter 玩家默认角色
type PlayerDefaultCharacter struct {
    PlayerID    int64 `json:"player_id"`
    CharacterID int   `json:"character_id"`
}

// CreatePlayerDefaultCharacterTable 创建玩家默认角色表SQL
const CreatePlayerDefaultCharacterTable = `
CREATE TABLE IF NOT EXISTS player_default_characters (
    player_id BIGINT REFERENCES players(id) PRIMARY KEY,
    character_id INT REFERENCES characters(id) NOT NULL
);
`
```

## 3. 游戏大厅功能完善

### 3.1 玩家资料API

在 `internal/gateway` 目录下创建 `profile.go` 文件，实现以下API：

1. **玩家资料查询**：
   - 路径：`/players/{player_id}/profile`
   - 方法：GET
   - 功能：查询玩家的详细资料，包括等级、经验、货币等

2. **玩家资料更新**：
   - 路径：`/players/{player_id}/profile`
   - 方法：PUT
   - 功能：更新玩家的可编辑资料

### 3.2 匹配系统扩展

在 `internal/match/handler.go` 中添加：

1. **匹配历史查询**：
   - 路径：`/match/history/{player_id}`
   - 方法：GET
   - 功能：查询玩家的匹配历史

2. **匹配偏好设置**：
   - 路径：`/match/preferences/{player_id}`
   - 方法：POST
   - 功能：设置玩家的匹配偏好，如地图偏好、游戏模式偏好等

## 4. 数据初始化

### 4.1 角色和技能数据

创建 `scripts/init_data.go` 脚本，用于初始化以下数据：

1. 默认角色（至少3-5个不同类型的角色）
2. 每个角色的默认技能
3. 游戏地图数据

### 4.2 测试账号

创建测试账号和相关数据，用于开发和测试。

## 5. 实现步骤

1. 创建数据模型和数据库表
2. 实现API端点
3. 编写数据初始化脚本
4. 进行API测试
5. 与客户端集成测试

## 6. API文档

所有API应提供详细的文档，包括：
- 请求方法和URL
- 请求参数和格式
- 响应格式和示例
- 错误码和处理

## 7. 安全考虑

1. 所有API应验证用户身份和权限
2. 敏感数据应加密存储
3. 实现请求频率限制，防止滥用

## 8. 性能优化

1. 为常用查询添加数据库索引
2. 实现缓存机制，特别是排行榜等高频访问数据
3. 对大量数据的查询实现分页 