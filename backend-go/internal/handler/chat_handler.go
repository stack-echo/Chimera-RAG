package handler

import (
	"Chimera-RAG/backend-go/internal/biz"
	"Chimera-RAG/backend-go/internal/service"
	"fmt"
	"io"
	"net/http"

	pb "Chimera-RAG/backend-go/api/rag/v1"

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

// HandleUpload 修改版
func (h *ChatHandler) HandleUpload(c *gin.Context) {
	// 1. 获取用户 ID
	userID := c.GetUint("userID") // 假设中间件设置了 uint 类型的 userID

	// 2. 获取文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "文件无效"})
		return
	}

	// 3. 调用 Service
	doc, err := h.svc.UploadDocument(c.Request.Context(), fileHeader, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 4. 返回结果
	c.JSON(200, gin.H{
		"msg":    "上传成功",
		"doc_id": doc.ID,
		"path":   doc.StoragePath,
	})
}

// HandleGetFile 下载/预览文件
// GET /api/v1/file/:filename
func (h *ChatHandler) HandleGetFile(c *gin.Context) {
	filename := c.Param("filename")

	// 1. 调用 Service 层获取流
	// 注意：obj 是一个 ReadCloser，必须关闭
	obj, size, err := h.svc.GetFile(c.Request.Context(), filename)
	if err != nil {
		// 生产环境建议区分 "文件不存在" 和 "服务器错误"
		c.JSON(http.StatusNotFound, gin.H{"error": "文件获取失败: " + err.Error()})
		return
	}
	// 🔥 重要：流传输完成后关闭连接
	defer obj.Close()

	// 2. 设置 HTTP 响应头
	// 告诉浏览器这是一个 PDF，文件大小是多少（方便显示进度条）
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "inline; filename="+filename) // inline=浏览器内预览, attachment=强制下载
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Length", fmt.Sprintf("%d", size))

	// 3. 将流拷贝到响应体 (Stream)
	// 这一步会阻塞直到文件传输完成，内存占用极低
	_, err = io.Copy(c.Writer, obj)
	if err != nil {
		// 如果传输过程中断，通常也没法写 JSON 错误了，只能记录日志
		fmt.Printf("Stream file error: %v\n", err)
	}
}
