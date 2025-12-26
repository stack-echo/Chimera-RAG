package handler

import (
	"Chimera-RAG/backend-go/internal/biz"
	"Chimera-RAG/backend-go/internal/service"
	"io"
	"net/http"

	pb "Chimera-RAG/api/rag/v1"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	svc *service.RagService
}

func NewChatHandler(svc *service.RagService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// HandleChatSSE 处理流式对话
// POST /api/v1/chat/stream
func (h *ChatHandler) HandleChatSSE(c *gin.Context) {
	// 1. 解析请求 JSON
	var jsonReq biz.ChatRequest
	if err := c.ShouldBindJSON(&jsonReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. 构造 gRPC 请求对象
	grpcReq := &pb.AskRequest{
		Query:     jsonReq.Query,
		SessionId: jsonReq.SessionID,
		UseGraph:  jsonReq.UseGraph,
	}

	// 3. 获取流数据管道
	// 注意：这里传入 c.Request.Context()，如果前端断开连接，gRPC 也会感知并取消
	respChan, err := h.svc.StreamChat(c.Request.Context(), grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call AI service"})
		return
	}

	// 4. 开启 SSE 流式响应
	c.Stream(func(w io.Writer) bool {
		// 从管道读取数据
		if msg, ok := <-respChan; ok {
			// SSE 格式: data: <内容>\n\n
			// Gin 的 c.SSEvent 会自动封装格式
			c.SSEvent("message", msg)
			return true // 继续保持连接
		}
		return false // 管道关闭，断开连接
	})
}

// HandleUpload 处理文件上传
// POST /api/v1/upload
func (h *ChatHandler) HandleUpload(c *gin.Context) {
	// 1. 获取上传的文件 (key 名为 "file")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}

	// 2. 调用 Service
	objectName, err := h.svc.UploadDocument(c.Request.Context(), file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "上传成功，已加入处理队列",
		"file_name": objectName,
	})
}
