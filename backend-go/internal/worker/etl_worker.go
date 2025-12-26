package worker

import (
	"context"
	"io"
	"log"
	"time"

	pb "Chimera-RAG/backend-go/api/rag/v1"
	"Chimera-RAG/backend-go/internal/data"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/qdrant/go-client/qdrant"
)

// ETLWorker 负责从 Redis 拿任务，并执行 ETL 流程
type ETLWorker struct {
	data       *data.Data
	grpcClient pb.LLMServiceClient
}

func NewETLWorker(data *data.Data, client pb.LLMServiceClient) *ETLWorker {
	return &ETLWorker{
		data:       data,
		grpcClient: client,
	}
}

// Start 启动 Worker (阻塞运行)
func (w *ETLWorker) Start(ctx context.Context, numWorkers int) {
	log.Printf("🚀 启动 %d 个 ETL Worker，开始监听队列 task:parse_pdf...", numWorkers)

	for i := 0; i < numWorkers; i++ {
		go w.processLoop(ctx, i)
	}
}

func (w *ETLWorker) processLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 1. 阻塞式获取任务 (BLPOP)
			result, err := w.data.Redis.BLPop(ctx, 0*time.Second, "task:parse_pdf").Result()
			if err != nil {
				// Redis 偶尔连接超时是正常的，不要 panic
				log.Printf("[Worker-%d] 等待任务中... (%v)", workerID, err)
				time.Sleep(3 * time.Second)
				continue
			}

			fileName := result[1]
			log.Printf("[Worker-%d] 收到任务: %s", workerID, fileName)

			// 2. 执行具体处理逻辑
			err = w.processFile(ctx, fileName)
			if err != nil {
				log.Printf("[Worker-%d] ❌ 处理失败: %s, 错误: %v", workerID, fileName, err)
			} else {
				log.Printf("[Worker-%d] ✅ 处理完成: %s", workerID, fileName)
			}
		}
	}
}

// processFile 单个文件的 ETL 流程
func (w *ETLWorker) processFile(ctx context.Context, fileName string) error {
	// A. 从 MinIO 获取文件流
	obj, err := w.data.Minio.GetObject(ctx, "chimera-docs", fileName, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer obj.Close()

	// 读取文件所有字节 (注意内存安全，大文件要分片，但Demo演示先直接读)
	fileBytes, err := io.ReadAll(obj)
	if err != nil {
		return err
	}

	// B. 调用 Python 进行 解析+切片+向量化
	log.Printf("📡 发送 PDF 给 Python 进行深度解析: %s", fileName)
	parseResp, err := w.grpcClient.ParseAndEmbed(ctx, &pb.ParseRequest{
		FileContent: fileBytes,
		FileName:    fileName,
	})
	if err != nil {
		return err
	}

	// C. 批量存入 Qdrant
	points := make([]*qdrant.PointStruct, 0, len(parseResp.Chunks))

	for i, chunk := range parseResp.Chunks {
		pointID := uuid.New().String()

		// 构造 Payload (元数据)
		// 这些数据就是以后检索回来给 DeepSeek 看的“背景知识”
		payloadMap := map[string]interface{}{
			"filename":    fileName,
			"content":     chunk.Content,    // 存正文！
			"page_number": chunk.PageNumber, // 存页码！
			"chunk_index": i,
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(pointID),
			Vectors: qdrant.NewVectors(chunk.Vector...),
			Payload: qdrant.NewValueMap(payloadMap),
		})
	}

	// 批量写入 (Batch Upsert)
	// 真实场景建议分批，每次 100 个
	if len(points) > 0 {
		_, err = w.data.Qdrant.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: "chimera_docs",
			Points:         points,
		})
		if err != nil {
			return err
		}
	}

	log.Printf("✅ ETL 完成: %s 生成了 %d 个向量切片", fileName, len(points))
	return nil
}
