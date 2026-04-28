FROM python:3.12-slim
LABEL maintainer="Aleph Core Team <devops@aleph.ai>"
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ build-essential && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

COPY . .
WORKDIR /app

RUN groupadd -r aleph && useradd -r -g aleph aleph && \
    chown -R aleph:aleph /app
USER aleph

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD python -c "import grpc; channel=grpc.insecure_channel('localhost:50051'); stub=grpc.channel_ready_future(channel); stub.result(timeout=5); print('healthy')" || exit 1

EXPOSE 8001
ENTRYPOINT ["python", "main.py"]
