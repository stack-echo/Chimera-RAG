import logging
from concurrent import futures
import grpc
from config import Config
from service.rag_service import ChimeraLLMService, rag_service_pb2_grpc

def serve():
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10),
        options=[
            ('grpc.max_send_message_length', Config.MAX_MESSAGE_LENGTH),
            ('grpc.max_receive_message_length', Config.MAX_MESSAGE_LENGTH),
        ]
    )

    rag_service_pb2_grpc.add_LLMServiceServicer_to_server(ChimeraLLMService(), server)

    server.add_insecure_port(f'[::]:{Config.PORT}')
    logging.info(f"🚀 Chimera Brain v0.2.0 running on port {Config.PORT}...")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
    serve()