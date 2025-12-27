import sys
import os
import logging

# 确保能导入 rpc 目录
sys.path.append(os.path.join(os.path.dirname(os.path.dirname(__file__)), 'rpc'))

import rag_service_pb2
import rag_service_pb2_grpc

# 引入核心组件
from core.llm import LLMClient
from core.embedding import EmbeddingModel
from tools.pdf_parser import PDFParser

class ChimeraLLMService(rag_service_pb2_grpc.LLMServiceServicer):
    def __init__(self):
        # 初始化 LLM 客户端
        self.llm = LLMClient()
        # 预加载 Embedding 模型
        EmbeddingModel.get_instance()

    # ----------------------------------------------------------------
    # 1. 核心问答接口 (Stream)
    # ----------------------------------------------------------------
    def AskStream(self, request, context):
        """
        接收 Go 传来的 Prompt，流式返回 LLM 的回答
        """
        logging.info(f"[LLM] 收到提问请求 (长度: {len(request.query)} chars)...")

        # 🔥 v0.3.5 关键点：System Prompt
        # 在这里强制要求 LLM 使用 <<文件名|页码>> 的格式
        system_prompt = """
        你是一个专业的科研助手 (Chimera-RAG)。请根据提供的上下文回答问题。

        【重要回复规则】
        1. 必须严格基于上下文回答，不要编造事实。
        2. 引用格式：当引用上下文内容时，必须在句尾加上来源标记。
           格式为：<<文件名|页码>>
           例如："...这一结论得到了实验验证<<research.pdf|4>>。"
        3. 如果上下文里没有答案，请诚实地说不知道。
        4. 保持回答简洁明了，使用 Markdown 格式。
        """

        # 调用 LLM (流式)
        # request.query 是 Go 拼装好的 "Context + User Question"
        try:
            generator = self.llm.stream_chat(request.query, system_prompt=system_prompt)

            for text_delta in generator:
                # 封装成 gRPC 响应
                yield rag_service_pb2.AskResponse(answer_delta=text_delta)

        except Exception as e:
            logging.error(f"❌ LLM 调用失败: {e}")
            yield rag_service_pb2.AskResponse(answer_delta=f"**Error**: {str(e)}")

    # ----------------------------------------------------------------
    # 2. 向量化接口
    # ----------------------------------------------------------------
    def EmbedData(self, request, context):
        text = request.text
        # 调用 core 层的 Embedding
        vector = EmbeddingModel.encode(text)
        return rag_service_pb2.EmbedResponse(vector=vector)

    # ----------------------------------------------------------------
    # 3. 文档解析接口 (v0.3.0 Docling)
    # ----------------------------------------------------------------
    def ParseAndEmbed(self, request, context):
        logging.info(f"[Parse] 收到文件: {request.file_name}, 大小: {len(request.file_content)} bytes")

        # 1. 调用 Docling 解析 (传入 bytes)
        raw_chunks = PDFParser.parse_and_chunk(
            file_source=request.file_content,
            filename=request.file_name
        )

        if not raw_chunks:
             logging.warning("⚠️ 解析结果为空")
             return rag_service_pb2.ParseResponse(chunks=[])

        # 2. 批量向量化并组装
        grpc_chunks = []
        for item in raw_chunks:
            # 向量化
            vector = EmbeddingModel.encode(item['content'])

            grpc_chunks.append(rag_service_pb2.DocChunk(
                content=item['content'],
                vector=vector,
                page_number=item['page'] # ✅ 确保这里透传了 Docling 解析出的页码
            ))

        logging.info(f"[Parse] 完成! 返回 {len(grpc_chunks)} 个 Chunk 给 Go 端")
        return rag_service_pb2.ParseResponse(chunks=grpc_chunks)