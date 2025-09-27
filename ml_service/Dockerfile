# ml_service/Dockerfile
FROM python:3.12-slim

# Pillow needs image libs; curl helps healthchecks/logs
RUN apt-get update && apt-get install -y --no-install-recommends \
    libjpeg62-turbo-dev zlib1g-dev libpng-dev curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Layer cache: install deps first
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy your FastAPI app (expects app/main.py etc.)
COPY app /app/app

# The service must listen on 0.0.0.0:8003
EXPOSE 8003
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8003"]
