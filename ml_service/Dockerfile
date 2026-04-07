FROM python:3.12-slim AS base

# Set up environment
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
<<<<<<< HEAD
    PIP_DEFAULT_TIMEOUT=180

WORKDIR /app

# Install system dependencies (fast and minimal)
=======
    HF_HOME=/app/models

WORKDIR /app

# Install system dependencies
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        libjpeg62-turbo-dev zlib1g-dev libpng-dev curl libgomp1 && \
    rm -rf /var/lib/apt/lists/*

# --- Stage 2: Dependencies ---
FROM base AS deps

<<<<<<< HEAD
# Copy only requirements to leverage Docker cache
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# --- Stage 3: Final image ---
=======
COPY requirements.txt .
RUN pip install --default-timeout=7200 --no-cache-dir -r requirements.txt

# --- Stage 3: Bake SigLIP Only ---
ARG HF_TOKEN
ENV HF_TOKEN=$HF_TOKEN

RUN python3 -c "from transformers import AutoProcessor, AutoModel; \
    print('Baking SigLIP...'); \
    AutoProcessor.from_pretrained('google/siglip2-so400m-patch14-384'); \
    AutoModel.from_pretrained('google/siglip2-so400m-patch14-384')"

# --- Stage 4: Final image ---
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
FROM base

COPY --from=deps /usr/local/lib/python3.12 /usr/local/lib/python3.12
COPY --from=deps /usr/local/bin /usr/local/bin
<<<<<<< HEAD
=======
COPY --from=deps /app/models /app/models
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1

COPY app /app/app

EXPOSE 8003
CMD ["python", "-m", "uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8003"]
<<<<<<< HEAD
=======

>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
