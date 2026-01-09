#!/usr/bin/env python3
"""
测试LLM分析功能的脚本
"""

import os
import sys
import json
import requests
from datetime import datetime
from dotenv import load_dotenv

# 添加src目录到Python路径
sys.path.append(os.path.join(os.path.dirname(__file__), 'src'))

from core.llm_analyzer import LLMAnalyzer

# 加载环境变量
load_dotenv()

def test_llm_analyzer_direct():
    """直接测试LLM分析器"""
    print("=== 直接测试LLM分析器 ===")
    
    analyzer = LLMAnalyzer()
    
    # 测试数据
    test_logs = [
        {
            'id': '1',
            'timestamp': '2024-12-25T10:00:00Z',
            'level': 'ERROR',
            'message': 'Database connection failed: Connection timeout after 30 seconds',
            'source': 'database-service'
        },
        {
            'id': '2',
            'timestamp': '2024-12-25T10:01:00Z',
            'level': 'ERROR',
            'message': 'Failed to process user request: Internal server error',
            'source': 'api-gateway'
        },
        {
            'id': '3',
            'timestamp': '2024-12-25T10:02:00Z',
            'level': 'WARN',
            'message': 'High memory usage detected: 85% of available memory in use',
            'source': 'monitoring-service'
        },
        {
            'id': '4',
            'timestamp': '2024-12-25T10:03:00Z',
            'level': 'INFO',
            'message': 'User login successful',
            'source': 'auth-service'
        }
    ]
    
    anomaly_results = {
        'is_anomaly': [True, True, False, False],
        'scores': [0.9, 0.8, 0.3, 0.1]
    }
    
    # 测试批量分析
    print("测试批量分析...")
    batch_result = analyzer.analyze_logs_batch(test_logs, anomaly_results)
    print("批量分析结果:")
    print(json.dumps(batch_result, ensure_ascii=False, indent=2))
    
    print("\n" + "="*50 + "\n")
    
    # 测试单个日志分析
    print("测试单个日志分析...")
    single_result = analyzer.analyze_single_log(test_logs[0], test_logs[1:3])
    print("单个日志分析结果:")
    print(json.dumps(single_result, ensure_ascii=False, indent=2))

def test_llm_api_server():
    """测试LLM API服务器"""
    print("=== 测试LLM API服务器 ===")
    
    base_url = "http://localhost:5000"
    
    # 测试健康检查
    try:
        response = requests.get(f"{base_url}/health")
        print("健康检查结果:")
        print(json.dumps(response.json(), ensure_ascii=False, indent=2))
    except requests.exceptions.ConnectionError:
        print("无法连接到LLM API服务器。请先启动服务器:")
        print("cd python-ai && python src/llm_api.py")
        return
    
    # 测试批量分析API
    test_data = {
        'logs': [
            {
                'id': '1',
                'timestamp': '2024-12-25T10:00:00Z',
                'level': 'ERROR',
                'message': 'Database connection failed',
                'source': 'database-service'
            }
        ],
        'anomaly_results': {
            'is_anomaly': [True],
            'scores': [0.9]
        }
    }
    
    try:
        response = requests.post(f"{base_url}/analyze/batch", json=test_data)
        print("\n批量分析API结果:")
        print(json.dumps(response.json(), ensure_ascii=False, indent=2))
    except Exception as e:
        print(f"批量分析API测试失败: {e}")
    
    # 测试单个日志分析API
    single_data = {
        'log': {
            'id': '1',
            'timestamp': '2024-12-25T10:00:00Z',
            'level': 'ERROR',
            'message': 'Database connection failed',
            'source': 'database-service'
        }
    }
    
    try:
        response = requests.post(f"{base_url}/analyze/single", json=single_data)
        print("\n单个日志分析API结果:")
        print(json.dumps(response.json(), ensure_ascii=False, indent=2))
    except Exception as e:
        print(f"单个日志分析API测试失败: {e}")

def test_custom_prompt():
    """测试自定义提示词"""
    print("=== 测试自定义提示词 ===")
    
    base_url = "http://localhost:5000"
    
    custom_data = {
        'prompt': '''请分析以下系统日志并提供建议：

日志内容：
2024-12-25 10:00:00 ERROR [database-service] Connection pool exhausted: max_connections=100, active=100, idle=0
2024-12-25 10:00:01 ERROR [api-gateway] Request timeout: upstream server not responding
2024-12-25 10:00:02 WARN [monitoring] CPU usage spike: 95% for 30 seconds

请用JSON格式回复分析结果。'''
    }
    
    try:
        response = requests.post(f"{base_url}/analyze/custom", json=custom_data)
        print("自定义分析结果:")
        print(json.dumps(response.json(), ensure_ascii=False, indent=2))
    except requests.exceptions.ConnectionError:
        print("无法连接到LLM API服务器")
    except Exception as e:
        print(f"自定义分析测试失败: {e}")

if __name__ == '__main__':
    print("LLM分析功能测试")
    print("="*50)
    
    # 检查API Token配置
    if not os.getenv('SILICONFLOW_API_TOKEN'):
        print("警告: SILICONFLOW_API_TOKEN 未配置")
        print("请在 .env 文件中设置您的API Token")
        print()
    
    # 运行测试
    test_llm_analyzer_direct()
    print("\n" + "="*50 + "\n")
    test_llm_api_server()
    print("\n" + "="*50 + "\n")
    test_custom_prompt()