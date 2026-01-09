# Runtime stage only
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*
WORKDIR /root/

# Copy the pre-built binary from local build
# CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server
COPY main .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]