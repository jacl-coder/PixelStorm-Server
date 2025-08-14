// db_manager.go

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
	action := flag.String("action", "help", "操作类型: reset, init, help")
	flag.Parse()

	// 显示帮助信息
	if *action == "help" {
		showHelp()
		return
	}

	// 加载配置
	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库连接
	if err := db.InitPostgres(); err != nil {
		log.Fatalf("初始化PostgreSQL失败: %v", err)
	}
	defer db.Close()

	// 执行操作
	switch *action {
	case "reset":
		resetDatabase()
	case "init":
		initDatabase()
	default:
		log.Fatalf("未知操作: %s", *action)
	}
}

// showHelp 显示帮助信息
func showHelp() {
	log.Println("PixelStorm 数据库管理工具")
	log.Println("")
	log.Println("用法:")
	log.Println("  go run scripts/db_manager.go -action=<操作> [-config=<配置文件>]")
	log.Println("")
	log.Println("操作:")
	log.Println("  reset  - 重置数据库（删除所有表和数据）")
	log.Println("  init   - 初始化数据库（创建表结构）")
	log.Println("  help   - 显示此帮助信息")
	log.Println("")
	log.Println("示例:")
	log.Println("  go run scripts/db_manager.go -action=reset")
	log.Println("  go run scripts/db_manager.go -action=init")
	log.Println("  go run scripts/db_manager.go -action=reset && go run scripts/db_manager.go -action=init")
}

// resetDatabase 重置数据库
func resetDatabase() {
	log.Println("⚠️  正在重置数据库...")
	log.Println("⚠️  这将删除所有表和数据！")

	// 删除所有表和视图的SQL
	resetSQL := `
-- 删除视图
DROP VIEW IF EXISTS leaderboard CASCADE;

-- 删除表（按依赖关系顺序）
DROP TABLE IF EXISTS player_match_preferences CASCADE;
DROP TABLE IF EXISTS match_history CASCADE;
DROP TABLE IF EXISTS player_match_records CASCADE;
DROP TABLE IF EXISTS match_records CASCADE;
DROP TABLE IF EXISTS map_modes CASCADE;
DROP TABLE IF EXISTS game_maps CASCADE;
DROP TABLE IF EXISTS player_default_characters CASCADE;
DROP TABLE IF EXISTS player_characters CASCADE;
DROP TABLE IF EXISTS character_skills CASCADE;
DROP TABLE IF EXISTS skills CASCADE;
DROP TABLE IF EXISTS characters CASCADE;
DROP TABLE IF EXISTS players CASCADE;
`

	_, err := db.DB.Exec(resetSQL)
	if err != nil {
		log.Fatalf("重置数据库失败: %v", err)
	}

	log.Println("✅ 数据库重置完成")
}

// initDatabase 初始化数据库
func initDatabase() {
	log.Println("🚀 正在初始化数据库...")

	// 使用统一的表结构创建所有表
	if err := db.InitAllTables(); err != nil {
		log.Fatalf("初始化数据库表失败: %v", err)
	}

	log.Println("✅ 数据库初始化完成")
	log.Println("")
	log.Println("📋 已创建的表:")
	log.Println("  - players (玩家表)")
	log.Println("  - characters (角色表)")
	log.Println("  - skills (技能表)")
	log.Println("  - character_skills (角色技能关联表)")
	log.Println("  - player_characters (玩家角色关系表)")
	log.Println("  - player_default_characters (玩家默认角色表)")
	log.Println("  - game_maps (游戏地图表)")
	log.Println("  - map_modes (地图模式关联表)")
	log.Println("  - match_records (对局记录表)")
	log.Println("  - player_match_records (玩家对局记录表)")
	log.Println("  - player_match_preferences (玩家匹配偏好表)")
	log.Println("  - match_history (匹配历史表)")
	log.Println("  - leaderboard (排行榜视图)")
	log.Println("")
	log.Println("💡 提示: 使用以下命令初始化测试数据:")
	log.Println("  go run scripts/init_data.go -config=config/config.yaml -type=all")
}
