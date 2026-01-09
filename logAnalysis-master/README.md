# 智能日志异常分析系统

一个基于AI的轻量级日志异常检测和分析系统，支持自动异常识别、根因推荐和大模型深度分析。

## 系统架构

- **Go Backend**: 基于Gin的HTTP API服务器，负责日志接收和任务调度
- **Python AI Module**: 基于gRPC的AI分析服务，提供异常检测和根因分析
- **LLM Analysis API**: 基于大模型的深度日志分析服务（SiliconFlow API）
- **Vue Frontend**: 基于Vue 3的前端界面，提供数据可视化和用户交互
- **PostgreSQL**: 主数据库，存储日志和分析结果
- **Redis**: 缓存服务，提供高性能数据访问

## 新功能：大模型分析

系统现已集成大模型分析功能，**完全保持与现有系统的兼容性**：
- 🤖 智能日志批量分析和洞察
- 🔍 单个日志深度分析和建议
- 📊 风险评估和趋势分析
- 💡 专业的根因分析和解决方案
- 🎯 自定义提示词分析
- ✅ **零修改集成**：现有代码无需改动

## 新功能：告警通知系统

系统现已支持智能告警通知功能：
- 📧 **邮件通知**：支持HTML格式的详细告警邮件
- 💬 **钉钉通知**：支持钉钉机器人消息推送
- ⚙️ **灵活配置**：可视化告警规则配置界面
- 🎯 **智能触发**：基于异常评分和条件的智能告警
- 🔕 **告警抑制**：防止重复告警的智能抑制机制
- 🧪 **测试功能**：一键测试告警配置是否正常


## 快速开始

### 一键设置（推荐）

```bash
# 克隆项目
git clone <repository-url>
cd intelligent-log-analysis

# 1. 复制环境配置文件
cp .env.example .env
# 编辑.env文件，设置SILICONFLOW_API_TOKEN等必要配置

# 2. 启动所有容器
make update

# 3. 运行测试脚本发送日志进行测试
python test_api.py
```

### 使用Docker Compose

1. 设置环境变量
```bash
cp .env.example .env
# 编辑.env文件，设置SILICONFLOW_API_TOKEN
```

2. 构建和启动所有服务
```bash
make update
```

3. 测试系统
```bash
python test_api.py
```

4. 访问系统
- 前端界面: http://localhost:3000
- 后端API: http://localhost:8080
- LLM分析API: http://localhost:5000

## API接口

### 日志上报接口
```http
POST /api/v1/logs
Content-Type: application/json

{
  "timestamp": "2024-01-01T10:00:00Z",
  "level": "ERROR",
  "message": "Database connection failed",
  "source": "user-service",
  "metadata": {
    "host": "server-01",
    "thread": "main"
  }
}
```

## 功能特性

- ✅ 多格式日志接收（HTTP/gRPC）
- ✅ AI驱动的异常检测
- ✅ 智能根因分析和建议
- ✅ 大模型深度分析
- ✅ 实时数据可视化
- ✅ 告警通知（邮件/钉钉）
- ✅ Docker容器化部署
- ✅ 响应式Web界面

## 环境配置

创建`.env`文件并配置：
```bash
# LLM配置
SILICONFLOW_API_TOKEN=your_token_here

# 数据库配置
POSTGRES_DB=log_analysis
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# 服务端口
BACKEND_PORT=8080
FRONTEND_PORT=3000
LLM_API_PORT=5000
```

## 许可证

MIT License