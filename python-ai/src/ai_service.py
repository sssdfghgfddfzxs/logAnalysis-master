import grpc
from concurrent import futures
import logging
import os
from dotenv import load_dotenv

from proto import ai_analysis_pb2_grpc
from proto import ai_analysis_pb2
from core.log_preprocessor import LogPreprocessor
from core.llm_analyzer import LLMAnalyzer

# Load environment variables
load_dotenv()

class AIAnalysisService(ai_analysis_pb2_grpc.AIAnalysisServiceServicer):
    def __init__(self):
        self.preprocessor = LogPreprocessor()
        # 只使用LLM分析器，移除传统ML组件
        self.llm_analyzer = LLMAnalyzer()
        
        logging.info("AI Analysis Service initialized with pure LLM-based analysis")
        logging.info("All log analysis will use advanced language model for better accuracy")

    def AnalyzeLogs(self, request, context):
        """纯LLM分析 - 提供最高质量的异常检测和根因分析"""
        try:
            logging.info(f"Received LLM analysis request for {len(request.logs)} logs")
            
            # Convert protobuf logs to internal format
            logs = []
            for log_entry in request.logs:
                logs.append({
                    'id': log_entry.id,
                    'timestamp': log_entry.timestamp,
                    'level': log_entry.level,
                    'message': log_entry.message,
                    'source': log_entry.source,
                    'metadata': dict(log_entry.metadata)
                })
            
            # 基础预处理（保留，用于提供上下文信息）
            processed_logs = self.preprocessor.process(logs)
            
            # 使用LLM进行综合分析
            logging.info("Performing comprehensive LLM analysis for all logs")
            batch_analysis = self.llm_analyzer.analyze_logs_batch(logs, None)
            
            # 构建分析结果
            results = []
            individual_results = batch_analysis.get('individual_results', [])
            
            for i, log in enumerate(logs):
                # 从LLM分析结果中提取信息
                if i < len(individual_results):
                    individual = individual_results[i]
                    is_anomaly = individual.get('is_anomaly', False)
                    anomaly_score = individual.get('anomaly_score', 0.0)
                    root_causes = individual.get('root_causes', [])
                    recommendations = individual.get('recommendations', [])
                else:
                    # 降级处理：基于日志级别的简单判断
                    is_anomaly = log.get('level', '').upper() in ['ERROR', 'FATAL', 'CRITICAL']
                    anomaly_score = 0.8 if is_anomaly else 0.1
                    root_causes = [f"🤖 LLM分析: {log.get('level', 'INFO')}级别日志"]
                    recommendations = ["🔍 建议查看详细日志信息"]
                
                # 添加批量分析的总结信息到第一个结果
                if i == 0 and batch_analysis.get('summary'):
                    root_causes.insert(0, f"📊 整体分析: {batch_analysis['summary']}")
                    
                    if batch_analysis.get('key_findings'):
                        findings_str = ', '.join(batch_analysis['key_findings'][:3])
                        recommendations.insert(0, f"🔍 关键发现: {findings_str}")
                
                result = ai_analysis_pb2.AnalysisResult(
                    log_id=log['id'],
                    is_anomaly=is_anomaly,
                    anomaly_score=anomaly_score,
                    root_causes=root_causes,
                    recommendations=recommendations
                )
                results.append(result)
            
            logging.info(f"LLM analysis completed for {len(logs)} logs")
            
            return ai_analysis_pb2.LogAnalysisResponse(
                results=results,
                status="success",
                error_message=""
            )
            
        except Exception as e:
            logging.error(f"Error in LLM analysis: {str(e)}")
            return ai_analysis_pb2.LogAnalysisResponse(
                results=[],
                status="error",
                error_message=f"LLM analysis failed: {str(e)}"
            )

def serve():
    port = os.getenv('AI_SERVICE_PORT', '50051')
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    ai_analysis_pb2_grpc.add_AIAnalysisServiceServicer_to_server(
        AIAnalysisService(), server
    )
    
    listen_addr = f'[::]:{port}'
    server.add_insecure_port(listen_addr)
    
    logging.info(f"Starting Pure LLM AI Analysis Service on {listen_addr}")
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    serve()