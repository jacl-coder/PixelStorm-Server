// schema.go

package db

// 统一的数据库表结构定义

// CreateAllTablesSQL 创建所有表的SQL语句
const CreateAllTablesSQL = `
-- 玩家表
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- 玩家等级和经验
    level INT DEFAULT 1,
    exp BIGINT DEFAULT 0,
    coins BIGINT DEFAULT 0,
    gems BIGINT DEFAULT 0,
    
    -- 战绩统计
    total_kills INT DEFAULT 0,
    total_deaths INT DEFAULT 0,
    total_assists INT DEFAULT 0,
    total_matches INT DEFAULT 0,
    total_wins INT DEFAULT 0
);

-- 角色表
CREATE TABLE IF NOT EXISTS characters (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    max_hp INT NOT NULL,
    speed DECIMAL(5,2) NOT NULL,
    base_attack INT NOT NULL,
    base_defense INT NOT NULL,
    special_ability VARCHAR(100),
    difficulty INT DEFAULT 1,
    role VARCHAR(20),
    unlockable BOOLEAN DEFAULT true,
    unlock_cost INT DEFAULT 0
);

-- 技能表
CREATE TABLE IF NOT EXISTS skills (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    type VARCHAR(20) NOT NULL,
    damage INT DEFAULT 0,
    cooldown_time DECIMAL(5,2) DEFAULT 0,
    range DECIMAL(8,2) DEFAULT 0,
    effect_time DECIMAL(5,2) DEFAULT 0,
    projectile_speed DECIMAL(8,2) DEFAULT 0,
    projectile_count INT DEFAULT 0,
    projectile_spread DECIMAL(5,2) DEFAULT 0,
    animation_key VARCHAR(50),
    effect_key VARCHAR(50)
);

-- 角色技能关联表
CREATE TABLE IF NOT EXISTS character_skills (
    character_id INT REFERENCES characters(id) ON DELETE CASCADE,
    skill_id INT REFERENCES skills(id) ON DELETE CASCADE,
    slot_index INT NOT NULL,
    PRIMARY KEY (character_id, skill_id)
);

-- 玩家角色关系表
CREATE TABLE IF NOT EXISTS player_characters (
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE,
    character_id INT REFERENCES characters(id) ON DELETE CASCADE,
    unlocked BOOLEAN DEFAULT false,
    unlocked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, character_id)
);

-- 玩家默认角色表
CREATE TABLE IF NOT EXISTS player_default_characters (
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE UNIQUE,
    character_id INT REFERENCES characters(id) ON DELETE CASCADE,
    PRIMARY KEY (player_id)
);

-- 游戏地图表
CREATE TABLE IF NOT EXISTS game_maps (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    image_path VARCHAR(200),
    width INT NOT NULL,
    height INT NOT NULL,
    max_players INT NOT NULL
);

-- 地图支持的游戏模式表
CREATE TABLE IF NOT EXISTS map_modes (
    map_id INT REFERENCES game_maps(id) ON DELETE CASCADE,
    mode VARCHAR(20) NOT NULL,
    PRIMARY KEY (map_id, mode)
);

-- 对局记录表
CREATE TABLE IF NOT EXISTS match_records (
    id VARCHAR(50) PRIMARY KEY,
    game_mode VARCHAR(20) NOT NULL,
    map_id INT REFERENCES game_maps(id),
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'waiting',
    max_players INT NOT NULL,
    current_players INT DEFAULT 0
);

-- 玩家对局记录表
CREATE TABLE IF NOT EXISTS player_match_records (
    match_id VARCHAR(50) REFERENCES match_records(id) ON DELETE CASCADE,
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE,
    character_id INT REFERENCES characters(id),
    team INT,
    score INT DEFAULT 0,
    kills INT DEFAULT 0,
    deaths INT DEFAULT 0,
    assists INT DEFAULT 0,
    exp_gained INT DEFAULT 0,
    coins_gained INT DEFAULT 0,
    mvp BOOLEAN DEFAULT false,
    play_time INT DEFAULT 0,
    join_time TIMESTAMP WITH TIME ZONE NOT NULL,
    leave_time TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (match_id, player_id)
);

-- 玩家匹配偏好表
CREATE TABLE IF NOT EXISTS player_match_preferences (
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE PRIMARY KEY,
    preferred_modes TEXT[], -- 偏好的游戏模式数组
    preferred_maps INT[], -- 偏好的地图ID数组
    max_wait_time INT DEFAULT 300, -- 最大等待时间(秒)
    skill_level VARCHAR(20) DEFAULT 'intermediate', -- 技能等级
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 匹配历史表
CREATE TABLE IF NOT EXISTS match_history (
    id SERIAL PRIMARY KEY,
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE,
    match_id VARCHAR(50),
    game_mode VARCHAR(20) NOT NULL,
    join_time TIMESTAMP WITH TIME ZONE NOT NULL,
    match_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL, -- waiting, matched, cancelled
    wait_time INT DEFAULT 0 -- 等待时间(秒)
);

-- 创建排行榜视图
CREATE OR REPLACE VIEW leaderboard AS
SELECT 
    p.id AS player_id,
    p.username,
    p.level,
    p.total_kills,
    p.total_matches,
    p.total_wins,
    CASE WHEN p.total_matches > 0 THEN (p.total_wins * 100.0 / p.total_matches) ELSE 0 END AS win_rate,
    CASE WHEN p.total_deaths > 0 THEN ((p.total_kills + p.total_assists) * 1.0 / p.total_deaths)
         ELSE (p.total_kills + p.total_assists) END AS kda,
    (p.total_wins * 10 + p.total_kills + p.total_assists * 0.5 - p.total_deaths * 0.5) AS score
FROM 
    players p
ORDER BY 
    score DESC;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_players_username ON players(username);
CREATE INDEX IF NOT EXISTS idx_players_email ON players(email);
CREATE INDEX IF NOT EXISTS idx_player_characters_player_id ON player_characters(player_id);
CREATE INDEX IF NOT EXISTS idx_player_match_records_player_id ON player_match_records(player_id);
CREATE INDEX IF NOT EXISTS idx_player_match_records_match_id ON player_match_records(match_id);
CREATE INDEX IF NOT EXISTS idx_match_records_game_mode ON match_records(game_mode);
CREATE INDEX IF NOT EXISTS idx_match_records_status ON match_records(status);
CREATE INDEX IF NOT EXISTS idx_match_history_player_id ON match_history(player_id);
CREATE INDEX IF NOT EXISTS idx_character_skills_character_id ON character_skills(character_id);
`

// InitAllTables 初始化所有数据库表
func InitAllTables() error {
	_, err := DB.Exec(CreateAllTablesSQL)
	if err != nil {
		return err
	}
	return nil
}
