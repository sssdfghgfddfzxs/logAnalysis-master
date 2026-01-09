# Use Python 3.12 slim image
FROM python:3.12.10-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements first for better caching
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt -i https://mirrors.aliyun.com/pypi/simple

# Copy source code
COPY . .

# Create models directory
RUN mkdir -p models

# Expose gRPC port and LLM API port
EXPOSE 50051 5000

# Health check for gRPC service
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD python -c "import grpc; print('OK')" || exit 1

# Set working directory to src for proper imports
WORKDIR /app/src

# Default command runs the gRPC AI service
CMD ["python", "ai_service.py"]