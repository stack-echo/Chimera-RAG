package service

import (
	"context"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"

	pb "Chimera-RAG/api/rag/v1"
	"Chimera-RAG/backend-go/internal/data"

	"github.com/minio/minio-go/v7"
)

// RagService 定义业务逻辑
type RagService struct {
	grpcClient pb.LLMServiceClient
	data       *data.Data
}

// NewRagService 构造函数
func NewRagService(client pb.LLMServiceClient, data *data.Data) *RagService {
	return &RagService{
		grpcClient: client,
		data:       data,
	}
}

// StreamChat 核心逻辑：调用 gRPC 并把结果推到一个 channel 里给 Handler 用
// 返回一个只读 channel，Handler 只需要从里面读字符串即可
func (s *RagService) StreamChat(ctx context.Context, req *pb.AskRequest) (<-chan string, error) {
	// 1. 调用 Python gRPC
	stream, err := s.grpcClient.AskStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// 2. 创建一个管道，用于把 gRPC 的数据“搬运”给 HTTP
	// 使用带缓冲的 channel 防止阻塞
	respChan := make(chan string, 10)

	// 3. 启动协程后台搬运
	go func() {
		defer close(respChan) // 搬运结束关闭管道

		for {
			// 从 Python 收数据
			resp, err := stream.Recv()
			if err == io.EOF {
				return // 流结束
			}
			if err != nil {
				log.Printf("gRPC Recv error: %v", err)
				respChan <- "ERR: " + err.Error() // 简单处理错误
				return
			}

			// 处理业务逻辑：这里可以将 ThinkingLog 和 Answer 拼成特定格式给前端
			// 或者通过 SSE 的 event type 区分
			// 这里演示最简单的：直接发 JSON 字符串给前端解析，或者简单拼接

			// 场景 A: 发送思考过程
			if resp.ThinkingLog != "" {
				respChan <- "THINKing: " + resp.ThinkingLog
			}

			// 场景 B: 发送答案
			if resp.AnswerDelta != "" {
				respChan <- "ANSWER: " + resp.AnswerDelta
			}
		}
	}()

	return respChan, nil
}

// UploadDocument 处理文件上传业务
func (s *RagService) UploadDocument(ctx context.Context, file *multipart.FileHeader) (string, error) {
	// 1. 打开文件流
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// 2. 生成对象名 (防止重名，这里简单用文件名，生产环境建议用 UUID)
	objectName := filepath.Base(file.Filename)
	bucketName := "chimera-docs"

	// 3. 流式上传到 MinIO (核心亮点：内存占用极低)
	info, err := s.data.Minio.PutObject(ctx, bucketName, objectName, src, file.Size, minio.PutObjectOptions{
		ContentType: "application/pdf", // 假设传的是 PDF
	})
	if err != nil {
		log.Printf("MinIO 上传失败: %v", err)
		return "", err
	}

	log.Printf("文件已存入 MinIO: %s (Size: %d)", objectName, info.Size)

	// 4. 写入 Redis 任务队列 (异步解耦)
	// 将文件名推送到 "task:parse_pdf" 队列中
	err = s.data.Redis.LPush(ctx, "task:parse_pdf", objectName).Err()
	if err != nil {
		log.Printf("Redis 推送失败: %v", err)
		return "", err
	}

	return objectName, nil
}
