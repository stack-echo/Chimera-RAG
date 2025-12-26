package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"

	pb "Chimera-RAG/backend-go/api/rag/v1"
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

// StreamChat RAG 核心流程
func (s *RagService) StreamChat(ctx context.Context, req *pb.AskRequest) (<-chan string, error) {
	respChan := make(chan string, 10)

	go func() {
		defer close(respChan)

		// 1. 向量化
		respChan <- "THINKing: 正在理解意图..."
		embResp, err := s.grpcClient.EmbedData(ctx, &pb.EmbedRequest{Data: &pb.EmbedRequest_Text{Text: req.Query}})
		if err != nil {
			respChan <- "ERR: " + err.Error()
			return
		}

		// 2. 检索 (Retrieval)
		respChan <- "THINKing: 正在检索知识库..."
		docs, err := s.data.SearchSimilar(ctx, embResp.Vector, 15)
		if err != nil {
			respChan <- "ERR: " + err.Error()
			return
		}

		// 3. 组装 Prompt (Augmentation)
		contextText := ""
		if len(docs) > 0 {
			// 🔥 修改点 2：修改日志文案，消除歧义
			respChan <- fmt.Sprintf("THINKing: 检索到 %d 个相关片段，正在阅读...", len(docs))

			for i, doc := range docs {
				// 这里为了调试，甚至可以把 Page Number 也打进日志里
				// 拼装上下文
				contextText += fmt.Sprintf("片段%d (第%d页): %s\n", i+1, doc.Page, doc.Content)
			}
		} else {
			respChan <- "THINKing: 未找到相关文档，将依靠通用知识回答..."
		}

		// 构造最终 Prompt
		finalPrompt := fmt.Sprintf("背景知识：\n%s\n\n用户问题：%s", contextText, req.Query)

		// 4. 生成 (Generation) - 调用 Python 的 AskStream
		respChan <- "THINKing: 正在生成回答..."
		stream, err := s.grpcClient.AskStream(ctx, &pb.AskRequest{Query: finalPrompt})
		if err != nil {
			respChan <- "ERR: LLM 连接失败 - " + err.Error()
			return
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				respChan <- "ERR: " + err.Error()
				return
			}
			// 将 AI 的回答推给前端
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
