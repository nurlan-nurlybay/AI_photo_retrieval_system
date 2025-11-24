FROM python:3.12-slim AS base

# Set up environment
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_DEFAULT_TIMEOUT=180

WORKDIR /app

# Install system dependencies (fast and minimal)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        libjpeg62-turbo-dev zlib1g-dev libpng-dev curl libgomp1 && \
    rm -rf /var/lib/apt/lists/*

# --- Stage 2: Dependencies ---
FROM base AS deps

# Copy only requirements to leverage Docker cache
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 3: Final image ---
FROM base

COPY --from=deps /usr/local/lib/python3.12 /usr/local/lib/python3.12
COPY --from=deps /usr/local/bin /usr/local/bin

COPY app /app/app

EXPOSE 8003
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8003"]
