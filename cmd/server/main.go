// main.go

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacl-coder/PixelStorm-Server/config"
	"github.com/jacl-coder/PixelStorm-Server/internal/game"
	"github.com/jacl-coder/PixelStorm-Server/internal/gateway"
	"github.com/jacl-coder/PixelStorm-Server/internal/match"
	"github.com/jacl-coder/PixelStorm-Server/pkg/db"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config/config.yaml", "配置文件路径")
	serviceType := flag.String("service", "all", "服务类型 (game, match, gateway, all)")
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

	// 初始化Redis连接
	if err := db.InitRedis(); err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}
	defer db.CloseRedis()



	// 根据服务类型启动不同的服务
	switch *serviceType {
	case "game":
		startGameServer()
	case "match":
		startMatchServer()
	case "gateway":
		startGatewayServer()
	case "all":
		startAllServices()
	default:
		log.Fatalf("未知的服务类型: %s", *serviceType)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("接收到关闭信号，正在关闭服务器...")

	log.Println("服务器已安全关闭")
}

// startGameServer 启动游戏服务器
func startGameServer() {
	// 创建游戏服务器
	server := game.NewGameServer(&config.GlobalConfig)

	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("启动游戏服务器失败: %v", err)
	}

	log.Println("游戏服务器已启动")
}

// startMatchServer 启动匹配服务器
func startMatchServer() {
	// 创建游戏服务器（匹配服务需要游戏服务器引用）
	gameServer := game.NewGameServer(&config.GlobalConfig)

	// 创建匹配服务
	matchService := match.NewMatchService(&config.GlobalConfig, gameServer)

	// 启动匹配服务
	if err := matchService.Start(); err != nil {
		log.Fatalf("启动匹配服务失败: %v", err)
	}

	log.Println("匹配服务已启动")
}

// startGatewayServer 启动网关服务器
func startGatewayServer() {
	// 创建网关服务
	gatewayServer := gateway.NewGateway(&config.GlobalConfig)

	// 启动网关服务
	if err := gatewayServer.Start(); err != nil {
		log.Fatalf("启动网关服务失败: %v", err)
	}

	log.Println("网关服务已启动")
}

// startAllServices 启动所有服务
func startAllServices() {
	// 创建游戏服务器
	gameServer := game.NewGameServer(&config.GlobalConfig)

	// 启动游戏服务器
	if err := gameServer.Start(); err != nil {
		log.Fatalf("启动游戏服务器失败: %v", err)
	}

	// 创建匹配服务
	matchService := match.NewMatchService(&config.GlobalConfig, gameServer)

	// 启动匹配服务
	if err := matchService.Start(); err != nil {
		log.Fatalf("启动匹配服务失败: %v", err)
	}

	// 创建网关服务
	gatewayServer := gateway.NewGateway(&config.GlobalConfig)

	// 启动网关服务
	if err := gatewayServer.Start(); err != nil {
		log.Fatalf("启动网关服务失败: %v", err)
	}

	log.Println("所有服务已启动")
}
