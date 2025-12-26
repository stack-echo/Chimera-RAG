import sys
import os
import time
import logging
from concurrent import futures
import grpc
from sentence_transformers import SentenceTransformer

sys.path.append(os.path.join(os.path.dirname(__file__), 'rpc'))

# 导入生成的代码
import rag_service_pb2
import rag_service_pb2_grpc

# --- 初始化 AI 模型 ---
print("📥 正在加载 Embedding 模型 (all-MiniLM-L6-v2)...")
# 这个模型很小(80MB)，下载很快，生成 384 维向量
embed_model = SentenceTransformer('all-MiniLM-L6-v2')
print("✅ 模型加载完毕！")

# --- 业务逻辑实现 ---
class ChimeraLLMService(rag_service_pb2_grpc.LLMServiceServicer):

    def AskStream(self, request, context):
            """
            暂时还保留 Mock 对话，下一步再接入 DeepSeek/OpenAI
            """
            query = request.query
            print(f"[收到提问] {query}")

            yield rag_service_pb2.AskResponse(thinking_log=f"正在计算查询向量 (384维)...")
            
            # 这里演示一下：我们真的去算一下提问的向量
            q_vector = embed_model.encode(query).tolist()
            yield rag_service_pb2.AskResponse(thinking_log=f"向量计算完毕，维度: {len(q_vector)}")
            time.sleep(0.5)

            yield rag_service_pb2.AskResponse(answer_delta="这是 Python 端集成 HuggingFace 模型后的测试回复。")

    def EmbedData(self, request, context):
            """
            【真实】向量化接口
            """
            start = time.time()

            # 1. 提取文本
            text = ""
            if request.text:
                text = request.text
            elif request.image_url:
                text = "Image embedding not implemented yet" # 暂时跳过图片

            print(f"[向量化请求] 正在处理文本，长度: {len(text)}")

            # 2. 调用模型推理 (Inference)
            # tolist() 是为了把 numpy 数组转为 Python list，否则 gRPC 传不过去
            vector = embed_model.encode(text).tolist()

            duration = (time.time() - start) * 1000
            print(f"✅ 向量化完成，耗时: {duration:.2f}ms，维度: {len(vector)}")

            # 3. 返回真实向量
            return rag_service_pb2.EmbedResponse(
                vector=vector
            )

# --- 服务器启动逻辑 ---
def serve():
    # 创建 gRPC 服务器
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))

    # 注册我们的服务
    rag_service_pb2_grpc.add_LLMServiceServicer_to_server(ChimeraLLMService(), server)

    # 监听端口
    server.add_insecure_port('[::]:50051')
    print("🚀 Chimera Brain is running on port 50051...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig()
    serve()