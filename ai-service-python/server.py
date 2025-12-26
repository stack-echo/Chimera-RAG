import sys
import os
import time
import logging
from concurrent import futures
import grpc

sys.path.append(os.path.join(os.path.dirname(__file__), 'rpc'))

# 导入生成的代码
import rag_service_pb2
import rag_service_pb2_grpc

# --- 业务逻辑实现 ---
class ChimeraLLMService(rag_service_pb2_grpc.LLMServiceServicer):

    def AskStream(self, request, context):
        """
        模拟流式问答接口
        """
        query = request.query
        print(f"[收到请求] Query: {query} | UseGraph: {request.use_graph}")

        # 1. 模拟 "思考过程" (Thinking Log)
        yield rag_service_pb2.AskResponse(
            thinking_log=f"正在分析意图... (Mock ID: {request.session_id})"
        )
        time.sleep(0.5) # 假装在思考

        if request.use_graph:
            yield rag_service_pb2.AskResponse(
                thinking_log="检测到专业术语，正在查询 NebulaGraph 知识图谱..."
            )
            time.sleep(0.5)

        # 2. 模拟 "流式吐字" (Answer Delta)
        # 假装这是 LLM 生成的回复
        mock_answer = f"这是 Chimera 针对问题 '{query}' 的模拟回答。"
        for char in mock_answer:
            yield rag_service_pb2.AskResponse(
                answer_delta=char
            )
            time.sleep(0.1) # 模拟打字机效果

        # 3. 模拟 "引用来源" (Source Docs)
        # 最后一次返回带上引用
        final_resp = rag_service_pb2.AskResponse()
        doc1 = final_resp.source_docs.add()
        doc1.doc_name = "危化品安全手册_v1.pdf"
        doc1.page_num = "12"
        doc1.score = 0.95
        yield final_resp

    def EmbedData(self, request, context):
        """
        模拟向量化接口
        """
        print(f"[向量化请求] Type: {'Image' if request.image_url else 'Text'}")

        # 模拟返回一个 4 维向量 (真实场景是 768 或 1024 维)
        return rag_service_pb2.EmbedResponse(
            vector=[0.1, 0.2, 0.3, 0.99]
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