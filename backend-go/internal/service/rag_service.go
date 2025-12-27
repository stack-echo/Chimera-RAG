package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	pb "Chimera-RAG/backend-go/api/rag/v1"
	"Chimera-RAG/backend-go/internal/data"
)

// RagService 定义业务逻辑
type RagService struct {
	grpcClient pb.LLMServiceClient
	Data       *data.Data
}

// NewRagService 构造函数
func NewRagService(client pb.LLMServiceClient, data *data.Data) *RagService {
	return &RagService{
		grpcClient: client,
		Data:       data,
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
		docs, err := s.Data.SearchSimilar(ctx, embResp.Vector, 15)
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

// UploadDocument 处理文件上传全流程
func (s *RagService) UploadDocument(ctx context.Context, fileHeader *multipart.FileHeader, userID uint) (*data.Document, error) {
	// 1. 打开文件流
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// 2. [Data层] 上传到 MinIO
	// Service 层不需要知道 MinIO SDK 的细节，只需要给文件流
	storagePath, err := s.Data.UploadFile(ctx, src, fileHeader.Size, fileHeader.Filename)
	if err != nil {
		return nil, err
	}

	// 3. [Data层] 写入数据库 (v0.2.0 文件确权)
	doc := &data.Document{
		Title:           fileHeader.Filename,
		FileName:        fileHeader.Filename,
		FileSize:        fileHeader.Size,
		FileType:        strings.ToLower(filepath.Ext(fileHeader.Filename)), // 简单的后缀判断工具函数
		StoragePath:     storagePath,
		KnowledgeBaseID: 0, // 默认归属根目录，后续可传参
		OwnerID:         userID,
		Status:          "pending",
	}

	if err := s.Data.CreateDocument(ctx, doc); err != nil {
		// ⚠️ 进阶思考: 如果数据库写入失败，最好把 MinIO 里的垃圾文件删掉 (补偿机制)
		// s.Data.DeleteFile(ctx, storagePath)
		return nil, err
	}

	// 4. [Data层] 写入 Redis 任务队列
	// 传递 Document ID 而不是路径，Worker 可以根据 ID 查库获取更多信息
	// 也可以传 JSON: {"doc_id": 1, "path": "xxx.pdf"}
	err = s.Data.PushTask(ctx, "task:parse_pdf", storagePath)
	if err != nil {
		// 同样，如果队列失败，考虑是否回滚数据库状态为 "failed"
		return nil, err
	}

	return doc, nil
}
