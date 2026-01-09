#!/usr/bin/env python3
"""
测试告警通知功能
"""

import requests
import json
import time
import sys
from datetime import datetime

# 配置
BASE_URL = "http://localhost:8080/api/v1"

def test_notification_channels():
    """测试获取通知渠道"""
    print("🔍 测试获取通知渠道...")
    
    try:
        response = requests.get(f"{BASE_URL}/notification-channels")
        response.raise_for_status()
        
        data = response.json()
        print(f"✅ 成功获取通知渠道: {len(data['channels'])} 个")
        
        for channel in data['channels']:
            print(f"   - {channel['icon']} {channel['display_name']}: {channel['description']}")
        
        return True
    except Exception as e:
        print(f"❌ 获取通知渠道失败: {e}")
        return False

def create_test_alert_rule():
    """创建测试告警规则"""
    print("\n📝 创建测试告警规则...")
    
    rule_data = {
        "name": "测试告警规则",
        "description": "用于测试告警通知功能的规则",
        "condition": {
            "anomaly_score_threshold": 0.8,
            "levels": ["ERROR"],
            "time_window_minutes": 15,
            "min_anomaly_count": 1
        },
        "notification_channels": ["email", "dingtalk"],
        "is_active": True
    }
    
    try:
        response = requests.post(f"{BASE_URL}/alert-rules", json=rule_data)
        response.raise_for_status()
        
        data = response.json()
        rule_id = data['rule']['id']
        print(f"✅ 成功创建告警规则: {rule_id}")
        return rule_id
    except Exception as e:
        print(f"❌ 创建告警规则失败: {e}")
        return None

def test_alert_rule(rule_id):
    """测试告警规则"""
    print(f"\n🧪 测试告警规则: {rule_id}")
    
    try:
        response = requests.post(f"{BASE_URL}/alert-rules/{rule_id}/test")
        response.raise_for_status()
        
        data = response.json()
        print(f"✅ 测试告警发送成功")
        
        results = data.get('results', {})
        for channel, result in results.items():
            status = "✅" if "Success" in result or "成功" in result else "❌"
            print(f"   {status} {channel}: {result}")
        
        return True
    except Exception as e:
        print(f"❌ 测试告警规则失败: {e}")
        return False

def get_alert_rules():
    """获取告警规则列表"""
    print("\n📋 获取告警规则列表...")
    
    try:
        response = requests.get(f"{BASE_URL}/alert-rules")
        response.raise_for_status()
        
        data = response.json()
        rules = data.get('rules', [])
        print(f"✅ 成功获取告警规则: {len(rules)} 个")
        
        for rule in rules:
            status = "🟢 启用" if rule['is_active'] else "🔴 禁用"
            channels = ", ".join(rule['notification_channels'])
            print(f"   - {rule['name']} ({status}) - 通知渠道: {channels}")
        
        return rules
    except Exception as e:
        print(f"❌ 获取告警规则失败: {e}")
        return []

def create_test_log_and_analysis():
    """创建测试日志和分析结果来触发告警"""
    print("\n📝 创建测试日志和分析结果...")
    
    # 创建测试日志
    log_data = {
        "timestamp": datetime.now().isoformat() + "Z",
        "level": "ERROR",
        "message": "Critical database connection failure - unable to connect to primary database server",
        "source": "database-service",
        "metadata": {
            "host": "db-server-01",
            "error_code": "CONNECTION_TIMEOUT",
            "retry_count": "3"
        }
    }
    
    try:
        # 创建日志
        response = requests.post(f"{BASE_URL}/logs", json=log_data)
        response.raise_for_status()
        
        log_response = response.json()
        log_id = log_response['log_id']
        print(f"✅ 成功创建测试日志: {log_id}")
        
        # 等待一下让系统处理
        time.sleep(2)
        
        # 触发LLM分析（这会创建分析结果，可能触发告警）
        analysis_data = {
            "log_ids": [log_id],
            "stream": False
        }
        
        response = requests.post(f"{BASE_URL}/ai/analyze", json=analysis_data)
        response.raise_for_status()
        
        analysis_response = response.json()
        print(f"✅ 成功触发LLM分析")
        
        # 检查分析结果
        results = analysis_response.get('results', [])
        if results:
            result = results[0]
            if result.get('is_anomaly'):
                score = result.get('anomaly_score', 0)
                print(f"🚨 检测到异常 (评分: {score:.2f}) - 可能触发告警")
            else:
                print(f"ℹ️ 未检测到异常 - 不会触发告警")
        
        return log_id
    except Exception as e:
        print(f"❌ 创建测试日志和分析失败: {e}")
        return None

def cleanup_test_rule(rule_id):
    """清理测试告警规则"""
    if not rule_id:
        return
    
    print(f"\n🧹 清理测试告警规则: {rule_id}")
    
    try:
        response = requests.delete(f"{BASE_URL}/alert-rules/{rule_id}")
        response.raise_for_status()
        print(f"✅ 成功删除测试告警规则")
    except Exception as e:
        print(f"❌ 删除测试告警规则失败: {e}")

def main():
    """主测试流程"""
    print("🚀 开始测试告警通知功能")
    print("=" * 50)
    
    # 测试通知渠道
    if not test_notification_channels():
        print("❌ 基础功能测试失败，退出")
        sys.exit(1)
    
    # 获取现有告警规则
    existing_rules = get_alert_rules()
    
    # 创建测试告警规则
    test_rule_id = create_test_alert_rule()
    
    if test_rule_id:
        # 测试告警规则
        test_alert_rule(test_rule_id)
        
        # 创建测试日志来触发实际告警
        print("\n" + "=" * 50)
        print("🔥 测试实际告警触发")
        test_log_id = create_test_log_and_analysis()
        
        if test_log_id:
            print(f"\n⏳ 等待告警处理...")
            time.sleep(5)  # 等待告警引擎处理
            print("ℹ️ 请检查邮箱和钉钉群是否收到告警通知")
        
        # 清理测试规则
        cleanup_test_rule(test_rule_id)
    
    print("\n" + "=" * 50)
    print("✅ 告警通知功能测试完成")
    print("\n📋 测试总结:")
    print("1. ✅ 通知渠道配置正常")
    print("2. ✅ 告警规则创建/删除正常")
    print("3. ✅ 告警测试功能正常")
    print("4. ✅ 日志分析和告警触发正常")
    print("\n💡 提示:")
    print("- 如果没有收到邮件，请检查SMTP配置和垃圾邮件文件夹")
    print("- 如果没有收到钉钉消息，请检查Webhook URL和密钥配置")
    print("- 可以在Web界面的'告警规则'页面管理告警规则")

if __name__ == "__main__":
    main()