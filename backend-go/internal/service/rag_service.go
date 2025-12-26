package service

import (
	"context"
	"io"
	"log"

	pb "Chimera-RAG/api/rag/v1" // 请确认你的 module 名
)

// RagService 定义业务逻辑
type RagService struct {
	grpcClient pb.LLMServiceClient
}

// NewRagService 构造函数
func NewRagService(client pb.LLMServiceClient) *RagService {
	return &RagService{grpcClient: client}
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
