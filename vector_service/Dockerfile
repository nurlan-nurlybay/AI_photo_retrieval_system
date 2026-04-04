FROM python:3.11-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_DEFAULT_TIMEOUT=180

WORKDIR /app

# Install minimal system build deps and runtime libs
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        build-essential \
        g++ gcc \
        libatomic1 \
        libgomp1 \
        curl && \
    rm -rf /var/lib/apt/lists/*

# --- Stage 2: Install Python deps (cached) ---
FROM base AS deps

COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip setuptools wheel && \
    pip install --no-cache-dir -r requirements.txt

# --- Stage 3: Final image ---
FROM base

# Copy installed packages only (so final image is smaller)
COPY --from=deps /usr/local/lib/python3.11 /usr/local/lib/python3.11
COPY --from=deps /usr/local/bin /usr/local/bin

# Copy app source
COPY app /app/app

# Copy the weights file that was missing
COPY production_weights.json /app/production_weights.json

EXPOSE 8002
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8002"]

