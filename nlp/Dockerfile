# Stage 1: Build — compile Prophet and other C extensions
FROM python:3.12-slim AS builder
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ build-essential && \
    rm -rf /var/lib/apt/lists/*
COPY requirements.txt .
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Stage 2: Runtime — minimal image
FROM python:3.12-slim
LABEL maintainer="Aleph Core Team <devops@aleph.ai>"
WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# Copy app code
COPY nlp_pb2.py nlp_pb2_grpc.py main.py ensemble.py markets.py simulator.py cache.py ./
COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

RUN groupadd -r aleph && useradd -r -g aleph aleph && \
    chown -R aleph:aleph /app
USER aleph

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD python -c "import grpc; channel=grpc.insecure_channel('localhost:8001'); stub=grpc.channel_ready_future(channel); stub.result(timeout=5); print('healthy')" || exit 1

EXPOSE 8001
ENTRYPOINT ["./docker-entrypoint.sh", "python", "main.py"]
