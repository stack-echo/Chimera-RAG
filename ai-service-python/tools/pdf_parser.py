import logging
import io
from pathlib import Path

# Docling 核心
from docling.document_converter import DocumentConverter, PdfFormatOption
from docling.datamodel.base_models import InputFormat, DocumentStream
from docling.datamodel.pipeline_options import PdfPipelineOptions, TableStructureOptions

# 🔥 新增：Docling 原生切分器
from docling.chunking import HybridChunker

class PDFParser:
    _converter = None
    _chunker = None

    @classmethod
    def _get_components(cls):
        """单例模式初始化 Converter 和 Chunker"""
        if cls._converter is None:
            logging.info("🐢 [Init] 正在初始化 Docling 模型...")

            # 1. 配置转换器
            pipeline_options = PdfPipelineOptions()
            pipeline_options.do_ocr = False
            pipeline_options.do_table_structure = True

            cls._converter = DocumentConverter(
                format_options={
                    InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_options)
                }
            )

            # 2. 配置切分器 (HybridChunker)
            # 它可以智能地结合“语义结构”和“Token限制”来切分
            cls._chunker = HybridChunker(
                tokenizer="sentence-transformers/all-MiniLM-L6-v2", # 用和 Embedding 一样的 tokenizer 估算长度
                max_tokens=500, # 每个块的最大 Token 数
                merge_peers=True, # 合并同级标题下的内容
            )

            logging.info("✅ [Init] Docling 组件就绪")
        return cls._converter, cls._chunker

    @staticmethod
    def parse_and_chunk(file_source, filename="temp.pdf"):
        """
        解析 PDF 并返回带有【真实页码】的语义切片
        """
        converter, chunker = PDFParser._get_components()
        logging.info(f"📄 [Docling] 开始解析: {filename}")

        try:
            # 1. 构建输入源
            input_doc = None
            if isinstance(file_source, bytes):
                input_doc = DocumentStream(name=filename, stream=io.BytesIO(file_source))
            else:
                input_doc = Path(file_source)

            # 2. 执行转换 (PDF -> DL Document)
            # 这一步比较耗时 (CPU/MPS)
            conv_result = converter.convert(input_doc)
            doc = conv_result.document
            logging.info(f"✅ [Docling] 转换完成，开始提取切片...")

            # 3. 使用 HybridChunker 切分 (提取真实页码的核心步骤)
            # chunker.chunk(doc) 返回的是 Docling 的 Chunk 对象迭代器
            chunk_iter = chunker.chunk(doc)

            final_chunks = []
            for i, chunk in enumerate(chunk_iter):
                # chunk.text: 包含了标题上下文的文本 (例如: "Header1 > Header2 \n 正文...")
                # chunk.meta: 包含了元数据

                # 🔥 提取页码
                # Docling 的 chunk 可能跨页，我们取这个 chunk 出现的“第一页”作为跳转目标
                page_num = 1
                if chunk.meta.doc_items:
                    # 追溯这个 chunk 来源于文档的哪个部分 (Provenance)
                    first_item = chunk.meta.doc_items[0]
                    if hasattr(first_item, 'prov') and first_item.prov:
                        page_num = first_item.prov[0].page_no

                # 序列化结果
                final_chunks.append({
                    "content": chunk.text, # HybridChunker 自动帮你拼好了上下文，不需要手动 join 标题了
                    "page": page_num       # ✅ 真实的页码！
                })

            logging.info(f"✂️ [HybridChunker] 生成了 {len(final_chunks)} 个带有页码的片段")

            # 打印前3个看看效果
            for idx, c in enumerate(final_chunks[:3]):
                logging.info(f"   🔹 P{c['page']}: {c['content'][:50]}...")

            return final_chunks

        except Exception as e:
            logging.error(f"❌ [Docling] 解析失败: {e}", exc_info=True)
            return []