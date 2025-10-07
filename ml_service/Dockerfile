# ml_service/Dockerfile
FROM python:3.12-slim

# Pillow needs image libs; curl for healthchecks
RUN apt-get update && apt-get install -y --no-install-recommends \
    libjpeg62-turbo-dev zlib1g-dev libpng-dev curl libgomp1 && \
    rm -rf /var/lib/apt/lists/*

# Helpful pip settings
ENV PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_DEFAULT_TIMEOUT=180

WORKDIR /app

# Install deps first (cache layer)
COPY requirements.txt /app/requirements.txt
RUN pip install --timeout 180 -r requirements.txt

# Copy code
COPY app /app/app

EXPOSE 8003
CMD ["python","-m","uvicorn","app.main:app","--host","0.0.0.0","--port","8003"]
