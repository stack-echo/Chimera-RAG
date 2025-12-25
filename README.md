# Chimera-RAG 🚀

A high-performance, enterprise-grade RAG (Retrieval-Augmented Generation) engine built with **Golang**, **Python**, and **GraphRAG** technology.

## 🌟 Key Features

* **Microservices Architecture:** * **Golang Gateway**: High-concurrency I/O, WebSocket/SSE streaming, and task orchestration.
    * **Python AI Service**: LLM inference, LangChain integration, and Multi-Agent planning.
* **Communication:** High-performance inter-service communication using **gRPC** & **Protobuf**.
* **Multi-Modal RAG:** Supports **Image & Text** retrieval using MinIO (Object Storage) and CLIP models.
* **Hybrid Search:** Combines **Milvus** (Vector Search) and **NebulaGraph** (Knowledge Graph) for precision.
* **Infrastructure:** Fully containerized with Docker Compose (MySQL, Redis, MinIO, Milvus, NebulaGraph).

## 🛠 Tech Stack

* **Language:** Golang (1.21+), Python (3.10+)
* **Framework:** Gin, gRPC, LangChain
* **Storage:** MySQL, Redis, MinIO
* **Vector/Graph DB:** Milvus, NebulaGraph

## 🚀 Quick Start

### Prerequisites
* Docker & Docker Compose
* Golang 1.21+
* Python 3.10+

### Run Infrastructure
```bash
cd deploy
docker-compose up -d