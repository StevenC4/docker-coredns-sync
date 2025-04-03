FROM python:3.11-slim AS base
WORKDIR /app
LABEL org.opencontainers.image.source="https://github.com/StevenC4/docker-coredns-sync"
ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY src/ ./src/


FROM base AS dev
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=1000
# Create non-root user
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get update && apt-get install -y \
        curl \
        bash \
        git \
        build-essential \
        libssl-dev \
        sudo \
        vim \
    && echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME \
    && rm -rf /var/lib/apt/lists/*
USER $USERNAME
WORKDIR /app
RUN pip install debugpy
EXPOSE 5678
CMD ["sleep", "infinity"]


FROM base AS release
CMD ["python", "-m", "src.main"]