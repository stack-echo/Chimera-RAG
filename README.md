# 🦄 Chimera-RAG (v0.1.0)

Chimera-RAG 是一个基于 **Go + Python** 混合架构的企业级 EHS 安全合规知识库助手。它实现了完整的 RAG (检索增强生成) 链路，支持 PDF 深度解析与流式问答。

## ✨ 核心特性 (v0.1.0)

- **多模态微服务架构**：Go 处理高并发 I/O，Python 处理 AI 推理。
- **全链路 RAG**：
  - 📄 **解析**：基于 PyMuPDF 的 PDF 文本提取与切片。
  - 🧠 **记忆**：Qdrant 向量数据库 (384维)。
  - 💬 **生成**：接入 DeepSeek V3 大模型，支持 Markdown 流式输出。
- **现代化前端**：Vue 3 + Arco Design 实现的极简交互界面。

## 🛠️ 技术栈

- **Backend**: Golang, Gin, gRPC, MinIO, Redis
- **AI Service**: Python, PyMuPDF, Sentence-Transformers, OpenAI SDK
- **Vector DB**: Qdrant
- **Frontend**: Vue 3, Vite, SSE (Server-Sent Events)

## 🚀 快速开始

### 1. 启动基础设施
```bash
cd deploy
docker-compose up -d
```

### 2. 启动 AI 服务 (Python)
```bash
cd ai-service-python
# 确保已配置 .env
python server.py
```

### 3. 启动后端网关 (Go)
```bash
cd backend-go
go run cmd/server/main.go
```

### 4. 启动前端
```bash
cd frontend-vue
npm run dev
```
## 📅 Roadmap
[x] v0.1.0: 基础 RAG 链路跑通，支持 PDF 上传与问答。

[ ] v0.2.0: 优化 PDF 表格解析，增加多轮对话上下文。