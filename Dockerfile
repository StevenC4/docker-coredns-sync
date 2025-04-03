FROM python:3.11-slim AS base
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/StevenC4/docker-coredns-sync"

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY src/ ./src/

FROM base AS dev
RUN apt-get update && apt-get install -y \
    curl \
    bash \
    git \
    build-essential \
    libssl-dev \
    vim \
    && rm -rf /var/lib/apt/lists/*
RUN pip install debugpy
EXPOSE 5678
CMD ["sleep", "infinity"]

FROM base AS release
CMD ["python", "-m", "src.main"]