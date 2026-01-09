import re
import pandas as pd
from datetime import datetime
from typing import List, Dict, Any
import logging
import math

class LogPreprocessor:
    """改进的日志预处理器 - 增强特征提取"""
    
    def __init__(self):
        self.common_patterns = [
            r'\d{4}-\d{2}-\d{2}',  # Date patterns
            r'\d{2}:\d{2}:\d{2}',  # Time patterns
            r'\b\d+\.\d+\.\d+\.\d+\b',  # IP addresses
            r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b',  # Email addresses
            r'\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b',  # UUIDs
        ]
        
        # 扩展的错误关键词
        self.critical_keywords = [
            'panic', 'crash', 'segmentation fault', 'out of memory',
            'connection refused', 'timeout', 'failed', 'exception',
            'critical', 'fatal', 'abort', 'denied', 'null pointer',
            'stack overflow', 'access denied', 'permission denied'
        ]
        
    def process(self, logs: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """处理日志列表，返回预处理后的日志"""
        processed_logs = []
        
        for log in logs:
            try:
                processed_log = self._process_single_log(log)
                processed_logs.append(processed_log)
            except Exception as e:
                logging.warning(f"Failed to process log {log.get('id', 'unknown')}: {str(e)}")
                # Return original log with minimal processing
                processed_logs.append(self._create_fallback_log(log))
                
        return processed_logs
    
    def _process_single_log(self, log: Dict[str, Any]) -> Dict[str, Any]:
        """处理单条日志"""
        processed = log.copy()
        
        # Clean and normalize message
        processed['cleaned_message'] = self._clean_message(log['message'])
        
        # Extract enhanced features
        processed['features'] = self._extract_enhanced_features(log)
        
        # Normalize timestamp
        processed['normalized_timestamp'] = self._normalize_timestamp(log['timestamp'])
        
        # Extract patterns
        processed['patterns'] = self._extract_patterns(log['message'])
        
        return processed
    
    def _clean_message(self, message: str) -> str:
        """清理日志消息"""
        if not message:
            return ""
            
        # Remove extra whitespace
        cleaned = re.sub(r'\s+', ' ', message.strip())
        
        # Remove control characters
        cleaned = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', cleaned)
        
        return cleaned
    
    def _extract_enhanced_features(self, log: Dict[str, Any]) -> Dict[str, Any]:
        """提取增强的日志特征"""
        message = log.get('message', '')
        level = log.get('level', '').upper()
        source = log.get('source', '')
        
        # 原有特征
        features = {
            'message_length': len(message),
            'word_count': len(message.split()) if message else 0,
            'level_numeric': self._level_to_numeric(level),
            'has_error_keywords': self._has_error_keywords(message),
            'has_warning_keywords': self._has_warning_keywords(message),
            'has_numbers': bool(re.search(r'\d+', message)),
            'has_special_chars': bool(re.search(r'[!@#$%^&*(),.?":{}|<>]', message)),
            'source_hash': hash(source) % 1000,  # Simple source encoding
        }
        
        # 新增增强特征
        features.update({
            'critical_keyword_count': self._count_critical_keywords(message),
            'error_pattern_score': self._calculate_error_pattern_score(message),
            'stack_trace_indicator': self._detect_stack_trace(message),
            'is_error_level': level in ['ERROR', 'FATAL', 'CRITICAL'],
            'message_entropy': self._calculate_message_entropy(message),
            'has_ip_address': bool(re.search(r'\b\d+\.\d+\.\d+\.\d+\b', message)),
            'has_timestamp': bool(re.search(r'\d{2}:\d{2}:\d{2}', message)),
            'uppercase_ratio': self._calculate_uppercase_ratio(message),
            'numeric_ratio': self._calculate_numeric_ratio(message),
        })
        
        return features
    
    def _count_critical_keywords(self, message: str) -> int:
        """统计关键错误关键词数量"""
        message_lower = message.lower()
        count = 0
        for keyword in self.critical_keywords:
            if keyword in message_lower:
                count += 1
        return count
    
    def _calculate_error_pattern_score(self, message: str) -> float:
        """计算错误模式分数"""
        message_lower = message.lower()
        score = 0.0
        
        # 检查各种错误模式
        error_patterns = {
            'connection': 0.3,
            'timeout': 0.4,
            'failed': 0.2,
            'error': 0.1,
            'exception': 0.3,
            'crash': 0.5,
            'panic': 0.6,
            'fatal': 0.5,
            'critical': 0.4,
        }
        
        for pattern, weight in error_patterns.items():
            if pattern in message_lower:
                score += weight
        
        return min(score, 1.0)  # 限制在[0,1]范围内
    
    def _detect_stack_trace(self, message: str) -> float:
        """检测堆栈跟踪指示器"""
        stack_indicators = [
            'at ', 'in ', 'line ', 'file ', '.java:', '.py:', '.cpp:',
            'stacktrace', 'traceback', 'caused by'
        ]
        
        message_lower = message.lower()
        indicator_count = sum(1 for indicator in stack_indicators if indicator in message_lower)
        
        return min(indicator_count / 3.0, 1.0)  # 标准化到[0,1]
    
    def _calculate_message_entropy(self, message: str) -> float:
        """计算消息熵（复杂度指标）"""
        if not message:
            return 0.0
        
        # 计算字符频率
        char_counts = {}
        for char in message:
            char_counts[char] = char_counts.get(char, 0) + 1
        
        # 计算熵
        entropy = 0.0
        message_length = len(message)
        
        for count in char_counts.values():
            probability = count / message_length
            if probability > 0:
                entropy -= probability * math.log2(probability)
        
        # 标准化熵值（假设最大熵约为5）
        return min(entropy / 5.0, 1.0)
    
    def _calculate_uppercase_ratio(self, message: str) -> float:
        """计算大写字母比例"""
        if not message:
            return 0.0
        
        uppercase_count = sum(1 for char in message if char.isupper())
        return uppercase_count / len(message)
    
    def _calculate_numeric_ratio(self, message: str) -> float:
        """计算数字字符比例"""
        if not message:
            return 0.0
        
        numeric_count = sum(1 for char in message if char.isdigit())
        return numeric_count / len(message)
    
    def _normalize_timestamp(self, timestamp_str: str) -> str:
        """标准化时间戳"""
        try:
            # Try to parse various timestamp formats
            formats = [
                '%Y-%m-%dT%H:%M:%S.%fZ',
                '%Y-%m-%dT%H:%M:%SZ',
                '%Y-%m-%d %H:%M:%S',
                '%Y-%m-%d %H:%M:%S.%f',
            ]
            
            for fmt in formats:
                try:
                    dt = datetime.strptime(timestamp_str, fmt)
                    return dt.isoformat()
                except ValueError:
                    continue
                    
            # If all formats fail, return original
            return timestamp_str
            
        except Exception:
            return timestamp_str
    
    def _extract_patterns(self, message: str) -> List[str]:
        """提取消息中的模式"""
        patterns = []
        
        for pattern in self.common_patterns:
            matches = re.findall(pattern, message)
            if matches:
                patterns.extend(matches)
                
        return patterns
    
    def _level_to_numeric(self, level: str) -> int:
        """将日志级别转换为数值（增加权重差异）"""
        level_map = {
            'DEBUG': 1,
            'INFO': 2,
            'WARN': 4,      # 增加权重
            'WARNING': 4,
            'ERROR': 8,     # 显著增加权重
            'FATAL': 10,    # 最高权重
            'CRITICAL': 10,
        }
        return level_map.get(level.upper(), 0)
    
    def _has_error_keywords(self, message: str) -> bool:
        """检查是否包含错误关键词（扩展列表）"""
        error_keywords = [
            'error', 'exception', 'failed', 'failure', 'crash', 'fatal',
            'critical', 'panic', 'abort', 'timeout', 'refused', 'denied',
            'null pointer', 'segmentation fault', 'out of memory', 'stack overflow'
        ]
        message_lower = message.lower()
        return any(keyword in message_lower for keyword in error_keywords)
    
    def _has_warning_keywords(self, message: str) -> bool:
        """检查是否包含警告关键词"""
        warning_keywords = [
            'warning', 'warn', 'deprecated', 'slow', 'retry', 'fallback',
            'degraded', 'limited', 'throttled', 'temporary', 'disabled'
        ]
        message_lower = message.lower()
        return any(keyword in message_lower for keyword in warning_keywords)
    
    def _create_fallback_log(self, log: Dict[str, Any]) -> Dict[str, Any]:
        """创建备用的最小处理日志"""
        return {
            **log,
            'cleaned_message': log.get('message', ''),
            'features': {
                'message_length': len(log.get('message', '')),
                'level_numeric': self._level_to_numeric(log.get('level', '')),
                'has_error_keywords': False,
                'has_warning_keywords': False,
                'critical_keyword_count': 0,
                'error_pattern_score': 0,
                'stack_trace_indicator': 0,
                'is_error_level': False,
                'message_entropy': 0,
            },
            'normalized_timestamp': log.get('timestamp', ''),
            'patterns': []
        }
