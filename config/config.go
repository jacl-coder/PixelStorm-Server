// config.go

package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 服务器配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// ServerConfig 服务器基本配置
type ServerConfig struct {
	GamePort     int    `mapstructure:"game_port"`
	MatchPort    int    `mapstructure:"match_port"`
	GatewayPort  int    `mapstructure:"gateway_port"`
	Debug        bool   `mapstructure:"debug"`
	LogLevel     string `mapstructure:"log_level"`
	MaxRoomCount int    `mapstructure:"max_room_count"`
	MaxPlayers   int    `mapstructure:"max_players"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

var (
	// GlobalConfig 全局配置实例
	GlobalConfig Config
)

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("无法读取配置文件: %w", err)
	}

	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("无法解析配置文件: %w", err)
	}

	return nil
}

// GetDSN 获取PostgreSQL连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// GetRedisAddr 获取Redis连接地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
