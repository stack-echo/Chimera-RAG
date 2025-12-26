package data

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

// Data 结构体持有所有数据库句柄
type Data struct {
	Minio *minio.Client
	Redis *redis.Client
}

func NewData() *Data {
	// 1. 初始化 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Docker 端口
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Redis 连接失败: %v", err)
	}

	// 2. 初始化 MinIO
	// 注意：生产环境 endpoint 不带 http
	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false, // 本地 Docker 没有 HTTPS
	})
	if err != nil {
		log.Fatalf("MinIO 初始化失败: %v", err)
	}

	// 自动创建 Bucket
	bucketName := "chimera-docs"
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Fatalf("检查 Bucket 失败: %v", err)
	}
	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Fatalf("创建 Bucket 失败: %v", err)
		}
		log.Printf("🎉 MinIO Bucket '%s' 创建成功", bucketName)
	}

	return &Data{Minio: minioClient, Redis: rdb}
}
