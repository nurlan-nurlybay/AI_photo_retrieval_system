# vector_service/Dockerfile
FROM python:3.12-slim

# libgomp1 often required by faiss-cpu wheels; curl for healthchecks
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Python deps first (faiss-cpu, fastapi, uvicorn, numpy, etc.)
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy service code (expects app/main.py, app/…)
COPY app /app/app

EXPOSE 8002
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8002"]
