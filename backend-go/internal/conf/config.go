package conf

import (
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	App  AppConfig
	Data DataConfig
	AI   AIConfig
}

type AppConfig struct {
	Port string
}

type DataConfig struct {
	RedisAddr      string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	QdrantAddr     string
}

type AIConfig struct {
	GRPCHost string
}

func LoadConfig() *Config {
	v := viper.New()

	// 1. 设置默认值 (开发环境兜底)
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("DATA_REDIS_ADDR", "localhost:6379")
	v.SetDefault("DATA_MINIO_ENDPOINT", "localhost:9000")
	v.SetDefault("DATA_MINIO_AK", "minioadmin") // 默认值
	v.SetDefault("DATA_MINIO_SK", "minioadmin")
	v.SetDefault("DATA_QDRANT_ADDR", "localhost:6334")
	v.SetDefault("AI_GRPC_HOST", "localhost:50051")

	// 2. 允许读取环境变量 (自动将 . 转换为 _)
	v.AutomaticEnv()

	// 3. 读取本地 .env 文件 (可选，方便本地调试)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	_ = v.ReadInConfig() // 忽略错误，因为生产环境可能只传环境变量

	var c Config
	// 手动映射或使用 viper 的 Unmarshal 功能
	// 这里为了简单直接读取
	c.App.Port = v.GetString("APP_PORT")
	c.Data.RedisAddr = v.GetString("DATA_REDIS_ADDR")
	c.Data.MinioEndpoint = v.GetString("DATA_MINIO_ENDPOINT")
	c.Data.MinioAccessKey = v.GetString("DATA_MINIO_AK")
	c.Data.MinioSecretKey = v.GetString("DATA_MINIO_SK")
	c.Data.QdrantAddr = v.GetString("DATA_QDRANT_ADDR")
	c.AI.GRPCHost = v.GetString("AI_GRPC_HOST")

	log.Println("✅ 配置加载完成")
	return &c
}
