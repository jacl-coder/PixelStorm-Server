package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	skipData := flag.Bool("skip-data", false, "跳过测试数据初始化")
	flag.Parse()

	log.Println("🎮 PixelStorm 数据库完整设置")
	log.Println("================================")

	// 步骤1: 重置数据库
	log.Println("📋 步骤 1/3: 重置数据库...")
	if err := runCommand("go", "run", "scripts/db_manager.go", "-action=reset", "-config="+*configPath); err != nil {
		log.Fatalf("重置数据库失败: %v", err)
	}

	// 步骤2: 初始化数据库表结构
	log.Println("📋 步骤 2/3: 初始化数据库表结构...")
	if err := runCommand("go", "run", "scripts/db_manager.go", "-action=init", "-config="+*configPath); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 步骤3: 初始化测试数据（可选）
	if !*skipData {
		log.Println("📋 步骤 3/3: 初始化测试数据...")
		if err := runCommand("go", "run", "scripts/init_data.go", "-config="+*configPath, "-type=all"); err != nil {
			log.Fatalf("初始化测试数据失败: %v", err)
		}
	} else {
		log.Println("📋 步骤 3/3: 跳过测试数据初始化")
	}

	log.Println("")
	log.Println("🎉 数据库设置完成！")
	log.Println("")
	log.Println("📊 数据库状态:")
	log.Println("  ✅ 表结构已创建")
	if !*skipData {
		log.Println("  ✅ 测试数据已初始化")
		log.Println("     - 5个默认角色")
		log.Println("     - 4个游戏地图")
		log.Println("     - 3个测试账号")
	} else {
		log.Println("  ⏭️  测试数据已跳过")
	}
	log.Println("")
	log.Println("🚀 现在可以启动服务器:")
	log.Println("  go run cmd/server/main.go")
}

// runCommand 运行命令
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
