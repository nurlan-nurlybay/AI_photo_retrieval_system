# AI Photo Retrieval System

A semantic photo retrieval system that lets you search your library using natural language or example images. Powered by CLIP, Milvus, and Go.

## Key Features
- **Semantic Search**: Search by meaning (e.g., "birthday party", "sunset") or image similarity using CLIP embeddings.
- **Smart Upload**: Multi-file upload with automatic duplicate detection and EXIF metadata extraction.
- **Cross-Platform**: Responsive Flutter client for Desktop and Mobile.
- **Scalable Architecture**: Modular Go backend with Python microservices for ML and Vector search.

## Tech Stack
- **Backend**: Go (Gin), PostgreSQL, Redis, SeaweedFS
- **ML & Search**: Python (FastAPI), OpenAI CLIP, Milvus Vector DB
- **Frontend**: Flutter

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Flutter SDK (for running the client)

### Run the System
```bash
# Start all backend services
docker-compose up -d

# Check service health
curl http://localhost:8080/healthz
```

### Run the Client
```bash
cd frontend_flutter
flutter pub get
flutter run
```

## Fine-Tuning Achievements
We fine-tuned the CLIP ViT-B/32 model on social event classification using LoRA (Low-Rank Adaptation).
- **Baseline Accuracy**: 83.21%
- **Fine-Tuned Accuracy**: **91.56%** (+8.35% improvement)
