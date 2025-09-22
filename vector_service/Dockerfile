FROM python:latest-slim

WORKDIR /app
COPY requirements.txt /app/requirements.txt

# faiss-cpu needs blas
RUN apt-get update && apt-get install -y --no-install-recommends \
      libopenblas-base && \
    pip install --no-cache-dir -r requirements.txt && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

COPY app /app/app

ENV PYTHONUNBUFFERED=1
EXPOSE 8002

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8002"]
