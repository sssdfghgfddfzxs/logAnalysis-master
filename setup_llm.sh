#!/bin/bash

# LLM功能设置和测试脚本

set -e

echo "🚀 设置LLM分析功能..."

# 检查是否存在.env文件
if [ ! -f ".env" ]; then
    echo "📝 创建.env文件..."
    cp .env.example .env
    echo ""
    echo "⚠️  请编辑.env文件并设置您的SILICONFLOW_API_TOKEN"
    echo "   获取Token: https://siliconflow.cn"
    echo ""
    read -p "按Enter键继续（确保已设置API Token）..."
fi

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker未运行，请启动Docker"
    exit 1
fi

# 构建和启动服务
echo "🔨 构建和启动所有服务..."
make rebuild-all

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "📊 检查服务状态..."
make status

# 运行测试
echo ""
echo "🧪 运行LLM集成测试..."
python3 test_llm_integration.py

echo ""
echo "✅ LLM功能设置完成！"
echo ""
echo "🌐 访问地址："
echo "  - 前端界面: http://localhost:3000"
echo "  - 后端API: http://localhost:8080"
echo "  - LLM API: http://localhost:5000"
echo ""
echo "📚 常用命令："
echo "  - 查看所有日志: make logs"
echo "  - 查看LLM日志: make logs-llm"
echo "  - 测试LLM功能: make test-llm"
echo "  - 更新LLM服务: make update-llm"
echo "  - 停止所有服务: make stop"