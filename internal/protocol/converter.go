package protocol

import (
	"github.com/jacl-coder/PixelStorm-Server/internal/models"
)

// ConvertCharacterToProto 将角色模型转换为协议消息
func ConvertCharacterToProto(character *models.Character) *CharacterInfo {
	skills := make([]*SkillInfo, len(character.Skills))
	for i, skill := range character.Skills {
		skills[i] = ConvertSkillToProto(&skill)
	}

	return &CharacterInfo{
		Id:             int32(character.ID),
		Name:           character.Name,
		Description:    character.Description,
		MaxHp:          int32(character.MaxHP),
		Speed:          float32(character.Speed),
		BaseAttack:     int32(character.BaseAttack),
		BaseDefense:    int32(character.BaseDefense),
		SpecialAbility: character.SpecialAbility,
		Skills:         skills,
		Difficulty:     int32(character.Difficulty),
		Role:           character.Role,
		Unlockable:     character.Unlockable,
		UnlockCost:     int32(character.UnlockCost),
	}
}

// ConvertSkillToProto 将技能模型转换为协议消息
func ConvertSkillToProto(skill *models.Skill) *SkillInfo {
	var skillType SkillType
	switch skill.Type {
	case models.ProjectileSkill:
		skillType = SkillType_SKILL_PROJECTILE
	case models.AOESkill:
		skillType = SkillType_SKILL_AOE
	case models.BuffSkill:
		skillType = SkillType_SKILL_BUFF
	case models.DebuffSkill:
		skillType = SkillType_SKILL_DEBUFF
	case models.MovementSkill:
		skillType = SkillType_SKILL_MOVEMENT
	case models.UtilitySkill:
		skillType = SkillType_SKILL_UTILITY
	default:
		skillType = SkillType_SKILL_PROJECTILE
	}

	return &SkillInfo{
		Id:               int32(skill.ID),
		Name:             skill.Name,
		Description:      skill.Description,
		Type:             skillType,
		Damage:           int32(skill.Damage),
		CooldownTime:     float32(skill.CooldownTime),
		Range:            float32(skill.Range),
		EffectTime:       float32(skill.EffectTime),
		ProjectileSpeed:  float32(skill.ProjectileSpeed),
		ProjectileCount:  int32(skill.ProjectileCount),
		ProjectileSpread: float32(skill.ProjectileSpread),
		AnimationKey:     skill.AnimationKey,
		EffectKey:        skill.EffectKey,
	}
}

// ConvertPlayerCharacterToProto 将玩家角色转换为协议消息
func ConvertPlayerCharacterToProto(pc *models.PlayerCharacter) *PlayerCharacterInfo {
	return &PlayerCharacterInfo{
		PlayerId:    pc.PlayerID,
		CharacterId: int32(pc.CharacterID),
		Level:       int32(pc.Level),
		Exp:         int32(pc.Exp),
		Unlocked:    pc.Unlocked,
		UsageCount:  int32(pc.UsageCount),
		WinCount:    int32(pc.WinCount),
		KillCount:   int32(pc.KillCount),
		DeathCount:  int32(pc.DeathCount),
	}
}

// ConvertPlayerStatsToProto 将玩家战绩转换为协议消息
func ConvertPlayerStatsToProto(stats *models.PlayerStats) *PlayerStats {
	return &PlayerStats{
		PlayerId:     stats.PlayerID,
		TotalMatches: int32(stats.TotalMatches),
		TotalWins:    int32(stats.TotalWins),
		Losses:       int32(stats.Losses),
		WinRate:      float32(stats.WinRate),
		TotalKills:   int32(stats.TotalKills),
		TotalDeaths:  int32(stats.TotalDeaths),
		TotalAssists: int32(stats.TotalAssists),
		Kda:          float32(stats.KDA),
		AverageScore: float32(stats.AverageScore),
		TotalMvp:     int32(stats.TotalMVP),
		PlayTime:     int32(stats.PlayTime),
	}
}

// ConvertMatchRecordToProto 将对局记录转换为协议消息
func ConvertMatchRecordToProto(record *models.MatchRecord) *MatchRecord {
	return &MatchRecord{
		Id:          record.ID,
		GameMode:    string(record.GameMode),
		StartTime:   record.StartTime.Unix(),
		EndTime:     record.EndTime.Unix(),
		WinningTeam: int32(record.WinningTeam),
		MapId:       int32(record.MapID),
		Duration:    int32(record.Duration),
	}
}

// ConvertPlayerMatchRecordToProto 将玩家对局记录转换为协议消息
func ConvertPlayerMatchRecordToProto(record *models.PlayerMatchRecord) *PlayerMatchRecord {
	return &PlayerMatchRecord{
		MatchId:     record.MatchID,
		PlayerId:    record.PlayerID,
		CharacterId: int32(record.CharacterID),
		Team:        int32(record.Team),
		Score:       int32(record.Score),
		Kills:       int32(record.Kills),
		Deaths:      int32(record.Deaths),
		Assists:     int32(record.Assists),
		ExpGained:   int32(record.ExpGained),
		CoinsGained: int32(record.CoinsGained),
		Mvp:         record.MVP,
		PlayTime:    int32(record.PlayTime),
		JoinTime:    record.JoinTime.Unix(),
		LeaveTime:   record.LeaveTime.Unix(),
	}
}

// ConvertLeaderboardEntryToProto 将排行榜条目转换为协议消息
func ConvertLeaderboardEntryToProto(entry *models.LeaderboardEntry) *LeaderboardEntry {
	return &LeaderboardEntry{
		PlayerId:     entry.PlayerID,
		Username:     entry.Username,
		Level:        int32(entry.Level),
		TotalKills:   int32(entry.TotalKills),
		TotalWins:    int32(entry.TotalWins),
		WinRate:      float32(entry.WinRate),
		Kda:          float32(entry.KDA),
		Score:        float32(entry.Score),
		Rank:         int32(entry.Rank),
	}
}

// ConvertGameMapToProto 将游戏地图转换为协议消息
func ConvertGameMapToProto(gameMap *models.GameMap) *GameMapInfo {
	supportedModes := make([]string, len(gameMap.SupportedModes))
	for i, mode := range gameMap.SupportedModes {
		supportedModes[i] = string(mode)
	}

	return &GameMapInfo{
		Id:             int32(gameMap.ID),
		Name:           gameMap.Name,
		Description:    gameMap.Description,
		ImagePath:      gameMap.ImagePath,
		Width:          int32(gameMap.Width),
		Height:         int32(gameMap.Height),
		MaxPlayers:     int32(gameMap.MaxPlayers),
		SupportedModes: supportedModes,
	}
}

// CreateSuccessResponse 创建成功响应
func CreateSuccessResponse(message string) *SuccessResponse {
	return &SuccessResponse{
		Success: true,
		Message: message,
	}
}

// CreateErrorResponse 创建错误响应
func CreateErrorResponse(message, errorCode string) *ErrorResponse {
	return &ErrorResponse{
		Success:   false,
		Message:   message,
		ErrorCode: errorCode,
	}
}

// CreateCharacterListResponse 创建角色列表响应
func CreateCharacterListResponse(characters []*models.Character) *CharacterListResponse {
	protoCharacters := make([]*CharacterInfo, len(characters))
	for i, character := range characters {
		protoCharacters[i] = ConvertCharacterToProto(character)
	}

	return &CharacterListResponse{
		Success: true,
		Message: "查询成功",
		Data:    protoCharacters,
	}
}

// CreatePlayerStatsResponse 创建玩家战绩响应
func CreatePlayerStatsResponse(stats *models.PlayerStats) *PlayerStatsResponse {
	return &PlayerStatsResponse{
		Success: true,
		Message: "查询成功",
		Data:    ConvertPlayerStatsToProto(stats),
	}
}

// CreateLeaderboardResponse 创建排行榜响应
func CreateLeaderboardResponse(entries []*models.LeaderboardEntry, leaderboardType string) *LeaderboardResponse {
	protoEntries := make([]*LeaderboardEntry, len(entries))
	for i, entry := range entries {
		protoEntries[i] = ConvertLeaderboardEntryToProto(entry)
	}

	return &LeaderboardResponse{
		Success:         true,
		Message:         "查询成功",
		Data:            protoEntries,
		LeaderboardType: leaderboardType,
	}
}
