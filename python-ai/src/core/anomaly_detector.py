import numpy as np
from sklearn.ensemble import IsolationForest
from sklearn.preprocessing import StandardScaler
from typing import List, Dict, Any
import logging
import joblib
import os

class AnomalyDetector:
    """改进的异常检测器 - 混合机器学习和规则引擎"""
    
    def __init__(self, contamination=0.05, model_path=None):  # 降低阈值
        self.contamination = contamination
        self.model = IsolationForest(
            contamination=contamination,
            random_state=42,
            n_estimators=100
        )
        self.scaler = StandardScaler()
        self.is_fitted = False
        self.model_path = model_path or os.path.join(os.path.dirname(__file__), '../../models')
        
        # Create models directory if it doesn't exist
        os.makedirs(self.model_path, exist_ok=True)
        
        # Try to load existing model
        self._load_model()
    
    def detect_anomalies(self, processed_logs: List[Dict[str, Any]]) -> Dict[str, List]:
        """检测异常日志 - 混合策略"""
        if not processed_logs:
            return {'is_anomaly': [], 'scores': []}
        
        try:
            # 1. 基于规则的异常检测（优先级最高）
            rule_based_results = self._rule_based_detection(processed_logs)
            
            # 2. 机器学习异常检测
            ml_results = self._ml_based_detection(processed_logs)
            
            # 3. 混合结果
            combined_results = self._combine_results(rule_based_results, ml_results)
            
            return combined_results
            
        except Exception as e:
            logging.error(f"Error in anomaly detection: {str(e)}")
            return self._get_default_results(len(processed_logs))
    
    def _rule_based_detection(self, processed_logs: List[Dict[str, Any]]) -> Dict[str, List]:
        """基于规则的异常检测"""
        is_anomaly = []
        scores = []
        
        for log in processed_logs:
            features = log.get('features', {})
            original_log = log  # 原始日志数据
            
            level = original_log.get('level', '').upper()
            message = original_log.get('message', '').lower()
            
            # 规则1: ERROR/FATAL/CRITICAL级别自动标记为异常
            if level in ['ERROR', 'FATAL', 'CRITICAL']:
                is_anomaly.append(True)
                scores.append(0.8)  # 高异常分数
                continue
            
            # 规则2: 包含严重错误关键词
            critical_keywords = [
                'panic', 'crash', 'segmentation fault', 'out of memory',
                'connection refused', 'timeout', 'failed', 'exception',
                'critical', 'fatal', 'abort', 'denied'
            ]
            
            critical_score = 0
            for keyword in critical_keywords:
                if keyword in message:
                    critical_score += 0.2
            
            if critical_score >= 0.4:  # 包含2个或更多关键词
                is_anomaly.append(True)
                scores.append(min(0.9, 0.5 + critical_score))
                continue
            
            # 规则3: WARN级别 + 错误关键词
            if level == 'WARN' and features.get('has_error_keywords', False):
                is_anomaly.append(True)
                scores.append(0.6)
                continue
            
            # 规则4: 异常长的消息（可能是堆栈跟踪）
            if features.get('message_length', 0) > 1000:
                is_anomaly.append(True)
                scores.append(0.5)
                continue
            
            # 规则5: 频繁出现的错误模式
            if self._is_frequent_error_pattern(message):
                is_anomaly.append(True)
                scores.append(0.7)
                continue
            
            # 默认为正常
            is_anomaly.append(False)
            scores.append(0.1)
        
        return {'is_anomaly': is_anomaly, 'scores': scores}
    
    def _ml_based_detection(self, processed_logs: List[Dict[str, Any]]) -> Dict[str, List]:
        """基于机器学习的异常检测"""
        try:
            # Extract enhanced features for ML model
            features = self._extract_enhanced_ml_features(processed_logs)
            
            if features.size == 0:
                logging.warning("No features extracted for ML detection")
                return self._get_default_results(len(processed_logs))
            
            # If model is not fitted, fit it with current data
            if not self.is_fitted:
                self._fit_model(features)
            
            # Predict anomalies
            predictions = self.model.predict(features)
            scores = self.model.score_samples(features)
            
            # Convert predictions (-1 for anomaly, 1 for normal) to boolean
            is_anomaly = [pred == -1 for pred in predictions]
            
            # Normalize scores to [0, 1] range (higher = more anomalous)
            normalized_scores = self._normalize_scores(scores)
            
            return {
                'is_anomaly': is_anomaly,
                'scores': normalized_scores.tolist()
            }
            
        except Exception as e:
            logging.error(f"Error in ML-based detection: {str(e)}")
            return self._get_default_results(len(processed_logs))
    
    def _combine_results(self, rule_results: Dict[str, List], ml_results: Dict[str, List]) -> Dict[str, List]:
        """组合规则和机器学习的结果"""
        combined_is_anomaly = []
        combined_scores = []
        
        for i in range(len(rule_results['is_anomaly'])):
            rule_anomaly = rule_results['is_anomaly'][i]
            rule_score = rule_results['scores'][i]
            
            ml_anomaly = ml_results['is_anomaly'][i] if i < len(ml_results['is_anomaly']) else False
            ml_score = ml_results['scores'][i] if i < len(ml_results['scores']) else 0.0
            
            # 规则检测优先级更高
            if rule_anomaly:
                combined_is_anomaly.append(True)
                combined_scores.append(max(rule_score, ml_score))
            elif ml_anomaly and ml_score > 0.7:  # 只有高置信度的ML结果才采用
                combined_is_anomaly.append(True)
                combined_scores.append(ml_score)
            else:
                combined_is_anomaly.append(False)
                combined_scores.append(min(rule_score, ml_score))
        
        return {
            'is_anomaly': combined_is_anomaly,
            'scores': combined_scores
        }
    
    def _extract_enhanced_ml_features(self, processed_logs: List[Dict[str, Any]]) -> np.ndarray:
        """提取增强的机器学习特征"""
        features_list = []
        
        for log in processed_logs:
            log_features = log.get('features', {})
            
            # 原有特征
            feature_vector = [
                log_features.get('message_length', 0),
                log_features.get('word_count', 0),
                log_features.get('level_numeric', 0),
                int(log_features.get('has_error_keywords', False)),
                int(log_features.get('has_warning_keywords', False)),
                int(log_features.get('has_numbers', False)),
                int(log_features.get('has_special_chars', False)),
                log_features.get('source_hash', 0),
            ]
            
            # 新增特征
            feature_vector.extend([
                log_features.get('critical_keyword_count', 0),
                log_features.get('error_pattern_score', 0),
                log_features.get('stack_trace_indicator', 0),
                int(log_features.get('is_error_level', False)),
                log_features.get('message_entropy', 0),
            ])
            
            features_list.append(feature_vector)
        
        if not features_list:
            return np.array([])
            
        features = np.array(features_list)
        
        # Handle any NaN or infinite values
        features = np.nan_to_num(features, nan=0.0, posinf=0.0, neginf=0.0)
        
        return features
    
    def _is_frequent_error_pattern(self, message: str) -> bool:
        """检查是否是频繁出现的错误模式"""
        error_patterns = [
            'connection refused',
            'timeout',
            'out of memory',
            'segmentation fault',
            'null pointer',
            'stack overflow',
            'access denied',
            'permission denied',
            'file not found',
            'network unreachable'
        ]
        
        message_lower = message.lower()
        return any(pattern in message_lower for pattern in error_patterns)
    
    def _fit_model(self, features: np.ndarray):
        """训练模型"""
        try:
            if features.size == 0:
                logging.warning("Cannot fit model with empty features")
                return
                
            # Scale features
            scaled_features = self.scaler.fit_transform(features)
            
            # Fit the isolation forest
            self.model.fit(scaled_features)
            self.is_fitted = True
            
            # Save the model
            self._save_model()
            
            logging.info(f"Enhanced model fitted with {len(features)} samples")
            
        except Exception as e:
            logging.error(f"Error fitting model: {str(e)}")
    
    def _normalize_scores(self, scores: np.ndarray) -> np.ndarray:
        """标准化异常分数到 [0, 1] 范围"""
        if len(scores) == 0:
            return np.array([])
            
        # Isolation Forest scores are typically negative
        # More negative = more anomalous
        min_score = np.min(scores)
        max_score = np.max(scores)
        
        if min_score == max_score:
            return np.zeros_like(scores)
        
        # Invert and normalize to [0, 1]
        normalized = (max_score - scores) / (max_score - min_score)
        
        return np.clip(normalized, 0, 1)
    
    def _get_default_results(self, num_logs: int) -> Dict[str, List]:
        """获取默认结果（当检测失败时）"""
        return {
            'is_anomaly': [False] * num_logs,
            'scores': [0.0] * num_logs
        }
    
    def _save_model(self):
        """保存模型"""
        try:
            model_file = os.path.join(self.model_path, 'enhanced_anomaly_model.joblib')
            scaler_file = os.path.join(self.model_path, 'enhanced_scaler.joblib')
            
            joblib.dump(self.model, model_file)
            joblib.dump(self.scaler, scaler_file)
            
            logging.info(f"Enhanced model saved to {model_file}")
            
        except Exception as e:
            logging.error(f"Error saving model: {str(e)}")
    
    def _load_model(self):
        """加载已保存的模型"""
        try:
            model_file = os.path.join(self.model_path, 'enhanced_anomaly_model.joblib')
            scaler_file = os.path.join(self.model_path, 'enhanced_scaler.joblib')
            
            if os.path.exists(model_file) and os.path.exists(scaler_file):
                self.model = joblib.load(model_file)
                self.scaler = joblib.load(scaler_file)
                self.is_fitted = True
                logging.info("Enhanced pre-trained model loaded successfully")
            else:
                logging.info("No enhanced pre-trained model found, will train on first batch")
                
        except Exception as e:
            logging.error(f"Error loading enhanced model: {str(e)}")
            self.is_fitted = False
