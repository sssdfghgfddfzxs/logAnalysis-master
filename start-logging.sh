#!/bin/bash

echo "Starting Log Analysis System with Nginx + Filebeat..."

# 确保网络存在
docker network create log-analysis-network 2>/dev/null || true

# 启动主系统
echo "Starting main log analysis system..."
docker compose up -d

# 等待系统启动
echo "Waiting for log analysis system to be ready..."
sleep 10

# 启动日志收集系统
echo "Starting nginx and filebeat..."
docker compose -f docker-compose.logging.yml up -d

echo "System started successfully!"
echo ""
echo "Services:"
echo "- Log Analysis Frontend: http://localhost:3000"
echo "- Log Analysis API: http://localhost:8080"
echo "- Nginx (Log Generator): http://localhost:80"
echo ""
echo "To view logs:"
echo "- docker compose logs -f filebeat"
echo "- docker compose logs -f nginx"
echo ""
echo "To stop:"
echo "- docker compose -f docker-compose.logging.yml down"
echo "- docker compose down"