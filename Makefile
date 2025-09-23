.PHONY: all backend faiss clip services

all: faiss clip backend

backend:
	cd backend/bin && ./photo-backend &

faiss:
	cd vector_service && uvicorn app.main:app --host 0.0.0.0 --port 8002 &

clip:
	cd ml_service && uvicorn app.main:app --host 0.0.0.0 --port 8003 &

stop:
	@pkill -f "uvicorn app.main:app" || true
