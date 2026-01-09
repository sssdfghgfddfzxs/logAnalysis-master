-- Initialize database for intelligent log analysis system

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create logs table
CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(100) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for logs table
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_source ON logs(source);
CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at);

-- Create analysis_results table
CREATE TABLE IF NOT EXISTS analysis_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    log_id UUID REFERENCES logs(id) ON DELETE CASCADE,
    is_anomaly BOOLEAN NOT NULL,
    anomaly_score DECIMAL(5,4),
    root_causes JSONB,
    recommendations JSONB,
    analyzed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for analysis_results table
CREATE INDEX IF NOT EXISTS idx_analysis_log_id ON analysis_results(log_id);
CREATE INDEX IF NOT EXISTS idx_analysis_anomaly ON analysis_results(is_anomaly);
CREATE INDEX IF NOT EXISTS idx_analysis_score ON analysis_results(anomaly_score);
CREATE INDEX IF NOT EXISTS idx_analysis_analyzed_at ON analysis_results(analyzed_at);

-- Create alert_rules table
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    condition JSONB NOT NULL,
    notification_channels TEXT[],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for alert_rules table
CREATE INDEX IF NOT EXISTS idx_alert_rules_active ON alert_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_alert_rules_created_at ON alert_rules(created_at);

-- Insert sample alert rules
INSERT INTO alert_rules (name, condition, notification_channels, is_active) VALUES
('High Error Rate', '{"anomaly_score": {"gt": 0.8}, "level": "ERROR"}', ARRAY['email', 'dingtalk'], true),
('Critical System Errors', '{"level": "FATAL", "source": {"in": ["system", "database"]}}', ARRAY['email', 'dingtalk'], true)
ON CONFLICT DO NOTHING;

-- Create a view for anomaly statistics
CREATE OR REPLACE VIEW anomaly_stats AS
SELECT 
    DATE_TRUNC('hour', l.timestamp) as hour,
    COUNT(*) as total_logs,
    COUNT(CASE WHEN ar.is_anomaly THEN 1 END) as anomaly_count,
    ROUND(
        COUNT(CASE WHEN ar.is_anomaly THEN 1 END)::DECIMAL / 
        NULLIF(COUNT(*), 0) * 100, 2
    ) as anomaly_rate
FROM logs l
LEFT JOIN analysis_results ar ON l.id = ar.log_id
WHERE l.timestamp >= NOW() - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', l.timestamp)
ORDER BY hour DESC;

-- Grant permissions (if needed for specific user)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO log_analysis_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO log_analysis_user;