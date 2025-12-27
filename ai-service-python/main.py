import grpc
from concurrent import futures
import logging
import os
import tempfile
from pathlib import Path

# 引入生成的 gRPC 代码
import rag_pb2
import rag_pb2_grpc

# 🔥 Docling 核心组件
from docling.document_converter import DocumentConverter
from docling.datamodel.base_models import InputFormat
from docling.datamodel.pipeline_options import PdfPipelineOptions

# 🔥 LangChain 切分工具 (按 Markdown 标题切分)
from langchain_text_splitters import MarkdownHeaderTextSplitter

class LLMService(rag_pb2_grpc.LLMServiceServicer):
    def __init__(self):
        logging.info("正在初始化 Docling Converter...")
        # 配置 Docling
        pipeline_options = PdfPipelineOptions()
        pipeline_options.do_ocr = False  # 如果是纯文本PDF，关掉OCR速度快；扫描件请开启
        pipeline_options.do_table_structure = True # 开启表格解析

        self.converter = DocumentConverter(
            format_options={
                InputFormat.PDF: pipeline_options
            }
        )
        logging.info("✅ Docling 初始化完成")

    def ParsePDF(self, request, context):
        """
        核心逻辑：接收 PDF URL/Path -> 下载/读取 -> Docling 转 Markdown -> 智能切分 -> 返回 Chunk
        """
        file_path = request.file_path
        logging.info(f"收到解析任务: {file_path}")

        # 1. 临时保存/读取文件
        # 注意：这里假设 backend-go 传过来的是本地路径 (minio 挂载或者是下载后的路径)
        # 如果是 URL，Docling 也支持直接传 URL

        if not os.path.exists(file_path):
             # 简单的容错，防止路径不对
             context.set_code(grpc.StatusCode.NOT_FOUND)
             context.set_details(f"File not found: {file_path}")
             return rag_pb2.ParseResponse()

        try:
            # 2. 🔥 Docling 核心解析：PDF -> Markdown
            logging.info("开始 Docling 解析 (可能需要几秒钟)...")
            conv_result = self.converter.convert(file_path)

            # 获取 Markdown 内容
            markdown_content = conv_result.document.export_to_markdown()
            logging.info(f"解析完成，Markdown 长度: {len(markdown_content)}")

            # 3. 🔥 智能切分 (Semantic Chunking)
            # 定义想要作为切分点的 Header 级别
            headers_to_split_on = [
                ("#", "Header 1"),
                ("##", "Header 2"),
                ("###", "Header 3"),
            ]

            markdown_splitter = MarkdownHeaderTextSplitter(
                headers_to_split_on=headers_to_split_on,
                strip_headers=False # 保留标题在内容里，让上下文更清晰
            )

            docs = markdown_splitter.split_text(markdown_content)

            logging.info(f"智能切分完成，共 {len(docs)} 个片段")

            # 4. 组装返回结果
            chunks = []
            for i, doc in enumerate(docs):
                # 组合元数据和内容
                # doc.page_content 是纯文本
                # doc.metadata 包含 {'Header 1': '...', 'Header 2': '...'}

                # 我们可以把标题拼回到内容前面，增强语义
                header_context = " > ".join(doc.metadata.values())
                final_content = f"【章节: {header_context}】\n{doc.page_content}"

                chunks.append(rag_pb2.Chunk(
                    content=final_content,
                    page_number=1 # Docling 目前转 Markdown 后页码对齐比较复杂，暂时由 Go 端处理或填 1
                ))

            return rag_pb2.ParseResponse(chunks=chunks)

        except Exception as e:
            logging.error(f"解析失败: {str(e)}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return rag_pb2.ParseResponse()

    def Embed(self, request, context):
        # ... (Embed 代码保持不变，或者暂时留空，如果你还在用模拟 Embed) ...
        # 这里为了演示，先返回模拟向量
        return rag_pb2.EmbedResponse(
            vectors=[rag_pb2.Vector(data=[0.1] * 768) for _ in request.documents]
        )

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    rag_pb2_grpc.add_LLMServiceServicer_to_server(LLMService(), server)
    server.add_insecure_port('[::]:50051')
    logging.info("🚀 Python AI Service (Docling版) 已启动，监听 50051...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    serve()