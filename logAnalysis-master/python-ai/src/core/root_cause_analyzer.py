import re
from typing import Dict, List, Any
import logging

class RootCauseAnalyzer:
    """根因分析器 - 基于模式匹配和规则推荐"""
    
    def __init__(self):
        self.error_patterns = {
            'database': {
                'patterns': [
                    r'database.*\w+',
                    r'connection.*timeout',
                    r'database.*not.*available',
                    r'sql.*error',
                    r'deadlock',
                    r'connection.*refused',
                    r'too many connections',
                    r'db.*connection.*failed',
                    r'mysql.*error',
                    r'postgresql.*error',
                    r'oracle.*error',
                    r'sqlite.*error',
                    r'table.*not.*found',
                    r'duplicate.*key',
                    r'constraint.*violation'
                ],
                'root_causes': [
                    'Database connection timeout',
                    'Database server unavailable',
                    'SQL query error',
                    'Database deadlock',
                    'Connection pool exhausted',
                    'Database schema issue',
                    'Data integrity violation'
                ],
                'recommendations': [
                    'Check database connectivity and network',
                    'Increase connection timeout settings',
                    'Review and optimize SQL queries',
                    'Monitor database performance metrics',
                    'Scale database connection pool',
                    'Verify database schema and migrations',
                    'Review data validation logic'
                ]
            },
            'network': {
                'patterns': [
                    r'network.*\w+',
                    r'connection.*reset',
                    r'host.*unreachable',
                    r'timeout.*exceeded',
                    r'dns.*resolution.*failed',
                    r'socket.*timeout',
                    r'connection.*aborted',
                    r'network.*unreachable',
                    r'port.*unreachable',
                    r'ssl.*handshake.*failed',
                    r'certificate.*error',
                    r'tls.*error'
                ],
                'root_causes': [
                    'Network connectivity issue',
                    'Connection reset by peer',
                    'Host unreachable',
                    'Network timeout',
                    'DNS resolution failure',
                    'SSL/TLS certificate issue',
                    'Firewall blocking connection'
                ],
                'recommendations': [
                    'Check network connectivity',
                    'Verify firewall and security group settings',
                    'Test DNS resolution',
                    'Monitor network latency',
                    'Implement retry mechanisms with backoff',
                    'Verify SSL/TLS certificates',
                    'Check proxy and load balancer configuration'
                ]
            },
            'memory': {
                'patterns': [
                    r'out.*of.*memory',
                    r'memory.*leak',
                    r'heap.*space',
                    r'stack.*overflow',
                    r'gc.*overhead'
                ],
                'root_causes': [
                    'Memory exhaustion',
                    'Memory leak detected',
                    'Heap space insufficient',
                    'Stack overflow',
                    'Garbage collection overhead'
                ],
                'recommendations': [
                    'Increase memory allocation',
                    'Profile application for memory leaks',
                    'Optimize data structures and algorithms',
                    'Review recursive function calls',
                    'Tune garbage collection parameters'
                ]
            },
            'authentication': {
                'patterns': [
                    r'authentication.*failed',
                    r'unauthorized',
                    r'access.*denied',
                    r'invalid.*credentials',
                    r'token.*expired'
                ],
                'root_causes': [
                    'Authentication failure',
                    'Unauthorized access attempt',
                    'Invalid credentials provided',
                    'Expired authentication token',
                    'Insufficient permissions'
                ],
                'recommendations': [
                    'Verify user credentials',
                    'Check authentication service status',
                    'Review access control policies',
                    'Implement token refresh mechanism',
                    'Monitor for suspicious access patterns'
                ]
            },
            'performance': {
                'patterns': [
                    r'slow.*query',
                    r'response.*time.*exceeded',
                    r'performance.*degraded',
                    r'high.*cpu',
                    r'thread.*pool.*exhausted',
                    r'request.*timeout',
                    r'processing.*slow',
                    r'latency.*high',
                    r'throughput.*low',
                    r'queue.*full',
                    r'backlog.*growing'
                ],
                'root_causes': [
                    'Slow database query',
                    'Response time exceeded threshold',
                    'Performance degradation',
                    'High CPU utilization',
                    'Thread pool exhaustion',
                    'Resource contention',
                    'System overload'
                ],
                'recommendations': [
                    'Optimize database queries and indexes',
                    'Scale application resources',
                    'Implement caching strategies',
                    'Monitor system resource usage',
                    'Tune thread pool configuration',
                    'Review algorithm efficiency',
                    'Implement load balancing'
                ]
            },
            'application': {
                'patterns': [
                    r'null.*pointer.*exception',
                    r'array.*index.*out.*of.*bounds',
                    r'class.*not.*found',
                    r'method.*not.*found',
                    r'illegal.*argument',
                    r'runtime.*exception',
                    r'assertion.*failed',
                    r'validation.*failed'
                ],
                'root_causes': [
                    'Null pointer dereference',
                    'Array bounds violation',
                    'Missing class or dependency',
                    'Method signature mismatch',
                    'Invalid input parameters',
                    'Runtime assertion failure',
                    'Data validation error'
                ],
                'recommendations': [
                    'Add null checks and defensive programming',
                    'Validate array bounds before access',
                    'Verify classpath and dependencies',
                    'Check method signatures and versions',
                    'Implement input validation',
                    'Review business logic assertions',
                    'Strengthen data validation rules'
                ]
            }
        }
    
    def analyze(self, log: Dict[str, Any], processed_log: Dict[str, Any]) -> Dict[str, List[str]]:
        """分析异常日志并生成根因和建议"""
        try:
            message = log.get('message', '') or ''  # Handle None messages
            message = message.lower()
            level = log.get('level', '').upper()
            source = log.get('source', '')
            
            root_causes = []
            recommendations = []
            
            # Only analyze ERROR, FATAL, CRITICAL level logs or logs with error keywords
            features = processed_log.get('features', {})
            should_analyze = (
                level in ['ERROR', 'FATAL', 'CRITICAL'] or 
                features.get('has_error_keywords', False)
            )
            
            if not should_analyze:
                return {
                    'root_causes': [],
                    'recommendations': []
                }
            
            # Pattern-based analysis with scoring
            matched_categories = self._match_error_patterns_with_score(message)
            
            # Process matched categories by priority (highest score first)
            for category, score in matched_categories:
                category_data = self.error_patterns[category]
                # Add more causes/recommendations for higher scoring matches
                num_items = 3 if score > 0.7 else 2 if score > 0.5 else 1
                root_causes.extend(category_data['root_causes'][:num_items])
                recommendations.extend(category_data['recommendations'][:num_items])
            
            # Level-based analysis
            if level in ['ERROR', 'FATAL', 'CRITICAL']:
                if not root_causes:  # If no patterns matched
                    root_causes.extend(self._get_generic_error_causes(message))
                    recommendations.extend(self._get_generic_error_recommendations())
            
            # Feature-based analysis
            if features.get('has_error_keywords'):
                additional_causes = self._analyze_error_keywords(message)
                root_causes.extend(additional_causes)
            
            # Source-based analysis
            source_analysis = self._analyze_by_source(source, message)
            if source_analysis:
                root_causes.extend(source_analysis.get('causes', []))
                recommendations.extend(source_analysis.get('recommendations', []))
            
            # Context-aware analysis
            context_analysis = self._analyze_context(log, processed_log)
            if context_analysis:
                root_causes.extend(context_analysis.get('causes', []))
                recommendations.extend(context_analysis.get('recommendations', []))
            
            # Remove duplicates while preserving order
            root_causes = list(dict.fromkeys(root_causes))
            recommendations = list(dict.fromkeys(recommendations))
            
            # Limit results to most relevant
            root_causes = root_causes[:4]  # Increased from 3
            recommendations = recommendations[:4]  # Increased from 3
            
            return {
                'root_causes': root_causes,
                'recommendations': recommendations
            }
            
        except Exception as e:
            logging.error(f"Error in root cause analysis: {str(e)}")
            return {
                'root_causes': ['Unknown error occurred'],
                'recommendations': ['Review log details and system status']
            }
    
    def _match_error_patterns(self, message: str) -> List[str]:
        """匹配错误模式"""
        matched_categories = []
        
        for category, data in self.error_patterns.items():
            for pattern in data['patterns']:
                if re.search(pattern, message, re.IGNORECASE):
                    matched_categories.append(category)
                    break  # Only match once per category
        
        return matched_categories
    
    def _match_error_patterns_with_score(self, message: str) -> List[tuple]:
        """匹配错误模式并返回评分"""
        matched_categories = []
        
        for category, data in self.error_patterns.items():
            max_score = 0
            for pattern in data['patterns']:
                match = re.search(pattern, message, re.IGNORECASE)
                if match:
                    # Calculate score based on match quality
                    match_length = len(match.group())
                    pattern_specificity = len(pattern) / 20.0  # Normalize pattern complexity
                    score = min(1.0, (match_length / len(message)) + pattern_specificity)
                    max_score = max(max_score, score)
            
            if max_score > 0:
                matched_categories.append((category, max_score))
        
        # Sort by score descending
        return sorted(matched_categories, key=lambda x: x[1], reverse=True)
    
    def _get_generic_error_causes(self, message: str) -> List[str]:
        """获取通用错误原因"""
        generic_causes = []
        
        if 'failed' in message:
            generic_causes.append('Operation failed to complete')
        if 'exception' in message:
            generic_causes.append('Unhandled exception occurred')
        if 'error' in message:
            generic_causes.append('System error detected')
        
        if not generic_causes:
            generic_causes.append('Unexpected system behavior')
        
        return generic_causes
    
    def _get_generic_error_recommendations(self) -> List[str]:
        """获取通用错误建议"""
        return [
            'Check system logs for additional context',
            'Verify system resources and dependencies',
            'Review recent changes and deployments'
        ]
    
    def _analyze_error_keywords(self, message: str) -> List[str]:
        """基于错误关键词分析"""
        additional_causes = []
        
        error_keyword_map = {
            'timeout': 'Request timeout occurred',
            'refused': 'Connection refused by target',
            'denied': 'Access or permission denied',
            'crash': 'Application crash detected',
            'panic': 'System panic condition',
            'abort': 'Operation aborted unexpectedly'
        }
        
        for keyword, cause in error_keyword_map.items():
            if keyword in message:
                additional_causes.append(cause)
        
        return additional_causes
    
    def _analyze_by_source(self, source: str, message: str) -> Dict[str, List[str]]:
        """基于服务来源分析"""
        source_patterns = {
            'auth': {
                'keywords': ['auth', 'login', 'user', 'session'],
                'causes': ['Authentication service issue', 'User session problem'],
                'recommendations': ['Check authentication service health', 'Verify user session management']
            },
            'database': {
                'keywords': ['db', 'sql', 'postgres', 'mysql', 'mongo'],
                'causes': ['Database service issue', 'Data access problem'],
                'recommendations': ['Monitor database performance', 'Check database connections']
            },
            'payment': {
                'keywords': ['payment', 'billing', 'transaction', 'stripe'],
                'causes': ['Payment processing issue', 'Transaction failure'],
                'recommendations': ['Check payment gateway status', 'Verify transaction logs']
            },
            'api': {
                'keywords': ['api', 'gateway', 'proxy', 'endpoint'],
                'causes': ['API service issue', 'Gateway problem'],
                'recommendations': ['Check API gateway health', 'Verify endpoint availability']
            }
        }
        
        source_lower = source.lower()
        message_lower = message.lower()
        
        for service_type, config in source_patterns.items():
            if any(keyword in source_lower for keyword in config['keywords']) or \
               any(keyword in message_lower for keyword in config['keywords']):
                return {
                    'causes': config['causes'],
                    'recommendations': config['recommendations']
                }
        
        return {}
    
    def _analyze_context(self, log: Dict[str, Any], processed_log: Dict[str, Any]) -> Dict[str, List[str]]:
        """基于上下文分析"""
        features = processed_log.get('features', {})
        message = log.get('message', '').lower()
        
        context_analysis = {
            'causes': [],
            'recommendations': []
        }
        
        # Message length analysis
        message_length = features.get('message_length', 0)
        if message_length > 1000:
            context_analysis['causes'].append('Verbose error message indicates complex issue')
            context_analysis['recommendations'].append('Review detailed error context')
        elif message_length < 10:
            context_analysis['causes'].append('Minimal error information available')
            context_analysis['recommendations'].append('Enable more detailed logging')
        
        # Special character analysis
        if features.get('has_special_chars'):
            if any(char in message for char in ['[', ']', '{', '}']):
                context_analysis['causes'].append('Structured data parsing issue')
                context_analysis['recommendations'].append('Verify data format and parsing logic')
        
        # Numeric pattern analysis
        if features.get('has_numbers'):
            if re.search(r'\b(404|500|503|502|401|403)\b', message):
                context_analysis['causes'].append('HTTP status code error')
                context_analysis['recommendations'].append('Check HTTP service and routing')
            elif re.search(r'\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b', message):
                context_analysis['causes'].append('Network address related issue')
                context_analysis['recommendations'].append('Verify network configuration')
        
        return context_analysis if context_analysis['causes'] else {}