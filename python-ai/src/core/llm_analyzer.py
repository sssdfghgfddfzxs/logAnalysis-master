import os
import json
import logging
import requests
import time
from typing import List, Dict, Any, Optional
from datetime import datetime

class LLMAnalyzer:
    """大模型分析器 - 使用SiliconFlow API进行深度日志分析"""
    
    def __init__(self):
        self.api_url = "https://api.siliconflow.cn/v1/chat/completions"
        self.api_token = os.getenv('SILICONFLOW_API_TOKEN')
        self.model = "Qwen/QwQ-32B"
        
        if not self.api_token:
            logging.warning("SILICONFLOW_API_TOKEN not found in environment variables")
    
    def analyze_logs_batch(self, logs: List[Dict[str, Any]], anomaly_results: Dict[str, List] = None) -> Dict[str, Any]:
        """批量分析日志，一次性完成异常检测和根因分析"""
        if not self.api_token:
            logging.info("LLM analysis skipped: API token not configured")
            return self._get_default_comprehensive_analysis(logs)
        
        try:
            # 构建综合分析提示词
            prompt = self._build_comprehensive_analysis_prompt(logs)
            
            # 调用大模型API
            response = self._call_llm_api(prompt)
            
            if response:
                return self._parse_comprehensive_response(response, logs)
            else:
                return self._get_default_comprehensive_analysis(logs)
                
        except Exception as e:
            logging.warning(f"LLM comprehensive analysis failed, using fallback: {str(e)}")
            return self._get_default_comprehensive_analysis(logs)
    
    def analyze_single_log(self, log: Dict[str, Any], context_logs: List[Dict[str, Any]] = None) -> Dict[str, Any]:
        """分析单个日志条目"""
        if not self.api_token:
            return self._get_default_single_analysis()
        
        try:
            prompt = self._build_single_log_prompt(log, context_logs)
            response = self._call_llm_api(prompt)
            
            if response:
                return self._parse_single_log_response(response)
            else:
                return self._get_default_single_analysis()
                
        except Exception as e:
            logging.warning(f"Single log LLM analysis failed: {str(e)}")
            return self._get_default_single_analysis()
    
    def _prepare_analysis_data(self, logs: List[Dict[str, Any]], anomaly_results: Dict[str, List]) -> Dict[str, Any]:
        """准备分析数据"""
        anomalies = []
        normal_logs = []
        error_logs = []
        
        for i, log in enumerate(logs):
            log_data = {
                'timestamp': log.get('timestamp', ''),
                'level': log.get('level', ''),
                'message': log.get('message', ''),
                'source': log.get('source', ''),
                'anomaly_score': anomaly_results['scores'][i] if i < len(anomaly_results['scores']) else 0
            }
            
            if anomaly_results['is_anomaly'][i] if i < len(anomaly_results['is_anomaly']) else False:
                anomalies.append(log_data)
            elif log.get('level', '').upper() in ['ERROR', 'FATAL', 'CRITICAL']:
                error_logs.append(log_data)
            else:
                normal_logs.append(log_data)
        
        return {
            'total_logs': len(logs),
            'anomalies': anomalies[:10],  # 限制数量避免token过多
            'error_logs': error_logs[:10],
            'normal_logs': normal_logs[:5],
            'anomaly_rate': len(anomalies) / len(logs) if logs else 0
        }
    
    def _build_analysis_prompt(self, data: Dict[str, Any]) -> str:
        """构建分析提示词"""
        prompt = f"""你是一个专业的日志分析专家。请分析以下日志数据并提供深度洞察。

## 日志统计
- 总日志数: {data['total_logs']}
- 异常日志数: {len(data['anomalies'])}
- 错误日志数: {len(data['error_logs'])}
- 异常率: {data['anomaly_rate']:.2%}

## 异常日志样本
{json.dumps(data['anomalies'], ensure_ascii=False, indent=2)}

## 错误日志样本
{json.dumps(data['error_logs'], ensure_ascii=False, indent=2)}

## 正常日志样本
{json.dumps(data['normal_logs'], ensure_ascii=False, indent=2)}

请提供以下分析结果（请用JSON格式回复）：
{{
    "summary": "整体分析摘要",
    "key_findings": ["关键发现1", "关键发现2", "关键发现3"],
    "risk_assessment": {{
        "level": "LOW/MEDIUM/HIGH",
        "description": "风险评估描述"
    }},
    "patterns": ["发现的模式1", "发现的模式2"],
    "recommendations": ["建议1", "建议2", "建议3"],
    "potential_issues": ["潜在问题1", "潜在问题2"],
    "trend_analysis": "趋势分析"
}}"""
        
        return prompt
    
    def _build_single_log_prompt(self, log: Dict[str, Any], context_logs: List[Dict[str, Any]] = None) -> str:
        """构建单个日志分析提示词"""
        context_info = ""
        if context_logs:
            context_info = f"\n## 上下文日志\n{json.dumps(context_logs[-5:], ensure_ascii=False, indent=2)}"
        
        prompt = f"""你是一个专业的日志分析专家。请分析以下单个日志条目：

## 目标日志
{json.dumps(log, ensure_ascii=False, indent=2)}
{context_info}

请提供以下分析结果（请用JSON格式回复）：
{{
    "severity": "LOW/MEDIUM/HIGH/CRITICAL",
    "category": "日志类别（如：系统错误、业务异常、性能问题等）",
    "root_cause": "可能的根本原因",
    "impact": "潜在影响",
    "immediate_actions": ["立即行动建议1", "立即行动建议2"],
    "investigation_steps": ["调查步骤1", "调查步骤2"],
    "related_components": ["相关组件1", "相关组件2"]
}}"""
        
        return prompt
    
    def _call_llm_api(self, prompt: str) -> Optional[str]:
        """调用大模型API"""
        try:
            logging.info(f"Starting LLM API call to {self.api_url}")
            logging.debug(f"Using model: {self.model}")
            logging.debug(f"Prompt length: {len(prompt)} characters")
            
            headers = {
                'Authorization': f'Bearer {self.api_token[:10]}...',  # 只显示token前10位用于调试
                'Content-Type': 'application/json'
            }
            
            data = {
                "model": self.model,
                "messages": [
                    {
                        "role": "user",
                        "content": prompt
                    }
                ],
                "stream": False,
                "temperature": 0.1,  # 降低随机性，提高分析的一致性
                "max_tokens": 2000
            }
            
            logging.info("Sending request to LLM API...")
            start_time = time.time()
            
            response = requests.post(
                self.api_url,
                headers={
                    'Authorization': f'Bearer {self.api_token}',
                    'Content-Type': 'application/json'
                },
                json=data,
                timeout=120
            )
            
            elapsed_time = time.time() - start_time
            logging.info(f"LLM API response received in {elapsed_time:.2f} seconds")
            logging.info(f"Response status code: {response.status_code}")
            
            if response.status_code == 200:
                result = response.json()
                content = result['choices'][0]['message']['content']
                logging.info(f"LLM API call successful, response length: {len(content)} characters")
                logging.debug(f"Response preview: {content[:200]}...")
                return content
            else:
                logging.error(f"LLM API error: {response.status_code} - {response.text}")
                return None
                
        except requests.exceptions.Timeout as e:
            logging.error(f"LLM API timeout after 120 seconds: {str(e)}")
            logging.error("Consider increasing timeout or checking network connectivity")
            return None
        except requests.exceptions.ConnectionError as e:
            logging.error(f"LLM API connection error: {str(e)}")
            logging.error("Check if the API endpoint is accessible")
            return None
        except requests.exceptions.RequestException as e:
            logging.error(f"LLM API request error: {str(e)}")
            return None
        except Exception as e:
            logging.error(f"Unexpected error calling LLM API: {str(e)}")
            logging.error(f"Error type: {type(e).__name__}")
            return None
    
    def _parse_llm_response(self, response: str) -> Dict[str, Any]:
        """解析大模型响应"""
        try:
            # 尝试提取JSON部分
            start_idx = response.find('{')
            end_idx = response.rfind('}') + 1
            
            if start_idx != -1 and end_idx != -1:
                json_str = response[start_idx:end_idx]
                return json.loads(json_str)
            else:
                # 如果没有找到JSON，返回文本分析
                return {
                    "summary": response[:500],
                    "key_findings": ["LLM分析结果需要人工解读"],
                    "risk_assessment": {"level": "MEDIUM", "description": "需要进一步分析"},
                    "patterns": [],
                    "recommendations": ["请查看完整的LLM分析结果"],
                    "potential_issues": [],
                    "trend_analysis": "分析结果格式异常"
                }
                
        except json.JSONDecodeError as e:
            logging.error(f"Error parsing LLM response: {str(e)}")
            return self._get_default_analysis()
    
    def _parse_single_log_response(self, response: str) -> Dict[str, Any]:
        """解析单个日志的大模型响应"""
        try:
            start_idx = response.find('{')
            end_idx = response.rfind('}') + 1
            
            if start_idx != -1 and end_idx != -1:
                json_str = response[start_idx:end_idx]
                return json.loads(json_str)
            else:
                return {
                    "severity": "MEDIUM",
                    "category": "未分类",
                    "root_cause": response[:200],
                    "impact": "需要进一步分析",
                    "immediate_actions": ["查看完整分析结果"],
                    "investigation_steps": ["人工审查"],
                    "related_components": []
                }
                
        except json.JSONDecodeError as e:
            logging.error(f"Error parsing single log LLM response: {str(e)}")
            return self._get_default_single_analysis()
    
    def _get_default_analysis(self) -> Dict[str, Any]:
        """获取默认分析结果"""
        return {
            "summary": "使用传统AI分析方法",
            "key_findings": ["基于规则的异常检测"],
            "risk_assessment": {"level": "MEDIUM", "description": "需要人工确认"},
            "patterns": [],
            "recommendations": ["检查系统资源使用情况"],
            "potential_issues": [],
            "trend_analysis": "传统分析模式"
        }
    
    def _get_default_single_analysis(self) -> Dict[str, Any]:
        """获取默认单个日志分析结果"""
        return {
            "severity": "MEDIUM",
            "category": "系统日志",
            "root_cause": "需要进一步分析",
            "impact": "可能影响系统稳定性",
            "immediate_actions": ["检查相关服务状态", "查看系统资源"],
            "investigation_steps": ["分析日志上下文", "检查系统指标"],
            "related_components": []
        }
    def _build_comprehensive_analysis_prompt(self, logs: List[Dict[str, Any]]) -> str:
        """构建优化的综合分析提示词，提升分析准确性"""
        logs_data = []
        for i, log in enumerate(logs[:15]):  # 适当减少数量，提升分析质量
            logs_data.append({
                'index': i,
                'timestamp': log.get('timestamp', ''),
                'level': log.get('level', ''),
                'message': log.get('message', ''),
                'source': log.get('source', '')
            })
        
        prompt = f"""你是一个资深的系统运维和日志分析专家，拥有丰富的故障诊断经验。请对以下日志进行专业的综合分析。

## 日志数据
{json.dumps(logs_data, ensure_ascii=False, indent=2)}

## 分析要求
请为每个日志提供精准的分析，并给出整体评估。重点关注：
1. **异常识别**：基于日志内容、级别、错误模式进行准确判断
2. **根因分析**：深入分析可能的技术原因和业务影响
3. **解决建议**：提供具体可执行的修复步骤
4. **风险评估**：评估对系统稳定性和业务的影响程度

## 异常判断标准
- **高风险异常**：FATAL/CRITICAL级别，系统崩溃、内存溢出、连接拒绝、安全攻击
- **中风险异常**：ERROR级别，业务逻辑错误、超时、认证失败、资源不足
- **低风险异常**：WARN级别，性能下降、配置问题、重试成功
- **正常日志**：INFO/DEBUG级别，无错误关键词的常规操作日志

请用以下JSON格式回复：

{{
    "individual_results": [
        {{
            "index": 0,
            "is_anomaly": true/false,
            "anomaly_score": 0.0-1.0,
            "severity": "CRITICAL/HIGH/MEDIUM/LOW",
            "category": "系统错误/业务异常/性能问题/安全问题/配置问题/网络问题",
            "root_causes": ["具体的技术原因1", "可能的业务原因2"],
            "recommendations": ["立即执行的修复步骤1", "预防措施2"],
            "impact": "对系统和业务的具体影响描述"
        }}
    ],
    "summary": "整体系统健康状况和主要问题总结",
    "key_findings": ["最重要的发现1", "关键问题2", "趋势观察3"],
    "risk_assessment": {{
        "level": "CRITICAL/HIGH/MEDIUM/LOW",
        "description": "当前系统风险状况的专业评估",
        "urgent_actions": ["需要立即处理的问题1", "紧急措施2"]
    }},
    "recommendations": ["系统级优化建议1", "监控改进建议2", "预防措施3"],
    "anomaly_count": 异常日志数量,
    "total_count": 总日志数量,
    "trend_analysis": "基于日志模式的趋势分析和预测"
}}

请确保分析结果专业、准确、可执行。"""
        
        return prompt

    def _parse_comprehensive_response(self, response: str, logs: List[Dict[str, Any]]) -> Dict[str, Any]:
        """解析综合分析响应"""
        try:
            # 尝试提取JSON部分
            start_idx = response.find('{')
            end_idx = response.rfind('}') + 1
            
            if start_idx != -1 and end_idx != -1:
                json_str = response[start_idx:end_idx]
                result = json.loads(json_str)
            else:
                raise json.JSONDecodeError("No JSON found", response, 0)
            
            # 确保individual_results的数量与输入日志匹配
            individual_results = result.get('individual_results', [])
            while len(individual_results) < len(logs):
                # 为缺失的日志添加默认分析结果
                log = logs[len(individual_results)]
                default_result = {
                    "index": len(individual_results),
                    "is_anomaly": log.get('level', '').upper() in ['ERROR', 'FATAL', 'CRITICAL'],
                    "anomaly_score": 0.8 if log.get('level', '').upper() in ['ERROR', 'FATAL', 'CRITICAL'] else 0.1,
                    "root_causes": [f"🤖 {log.get('level', 'INFO')}级别日志"],
                    "recommendations": ["🔍 建议查看详细信息"],
                    "severity": "HIGH" if log.get('level', '').upper() in ['ERROR', 'FATAL', 'CRITICAL'] else "LOW"
                }
                individual_results.append(default_result)
            
            result['individual_results'] = individual_results
            return result
            
        except json.JSONDecodeError as e:
            logging.warning(f"Failed to parse LLM response as JSON: {e}")
            return self._get_default_comprehensive_analysis(logs)
        except Exception as e:
            logging.warning(f"Error parsing comprehensive response: {e}")
            return self._get_default_comprehensive_analysis(logs)

    def _get_default_comprehensive_analysis(self, logs: List[Dict[str, Any]]) -> Dict[str, Any]:
        """获取默认的综合分析结果"""
        individual_results = []
        anomaly_count = 0
        
        for i, log in enumerate(logs):
            level = log.get('level', '').upper()
            is_anomaly = level in ['ERROR', 'FATAL', 'CRITICAL']
            if is_anomaly:
                anomaly_count += 1
            
            individual_results.append({
                "index": i,
                "is_anomaly": is_anomaly,
                "anomaly_score": 0.8 if is_anomaly else 0.1,
                "root_causes": [f"📊 传统分析: {level}级别日志"],
                "recommendations": ["🔍 建议进一步分析"] if is_anomaly else ["✅ 日志正常"],
                "severity": "HIGH" if is_anomaly else "LOW"
            })
        
        return {
            "individual_results": individual_results,
            "summary": f"分析了{len(logs)}条日志，发现{anomaly_count}个异常",
            "key_findings": ["使用传统规则分析", "基于日志级别判断异常"],
            "risk_assessment": {
                "level": "HIGH" if anomaly_count > 0 else "LOW",
                "description": f"发现{anomaly_count}个异常日志" if anomaly_count > 0 else "所有日志正常"
            },
            "recommendations": ["建议启用LLM深度分析获得更准确的结果"],
            "anomaly_count": anomaly_count,
            "total_count": len(logs)
        }