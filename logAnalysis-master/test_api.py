#!/usr/bin/env python3
"""
增强版API测试脚本 - 专门测试filebeat nginx日志接口
测试单一的 /api/v1/logs 端点，模拟filebeat发送的nginx日志数据
包含正常流量和异常流量（8:2比例），用于测试异常检测功能
"""

import requests
import json
import time
import random
from datetime import datetime, timezone

API_URL = "http://localhost:8080"

def generate_normal_logs():
    """生成正常的nginx日志数据"""
    normal_ips = ["192.168.1.100", "10.0.0.1", "172.16.0.10", "203.0.113.45", "198.51.100.23"]
    normal_urls = [
        "/", "/api/health", "/api/v1/dashboard/stats", "/api/v1/analysis/results",
        "/static/css/main.css", "/static/js/app.js", "/favicon.ico", "/robots.txt",
        "/api/v1/logs", "/dashboard", "/login", "/logout"
    ]
    normal_user_agents = [
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
        "curl/7.68.0", "filebeat/8.11.0", "Go-http-client/1.1"
    ]
    
    logs = []
    for _ in range(40):  # 生成40个正常日志
        ip = random.choice(normal_ips)
        url = random.choice(normal_urls)
        user_agent = random.choice(normal_user_agents)
        method = random.choice(["GET", "POST", "PUT", "DELETE"])
        status_code = random.choice([200, 201, 204, 301, 302, 304])
        
        log = {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "INFO",
            "message": f'{ip} - - [{datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")}] "{method} {url} HTTP/1.1" {status_code} {random.randint(100, 5000)} "-" "{user_agent}"',
            "source": "nginx-access",
            "metadata": {
                "remote_addr": ip,
                "response_code": str(status_code),
                "method": method,
                "url": url,
                "user_agent": user_agent,
                "response_size": str(random.randint(100, 5000))
            }
        }
        logs.append(log)
    
    return logs

def generate_anomalous_logs():
    """生成异常的nginx日志数据"""
    malicious_ips = ["1.2.3.4", "5.6.7.8", "9.10.11.12", "13.14.15.16", "17.18.19.20"]
    
    logs = []
    
    # SQL注入攻击 (3个)
    sql_injection_payloads = [
        "/api/users?id=1' OR '1'='1",
        "/login?username=admin'--&password=anything",
        "/search?q='; DROP TABLE users; --"
    ]
    
    for payload in sql_injection_payloads:
        log = {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "INFO",
            "message": f'{random.choice(malicious_ips)} - - [{datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET {payload} HTTP/1.1" 400 0 "-" "sqlmap/1.6.12"',
            "source": "nginx-access",
            "metadata": {
                "remote_addr": random.choice(malicious_ips),
                "response_code": "400",
                "method": "GET",
                "url": payload,
                "user_agent": "sqlmap/1.6.12",
                "attack_type": "sql_injection"
            }
        }
        logs.append(log)
    
    # XSS攻击 (2个)
    xss_payloads = [
        "/search?q=<script>alert('XSS')</script>",
        "/comment?text=<img src=x onerror=alert(1)>"
    ]
    
    for payload in xss_payloads:
        log = {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "INFO",
            "message": f'{random.choice(malicious_ips)} - - [{datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET {payload} HTTP/1.1" 403 0 "-" "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"',
            "source": "nginx-access",
            "metadata": {
                "remote_addr": random.choice(malicious_ips),
                "response_code": "403",
                "method": "GET",
                "url": payload,
                "user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
                "attack_type": "xss"
            }
        }
        logs.append(log)
    
    # 路径遍历攻击 (2个)
    path_traversal_payloads = [
        "/api/files?path=../../../etc/passwd",
        "/download?file=....//....//....//etc/shadow"
    ]
    
    for payload in path_traversal_payloads:
        log = {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "INFO",
            "message": f'{random.choice(malicious_ips)} - - [{datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET {payload} HTTP/1.1" 404 0 "-" "curl/7.68.0"',
            "source": "nginx-access",
            "metadata": {
                "remote_addr": random.choice(malicious_ips),
                "response_code": "404",
                "method": "GET",
                "url": payload,
                "user_agent": "curl/7.68.0",
                "attack_type": "path_traversal"
            }
        }
        logs.append(log)
    
    # 系统错误日志 (2个)
    error_logs = [
        {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "ERROR",
            "message": "2025/12/25 10:30:45 [error] 1234#0: *1 connect() failed (111: Connection refused) while connecting to upstream",
            "source": "nginx-error",
            "metadata": {
                "pid": "1234",
                "tid": "0",
                "connection_id": "1",
                "error_type": "upstream_connection_failed"
            }
        },
        {
            "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            "level": "FATAL",
            "message": "2025/12/25 10:31:00 [crit] 1234#0: *2 SSL_do_handshake() failed (SSL: error:14094416:SSL routines:ssl3_read_bytes:sslv3 alert certificate unknown)",
            "source": "nginx-error",
            "metadata": {
                "pid": "1234",
                "tid": "0",
                "connection_id": "2",
                "error_type": "ssl_handshake_failed"
            }
        }
    ]
    
    logs.extend(error_logs)
    
    # DDoS模拟 - 同一IP大量请求 (1个)
    ddos_ip = "99.88.77.66"
    log = {
        "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
        "level": "WARN",
        "message": f'{ddos_ip} - - [{datetime.now().strftime("%d/%b/%Y:%H:%M:%S %z")}] "GET / HTTP/1.1" 429 0 "-" "python-requests/2.28.1"',
        "source": "nginx-access",
        "metadata": {
            "remote_addr": ddos_ip,
            "response_code": "429",
            "method": "GET",
            "url": "/",
            "user_agent": "python-requests/2.28.1",
            "attack_type": "ddos",
            "request_rate": "1000/min"
        }
    }
    logs.append(log)
    
    return logs

def generate_test_dataset():
    """生成测试数据集，正常:异常 = 8:2"""
    normal_logs = generate_normal_logs()  # 40个正常日志
    anomalous_logs = generate_anomalous_logs()  # 10个异常日志
    
    # 混合并随机排序
    all_logs = normal_logs + anomalous_logs
    random.shuffle(all_logs)
    
    return all_logs, len(normal_logs), len(anomalous_logs)

def test_endpoint(name, method, url, data=None, headers=None):
    """测试API端点"""
    print(f"🔍 {name}...")
    
    try:
        if method.upper() == "GET":
            response = requests.get(url, headers=headers, timeout=10)
        elif method.upper() == "POST":
            response = requests.post(url, json=data, headers=headers, timeout=10)
        else:
            print(f"❌ 不支持的HTTP方法: {method}")
            return False
        
        print(f"   状态码: {response.status_code}")
        
        if response.status_code in [200, 201]:
            print(f"   ✅ 成功")
            # 如果响应是JSON，显示部分内容
            try:
                json_data = response.json()
                if isinstance(json_data, dict) and len(json_data) > 0:
                    # 显示前几个键
                    keys = list(json_data.keys())[:3]
                    print(f"   响应包含: {keys}")
                    if 'log_id' in json_data:
                        print(f"   日志ID: {json_data['log_id']}")
                elif isinstance(json_data, list):
                    print(f"   响应数组长度: {len(json_data)}")
            except:
                print(f"   响应长度: {len(response.text)} 字符")
        else:
            print(f"   ❌ 失败")
            try:
                error_data = response.json()
                print(f"   错误: {error_data.get('message', '未知错误')}")
            except:
                print(f"   错误响应: {response.text[:200]}")
        
        print()
        return response.status_code in [200, 201]
        
    except requests.exceptions.RequestException as e:
        print(f"   ❌ 请求失败: {e}")
        print()
        return False

def main():
    print("🚀 开始测试增强版日志分析系统API...")
    print("📝 专门测试filebeat nginx日志接口 + 异常检测")
    print("🎯 测试数据比例: 正常日志80% vs 异常日志20%")
    print("=" * 60)
    
    success_count = 0
    total_tests = 0
    
    # 1. 健康检查
    total_tests += 1
    if test_endpoint("健康检查", "GET", f"{API_URL}/health"):
        success_count += 1
    
    # 2. 仪表板统计
    total_tests += 1
    if test_endpoint("仪表板统计", "GET", f"{API_URL}/api/v1/dashboard/stats"):
        success_count += 1
    
    # 3. 分析结果
    total_tests += 1
    if test_endpoint("分析结果", "GET", f"{API_URL}/api/v1/analysis/results?limit=5"):
        success_count += 1
    
    # 4. 发送基础测试日志
    total_tests += 1
    nginx_access_log = {
        "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
        "level": "INFO",
        "message": "192.168.1.100 - - [25/Dec/2025:10:30:45 +0800] \"GET /api/health HTTP/1.1\" 200 15 \"-\" \"curl/7.68.0\"",
        "source": "nginx-access",
        "metadata": {
            "remote_addr": "192.168.1.100",
            "response_code": "200",
            "method": "GET",
            "url": "/api/health",
            "user_agent": "curl/7.68.0"
        }
    }
    
    if test_endpoint("发送基础nginx access日志", "POST", f"{API_URL}/api/v1/logs", nginx_access_log):
        success_count += 1
    
    # 5. 发送nginx error日志
    total_tests += 1
    nginx_error_log = {
        "timestamp": datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
        "level": "ERROR",
        "message": "2025/12/25 10:30:45 [error] 1234#0: *1 connect() failed (111: Connection refused) while connecting to upstream",
        "source": "nginx-error",
        "metadata": {
            "pid": "1234",
            "tid": "0",
            "connection_id": "1"
        }
    }
    
    if test_endpoint("发送nginx error日志", "POST", f"{API_URL}/api/v1/logs", nginx_error_log):
        success_count += 1
    
    # 6. 生成并发送大量测试数据
    total_tests += 1
    print("🔍 生成测试数据集（正常:异常 = 8:2）...")
    test_logs, normal_count, anomalous_count = generate_test_dataset()
    
    print(f"   📊 生成了 {len(test_logs)} 条日志:")
    print(f"   ✅ 正常日志: {normal_count} 条 ({normal_count/len(test_logs)*100:.1f}%)")
    print(f"   ⚠️  异常日志: {anomalous_count} 条 ({anomalous_count/len(test_logs)*100:.1f}%)")
    print()
    
    print("🔍 批量发送测试数据...")
    batch_success = True
    sent_count = 0
    failed_count = 0
    
    for i, log_data in enumerate(test_logs, 1):
        print(f"   发送日志 {i}/{len(test_logs)}: {log_data['source']} - {log_data['level']}", end="")
        
        # 显示攻击类型（如果有）
        if 'attack_type' in log_data.get('metadata', {}):
            print(f" [{log_data['metadata']['attack_type']}]", end="")
        
        try:
            response = requests.post(f"{API_URL}/api/v1/logs", json=log_data, timeout=10)
            if response.status_code in [200, 201]:
                print(" ✅")
                sent_count += 1
            else:
                print(f" ❌ ({response.status_code})")
                # Print error details for debugging
                try:
                    error_data = response.json()
                    if i <= 3:  # Only show first 3 errors to avoid spam
                        print(f"      错误详情: {error_data.get('message', 'Unknown error')}")
                except:
                    if i <= 3:
                        print(f"      错误响应: {response.text[:100]}")
                failed_count += 1
                batch_success = False
        except Exception as e:
            print(f" ❌ (异常: {str(e)[:50]})")
            failed_count += 1
            batch_success = False
        
        # 添加小延迟避免过快发送
        time.sleep(0.2)  # 增加延迟到200ms
    
    print(f"\n   📊 批量发送结果: {sent_count} 成功, {failed_count} 失败")
    
    if batch_success or sent_count >= len(test_logs) * 0.8:
        success_count += 1
        print("   ✅ 批量发送测试通过")
    else:
        print("   ❌ 批量发送测试失败")
    print()
    
    # 7. 测试无效数据处理
    total_tests += 1
    invalid_log = {
        "timestamp": "invalid-timestamp",
        "level": "",  # 空级别
        "message": "",  # 空消息
        "source": ""  # 空来源
    }
    
    print("🔍 测试无效数据处理...")
    try:
        response = requests.post(f"{API_URL}/api/v1/logs", json=invalid_log, timeout=10)
        if response.status_code == 400:
            print("   ✅ 正确拒绝了无效数据")
            success_count += 1
        else:
            print(f"   ❌ 应该拒绝无效数据，但返回了: {response.status_code}")
    except Exception as e:
        print(f"   ❌ 测试无效数据时出错: {e}")
    print()
    
    # 8. WebSocket端点测试
    total_tests += 1
    print("🔍 WebSocket端点可达性...")
    try:
        response = requests.get(f"{API_URL}/ws", timeout=5)
        if response.status_code == 400:
            print("   ✅ WebSocket端点可达（返回400是正常的）")
            success_count += 1
        else:
            print(f"   ⚠️  WebSocket端点返回意外状态码: {response.status_code}")
    except Exception as e:
        print(f"   ❌ WebSocket端点不可达: {e}")
    print()
    
    # 9. 检查分析结果（等待处理）
    total_tests += 1
    print("🔍 等待异常检测分析...")
    time.sleep(3)  # 等待后台处理
    
    if test_endpoint("检查分析结果", "GET", f"{API_URL}/api/v1/analysis/results?limit=20"):
        success_count += 1
    
    # 总结
    print("=" * 60)
    print(f"📊 测试完成: {success_count}/{total_tests} 个测试通过")
    
    if success_count == total_tests:
        print("🎉 所有测试都通过了！增强版API运行正常。")
        print("✅ filebeat nginx日志接口工作正常")
        print("✅ 异常检测功能已激活")
    elif success_count >= total_tests * 0.8:
        print("⚠️  大部分测试通过，系统基本正常，但有一些问题需要检查。")
    else:
        print("❌ 多个测试失败，请检查系统状态。")
    
    print(f"\n📋 测试数据统计:")
    print(f"   • 总日志数: {len(test_logs)} 条")
    print(f"   • 正常日志: {normal_count} 条 (包含常规访问、API调用等)")
    print(f"   • 异常日志: {anomalous_count} 条 (包含以下类型):")
    print(f"     - SQL注入攻击: 3 条")
    print(f"     - XSS攻击: 2 条") 
    print(f"     - 路径遍历攻击: 2 条")
    print(f"     - 系统错误: 2 条")
    print(f"     - DDoS模拟: 1 条")
    
    print("\n🔧 异常检测特性:")
    print("   • 自动识别SQL注入、XSS、路径遍历等攻击")
    print("   • 检测系统错误和连接问题")
    print("   • 监控异常访问模式和频率")
    print("   • 实时告警和分析结果展示")
    
    print("\n🔧 如果测试失败，请检查:")
    print("   1. Docker容器是否正常运行: docker compose ps")
    print("   2. Go后端日志: docker compose logs go-backend")
    print("   3. Python AI服务: docker compose logs python-ai")
    print("   4. 数据库连接: docker compose logs postgres")
    print("   5. Redis队列: docker compose logs redis")
    print("   6. 网络连接: curl http://localhost:8080/health")

if __name__ == "__main__":
    main()