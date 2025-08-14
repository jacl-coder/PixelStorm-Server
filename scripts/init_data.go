// init_data.go

package main

import (
	"flag"
	"log"

	"github.com/jacl-coder/PixelStorm-Server/config"
	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	dataType := flag.String("type", "all", "初始化数据类型 (characters, maps, accounts, all)")
	flag.Parse()

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接
	if err := db.InitPostgres(); err != nil {
		log.Fatalf("初始化PostgreSQL失败: %v", err)
	}
	defer db.Close()

	// 初始化数据库表
	if err := db.InitAllTables(); err != nil {
		log.Fatalf("初始化数据库表失败: %v", err)
	}
	log.Println("✓ 数据库表初始化完成")

	// 根据类型初始化数据
	switch *dataType {
	case "characters":
		if err := initCharacterData(); err != nil {
			log.Fatalf("初始化角色数据失败: %v", err)
		}
		log.Println("角色数据初始化完成")
	case "maps":
		if err := initMapData(); err != nil {
			log.Fatalf("初始化地图数据失败: %v", err)
		}
		log.Println("地图数据初始化完成")
	case "accounts":
		if err := initTestAccounts(); err != nil {
			log.Fatalf("初始化测试账号失败: %v", err)
		}
		log.Println("测试账号初始化完成")
	case "all":
		log.Println("开始初始化所有数据...")
		
		if err := initCharacterData(); err != nil {
			log.Fatalf("初始化角色数据失败: %v", err)
		}
		log.Println("✓ 角色数据初始化完成")

		if err := initMapData(); err != nil {
			log.Fatalf("初始化地图数据失败: %v", err)
		}
		log.Println("✓ 地图数据初始化完成")

		if err := initTestAccounts(); err != nil {
			log.Fatalf("初始化测试账号失败: %v", err)
		}
		log.Println("✓ 测试账号初始化完成")

		log.Println("🎉 所有数据初始化完成！")
	default:
		log.Fatalf("未知的数据类型: %s", *dataType)
	}
}

// initCharacterData 初始化角色数据
func initCharacterData() error {
	log.Println("正在初始化角色数据...")

	// 检查是否已有角色数据
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM characters").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("角色表已有 %d 条数据，跳过初始化", count)
		return nil
	}

	// 插入默认角色数据
	characters := []struct {
		name         string
		description  string
		maxHP        int
		speed        float64
		baseAttack   int
		baseDefense  int
		specialAbility string
		difficulty   int
		role         string
		unlockable   bool
		unlockCost   int
	}{
		{
			name:         "突击兵",
			description:  "平衡型角色，适合新手使用。拥有良好的攻击力和生存能力。",
			maxHP:        100,
			speed:        5.0,
			baseAttack:   20,
			baseDefense:  15,
			specialAbility: "快速冲刺",
			difficulty:   1,
			role:         "攻击手",
			unlockable:   false,
			unlockCost:   0,
		},
		{
			name:         "狙击手",
			description:  "远程输出专家，拥有超远射程和高伤害，但血量较低。",
			maxHP:        80,
			speed:        4.0,
			baseAttack:   35,
			baseDefense:  10,
			specialAbility: "精准射击",
			difficulty:   3,
			role:         "射手",
			unlockable:   true,
			unlockCost:   1000,
		},
		{
			name:         "重装兵",
			description:  "坦克型角色，拥有超高血量和防御力，但移动速度较慢。",
			maxHP:        150,
			speed:        3.0,
			baseAttack:   15,
			baseDefense:  25,
			specialAbility: "护盾展开",
			difficulty:   2,
			role:         "坦克",
			unlockable:   true,
			unlockCost:   800,
		},
		{
			name:         "医疗兵",
			description:  "支援型角色，可以治疗队友并提供增益效果。",
			maxHP:        90,
			speed:        4.5,
			baseAttack:   12,
			baseDefense:  12,
			specialAbility: "治疗光环",
			difficulty:   2,
			role:         "辅助",
			unlockable:   true,
			unlockCost:   1200,
		},
		{
			name:         "刺客",
			description:  "高机动性角色，拥有极高的爆发伤害和移动速度。",
			maxHP:        70,
			speed:        6.0,
			baseAttack:   30,
			baseDefense:  8,
			specialAbility: "隐身突袭",
			difficulty:   4,
			role:         "刺客",
			unlockable:   true,
			unlockCost:   1500,
		},
	}

	// 插入角色数据
	for _, char := range characters {
		_, err := db.DB.Exec(`
			INSERT INTO characters (name, description, max_hp, speed, base_attack, base_defense, 
			                       special_ability, difficulty, role, unlockable, unlock_cost)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, char.name, char.description, char.maxHP, char.speed, char.baseAttack, char.baseDefense,
			char.specialAbility, char.difficulty, char.role, char.unlockable, char.unlockCost)
		
		if err != nil {
			return err
		}
		log.Printf("✓ 插入角色: %s", char.name)
	}

	// 初始化技能数据
	if err := initSkillData(); err != nil {
		return err
	}

	return nil
}

// initSkillData 初始化技能数据
func initSkillData() error {
	log.Println("正在初始化技能数据...")

	// 检查是否已有技能数据
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM skills").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("技能表已有 %d 条数据，跳过初始化", count)
		return nil
	}

	// 插入默认技能数据
	skills := []struct {
		name            string
		description     string
		skillType       string
		damage          int
		cooldownTime    float64
		range_          float64
		effectTime      float64
		projectileSpeed float64
		projectileCount int
		animationKey    string
		effectKey       string
	}{
		{
			name:            "普通射击",
			description:     "基础射击技能，发射单发子弹",
			skillType:       "projectile",
			damage:          10,
			cooldownTime:    0.5,
			range_:          500,
			effectTime:      0,
			projectileSpeed: 800,
			projectileCount: 1,
			animationKey:    "shoot_basic",
			effectKey:       "bullet_basic",
		},
		{
			name:            "散射",
			description:     "发射多发子弹，覆盖更大范围",
			skillType:       "projectile",
			damage:          8,
			cooldownTime:    3.0,
			range_:          400,
			effectTime:      0,
			projectileSpeed: 700,
			projectileCount: 3,
			animationKey:    "shoot_scatter",
			effectKey:       "bullet_scatter",
		},
		{
			name:            "穿透弹",
			description:     "发射穿透子弹，可击中多个敌人",
			skillType:       "projectile",
			damage:          15,
			cooldownTime:    5.0,
			range_:          600,
			effectTime:      0,
			projectileSpeed: 900,
			projectileCount: 1,
			animationKey:    "shoot_pierce",
			effectKey:       "bullet_pierce",
		},
		{
			name:            "治疗",
			description:     "恢复自己或队友的生命值",
			skillType:       "buff",
			damage:          -20, // 负数表示治疗
			cooldownTime:    8.0,
			range_:          200,
			effectTime:      1.0,
			projectileSpeed: 0,
			projectileCount: 0,
			animationKey:    "heal",
			effectKey:       "heal_effect",
		},
		{
			name:            "冲刺",
			description:     "快速向前冲刺一段距离",
			skillType:       "movement",
			damage:          0,
			cooldownTime:    6.0,
			range_:          300,
			effectTime:      0.5,
			projectileSpeed: 0,
			projectileCount: 0,
			animationKey:    "dash",
			effectKey:       "dash_effect",
		},
	}

	// 插入技能数据
	for _, skill := range skills {
		_, err := db.DB.Exec(`
			INSERT INTO skills (name, description, type, damage, cooldown_time, range, effect_time,
			                   projectile_speed, projectile_count, animation_key, effect_key)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, skill.name, skill.description, skill.skillType, skill.damage, skill.cooldownTime,
			skill.range_, skill.effectTime, skill.projectileSpeed, skill.projectileCount,
			skill.animationKey, skill.effectKey)
		
		if err != nil {
			return err
		}
		log.Printf("✓ 插入技能: %s", skill.name)
	}

	// 关联角色和技能
	if err := initCharacterSkills(); err != nil {
		return err
	}

	return nil
}

// initCharacterSkills 初始化角色技能关联
func initCharacterSkills() error {
	log.Println("正在关联角色和技能...")

	// 角色技能关联配置
	characterSkills := []struct {
		characterName string
		skillNames    []string
	}{
		{
			characterName: "突击兵",
			skillNames:    []string{"普通射击", "散射", "冲刺"},
		},
		{
			characterName: "狙击手",
			skillNames:    []string{"普通射击", "穿透弹"},
		},
		{
			characterName: "重装兵",
			skillNames:    []string{"普通射击", "散射"},
		},
		{
			characterName: "医疗兵",
			skillNames:    []string{"普通射击", "治疗"},
		},
		{
			characterName: "刺客",
			skillNames:    []string{"普通射击", "冲刺"},
		},
	}

	for _, cs := range characterSkills {
		// 获取角色ID
		var characterID int
		err := db.DB.QueryRow("SELECT id FROM characters WHERE name = $1", cs.characterName).Scan(&characterID)
		if err != nil {
			return err
		}

		// 关联技能
		for slotIndex, skillName := range cs.skillNames {
			var skillID int
			err := db.DB.QueryRow("SELECT id FROM skills WHERE name = $1", skillName).Scan(&skillID)
			if err != nil {
				return err
			}

			_, err = db.DB.Exec(`
				INSERT INTO character_skills (character_id, skill_id, slot_index)
				VALUES ($1, $2, $3)
			`, characterID, skillID, slotIndex)

			if err != nil {
				return err
			}
		}
		log.Printf("✓ 关联角色 %s 的技能", cs.characterName)
	}

	return nil
}

// initMapData 初始化地图数据
func initMapData() error {
	log.Println("正在初始化地图数据...")

	// 检查是否已有地图数据
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM game_maps").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("地图表已有 %d 条数据，跳过初始化", count)
		return nil
	}

	// 插入默认地图数据
	maps := []struct {
		name           string
		description    string
		imagePath      string
		width          int
		height         int
		maxPlayers     int
		supportedModes []string
	}{
		{
			name:           "城市废墟",
			description:    "被战争摧毁的城市，到处都是废墟和掩体",
			imagePath:      "/maps/city_ruins.jpg",
			width:          1000,
			height:         1000,
			maxPlayers:     8,
			supportedModes: []string{"deathmatch", "team_deathmatch"},
		},
		{
			name:           "沙漠基地",
			description:    "炎热的沙漠中的军事基地",
			imagePath:      "/maps/desert_base.jpg",
			width:          1200,
			height:         800,
			maxPlayers:     10,
			supportedModes: []string{"deathmatch", "team_deathmatch", "flag_capture"},
		},
		{
			name:           "森林小径",
			description:    "茂密森林中的蜿蜒小径",
			imagePath:      "/maps/forest_path.jpg",
			width:          800,
			height:         1200,
			maxPlayers:     6,
			supportedModes: []string{"deathmatch"},
		},
		{
			name:           "工业区",
			description:    "充满管道和机械的工业区域",
			imagePath:      "/maps/industrial.jpg",
			width:          1000,
			height:         1000,
			maxPlayers:     8,
			supportedModes: []string{"team_deathmatch", "flag_capture"},
		},
	}

	// 插入地图数据
	for _, gameMap := range maps {
		// 插入地图基本信息
		var mapID int
		err := db.DB.QueryRow(`
			INSERT INTO game_maps (name, description, image_path, width, height, max_players)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, gameMap.name, gameMap.description, gameMap.imagePath, gameMap.width, gameMap.height,
			gameMap.maxPlayers).Scan(&mapID)

		if err != nil {
			return err
		}

		// 插入支持的游戏模式
		for _, mode := range gameMap.supportedModes {
			_, err := db.DB.Exec(`
				INSERT INTO map_modes (map_id, mode)
				VALUES ($1, $2)
			`, mapID, mode)

			if err != nil {
				return err
			}
		}

		log.Printf("✓ 插入地图: %s (支持 %d 种模式)", gameMap.name, len(gameMap.supportedModes))
	}

	return nil
}

// initTestAccounts 初始化测试账号
func initTestAccounts() error {
	log.Println("正在初始化测试账号...")

	// 检查是否已有测试账号
	var count int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM players WHERE username LIKE 'test%'").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("已有 %d 个测试账号，跳过初始化", count)
		return nil
	}

	// 创建测试账号
	testAccounts := []struct {
		username string
		password string
		email    string
		level    int
		exp      int64
		coins    int64
		gems     int64
	}{
		{
			username: "testuser1",
			password: "password123", // 实际应用中应该加密
			email:    "test1@pixelstorm.com",
			level:    5,
			exp:      2500,
			coins:    5000,
			gems:     100,
		},
		{
			username: "testuser2",
			password: "password123",
			email:    "test2@pixelstorm.com",
			level:    10,
			exp:      8000,
			coins:    12000,
			gems:     250,
		},
		{
			username: "testuser3",
			password: "password123",
			email:    "test3@pixelstorm.com",
			level:    1,
			exp:      0,
			coins:    1000,
			gems:     50,
		},
	}

	// 插入测试账号
	for _, account := range testAccounts {
		// 简单的密码哈希（实际应用中应使用更安全的方法）
		hashedPassword := hashPassword(account.password)

		_, err := db.DB.Exec(`
			INSERT INTO players (username, password, email, level, exp, coins, gems, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, account.username, hashedPassword, account.email, account.level, account.exp, account.coins, account.gems)

		if err != nil {
			return err
		}
		log.Printf("✓ 创建测试账号: %s", account.username)
	}

	// 为测试账号分配默认角色
	if err := assignDefaultCharacters(); err != nil {
		return err
	}

	return nil
}

// assignDefaultCharacters 为测试账号分配默认角色
func assignDefaultCharacters() error {
	log.Println("正在为测试账号分配角色...")

	// 获取所有测试账号
	rows, err := db.DB.Query("SELECT id FROM players WHERE username LIKE 'test%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var playerIDs []int64
	for rows.Next() {
		var playerID int64
		if err := rows.Scan(&playerID); err != nil {
			return err
		}
		playerIDs = append(playerIDs, playerID)
	}

	// 获取突击兵角色ID（默认角色）
	var defaultCharacterID int
	err = db.DB.QueryRow("SELECT id FROM characters WHERE name = '突击兵'").Scan(&defaultCharacterID)
	if err != nil {
		return err
	}

	// 为每个测试账号分配角色
	for _, playerID := range playerIDs {
		// 分配突击兵角色
		_, err = db.DB.Exec(`
			INSERT INTO player_characters (player_id, character_id, unlocked, unlocked_at)
			VALUES ($1, $2, true, NOW())
		`, playerID, defaultCharacterID)
		if err != nil {
			return err
		}

		// 设置为默认角色
		_, err = db.DB.Exec(`
			INSERT INTO player_default_characters (player_id, character_id)
			VALUES ($1, $2)
		`, playerID, defaultCharacterID)
		if err != nil {
			return err
		}
	}

	log.Printf("✓ 为 %d 个测试账号分配了默认角色", len(playerIDs))
	return nil
}

// hashPassword 简单的密码哈希函数（实际应用中应使用更安全的方法）
func hashPassword(password string) string {
	// 这里使用简单的方法，实际应用中应使用 bcrypt 等安全的哈希算法
	return "hashed_" + password
}
