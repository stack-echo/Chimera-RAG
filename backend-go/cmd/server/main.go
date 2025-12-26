package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "Chimera-RAG/api/rag/v1"
	"Chimera-RAG/backend-go/internal/data"
	"Chimera-RAG/backend-go/internal/handler"
	"Chimera-RAG/backend-go/internal/service"
)

func main() {
	// 1. 初始化基础设施
	// 注意：生产环境这里应该用 Config 配置地址
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("无法连接 Chimera 大脑: %v", err)
	}
	defer conn.Close()

	dataClient := data.NewData()

	// 2. 依赖注入 (DI)
	// Client -> Service -> Handler
	grpcClient := pb.NewLLMServiceClient(conn)
	ragService := service.NewRagService(grpcClient, dataClient)
	chatHandler := handler.NewChatHandler(ragService)

	// 3. 初始化 Gin 引擎
	r := gin.Default()

	// 4. 配置 CORS (跨域)
	// 允许前端 (localhost:3000 等) 访问接口
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 5. 注册路由
	v1 := r.Group("/api/v1")
	{
		v1.POST("/chat/stream", chatHandler.HandleChatSSE)
		v1.POST("/upload", chatHandler.HandleUpload)
	}

	// 6. 启动服务
	log.Println("🚀 Chimera Gateway running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}
