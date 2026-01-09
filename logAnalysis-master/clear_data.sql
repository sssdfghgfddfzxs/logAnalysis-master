-- 清空日志分析系统数据库脚本
-- 注意：此操作将删除所有数据，请谨慎使用

\echo '开始清空数据库...'

-- 清空分析结果表
TRUNCATE TABLE analysis_results CASCADE;
\echo '✓ 已清空 analysis_results 表'

-- 清空日志表
TRUNCATE TABLE logs CASCADE;
\echo '✓ 已清空 logs 表'

-- 清空告警规则表（如果需要保留规则，请注释掉下面这行）
-- TRUNCATE TABLE alert_rules CASCADE;
-- \echo '✓ 已清空 alert_rules 表'

-- 重置序列（如果有自增ID）
-- ALTER SEQUENCE logs_id_seq RESTART WITH 1;
-- ALTER SEQUENCE analysis_results_id_seq RESTART WITH 1;

-- 显示清空后的表统计
\echo '数据库清空完成，当前表统计：'
SELECT 
    'logs' as table_name, 
    COUNT(*) as record_count 
FROM logs
UNION ALL
SELECT 
    'analysis_results' as table_name, 
    COUNT(*) as record_count 
FROM analysis_results
UNION ALL
SELECT 
    'alert_rules' as table_name, 
    COUNT(*) as record_count 
FROM alert_rules;

\echo '✅ 数据库清空成功完成！'