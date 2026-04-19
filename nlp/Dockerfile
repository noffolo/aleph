FROM python:3.12-slim AS builder
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir --upgrade pip && pip install --no-cache-dir -r requirements.txt

FROM python:3.12-slim
LABEL maintainer="Aleph Core Team <devops@aleph.ai>"
WORKDIR /app
COPY --from=builder /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages
COPY . .
WORKDIR /app
EXPOSE 8001
ENTRYPOINT ["python", "main.py"]
