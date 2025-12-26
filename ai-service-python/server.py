import sys
import os
import time
import logging
from dotenv import load_dotenv
from concurrent import futures
import grpc
from sentence_transformers import SentenceTransformer
from openai import OpenAI

sys.path.append(os.path.join(os.path.dirname(__file__), 'rpc'))

# 导入生成的代码
import rag_service_pb2
import rag_service_pb2_grpc

# 1. 加载 .env 文件
load_dotenv()

# 2. 从环境变量读取 (如果读不到，可以给个默认值或者报错)
API_KEY = os.getenv("DEEPSEEK_API_KEY")
BASE_URL = os.getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com")

if not API_KEY:
    raise ValueError("❌ 未找到 DEEPSEEK_API_KEY，请在 .env 文件中配置！")

# --- 初始化 ---
print("📥 正在加载 Embedding 模型...")
try:
    # 尝试从本地加载，如果失败则下载
    model_dir = snapshot_download('AI-ModelScope/all-MiniLM-L6-v2')
    embed_model = SentenceTransformer(model_dir)
except:
    embed_model = SentenceTransformer('all-MiniLM-L6-v2')
print("✅ Embedding 模型加载完毕！")

# 初始化 LLM 客户端
llm_client = OpenAI(api_key=API_KEY, base_url=BASE_URL)

# --- 业务逻辑实现 ---
class ChimeraLLMService(rag_service_pb2_grpc.LLMServiceServicer):

    def AskStream(self, request, context):
            """
            核心问答接口：接收 Prompt -> 调用 LLM -> 流式返回
            """
            query = request.query # 这里的 query 实际上是 Go 拼装好的 Prompt (包含上下文)
            print(f"[LLM] 收到 Prompt，准备生成回答...")

            # 1. 调用 DeepSeek API
            try:
                response = llm_client.chat.completions.create(
                    model="deepseek-chat",
                    messages=[
                        {"role": "system", "content": "你是一个专业的EHS安全助手。请根据提供的上下文回答问题。如果上下文里没有答案，请诚实地说不知道。"},
                        {"role": "user", "content": query},
                    ],
                    stream=True # 开启流式
                )

                # 2. 流式转发给 Go
                for chunk in response:
                    if chunk.choices[0].delta.content:
                        content = chunk.choices[0].delta.content
                        yield rag_service_pb2.AskResponse(answer_delta=content)

            except Exception as e:
                print(f"❌ LLM 调用失败: {e}")
                yield rag_service_pb2.AskResponse(answer_delta=f"[Error] 大模型服务异常: {str(e)}")

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