# 告警邮件和钉钉通知配置指南

本指南将帮助你配置智能日志异常分析系统的告警通知功能，包括邮件通知和钉钉通知。

## 功能概述

系统支持以下告警通知方式：
- 📧 **邮件通知**：通过SMTP发送HTML格式的告警邮件
- 💬 **钉钉通知**：通过钉钉机器人发送Markdown格式的告警消息

## 1. 邮件通知配置

### 1.1 环境变量配置

在 `.env` 文件中添加以下邮件配置：

```bash
# Email Notification Configuration
EMAIL_SMTP_HOST=smtp.gmail.com          # SMTP服务器地址
EMAIL_SMTP_PORT=587                     # SMTP端口
EMAIL_USERNAME=your_email@gmail.com     # 邮箱用户名
EMAIL_PASSWORD=your_app_password        # 邮箱密码或应用专用密码
EMAIL_FROM=your_email@gmail.com         # 发件人邮箱
EMAIL_TO=admin@company.com,ops@company.com  # 收件人邮箱（多个用逗号分隔）
EMAIL_USE_TLS=false                     # 是否使用直接TLS连接
EMAIL_USE_STARTTLS=true                 # 是否使用STARTTLS
```

### 1.2 常见邮箱服务器配置

#### Gmail
```bash
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_USE_TLS=false
EMAIL_USE_STARTTLS=true
```

#### 163邮箱
```bash
EMAIL_SMTP_HOST=smtp.163.com
EMAIL_SMTP_PORT=25
EMAIL_USE_TLS=false
EMAIL_USE_STARTTLS=true
```

#### QQ邮箱
```bash
EMAIL_SMTP_HOST=smtp.qq.com
EMAIL_SMTP_PORT=587
EMAIL_USE_TLS=false
EMAIL_USE_STARTTLS=true
```

#### 企业邮箱（腾讯企业邮）
```bash
EMAIL_SMTP_HOST=smtp.exmail.qq.com
EMAIL_SMTP_PORT=587
EMAIL_USE_TLS=false
EMAIL_USE_STARTTLS=true
```

### 1.3 Gmail应用专用密码设置

如果使用Gmail，需要设置应用专用密码：

1. 登录Google账户
2. 进入"安全性"设置
3. 启用"两步验证"
4. 生成"应用专用密码"
5. 将生成的密码用作 `EMAIL_PASSWORD`

## 2. 钉钉通知配置

### 2.1 创建钉钉机器人

1. 在钉钉群聊中，点击右上角设置图标
2. 选择"智能群助手" → "添加机器人"
3. 选择"自定义"机器人
4. 设置机器人名称和头像
5. 选择安全设置（推荐使用"加签"方式）
6. 复制Webhook地址和密钥

### 2.2 环境变量配置

```bash
# DingTalk Notification Configuration
DINGTALK_WEBHOOK_URL=https://oapi.dingtalk.com/robot/send?access_token=your_access_token
DINGTALK_SECRET=your_webhook_secret  # 如果使用加签验证，填入密钥；否则留空
```

### 2.3 安全设置说明

钉钉机器人支持三种安全设置：

1. **自定义关键词**：消息中必须包含指定关键词
2. **加签**：使用密钥对请求进行签名验证（推荐）
3. **IP地址**：限制特定IP地址访问

推荐使用"加签"方式，安全性更高。

## 3. 告警规则配置

### 3.1 通过Web界面配置

1. 访问系统Web界面
2. 点击导航栏的"告警规则"
3. 点击"创建告警规则"按钮
4. 填写规则信息：
   - **规则名称**：给规则起一个描述性的名称
   - **异常评分阈值**：设置触发告警的异常评分阈值（0-1）
   - **日志级别**：选择要监控的日志级别
   - **日志来源**：指定要监控的日志来源
   - **时间窗口**：设置统计时间窗口
   - **最小异常数量**：在时间窗口内的最小异常数量
   - **通知渠道**：选择邮件和/或钉钉通知

### 3.2 告警条件示例

#### 高错误率告警
```json
{
  "anomaly_score_threshold": 0.8,
  "levels": ["ERROR"],
  "time_window_minutes": 15,
  "min_anomaly_count": 3
}
```

#### 系统关键错误告警
```json
{
  "anomaly_score_threshold": 0.9,
  "levels": ["ERROR", "FATAL"],
  "sources": ["system", "database"],
  "time_window_minutes": 5,
  "min_anomaly_count": 1
}
```

#### 服务异常告警
```json
{
  "anomaly_score_threshold": 0.7,
  "sources": ["user-service", "payment-service"],
  "time_window_minutes": 30,
  "min_anomaly_count": 5
}
```

## 4. 通知消息格式

### 4.1 邮件通知格式

邮件采用HTML格式，包含以下信息：
- 告警规则名称
- 日志来源和级别
- 异常评分
- 日志消息内容
- 根本原因分析
- 建议措施
- 告警时间

### 4.2 钉钉通知格式

钉钉消息采用Markdown格式，包含相同的告警信息，格式更适合移动端查看。

## 5. 测试配置

### 5.1 测试邮件配置

```bash
# 使用curl测试邮件配置
curl -X POST http://localhost:8080/api/v1/alert-rules/test-email \
  -H "Content-Type: application/json" \
  -d '{
    "to": ["test@example.com"],
    "subject": "测试邮件",
    "message": "这是一条测试邮件"
  }'
```

### 5.2 测试钉钉配置

```bash
# 使用curl测试钉钉配置
curl -X POST http://localhost:8080/api/v1/alert-rules/test-dingtalk \
  -H "Content-Type: application/json" \
  -d '{
    "message": "这是一条测试消息"
  }'
```

### 5.3 通过Web界面测试

1. 在告警规则列表中，点击规则的"测试"按钮
2. 系统会发送测试告警到配置的通知渠道
3. 检查邮箱和钉钉群是否收到测试消息

## 6. 故障排除

### 6.1 邮件发送失败

**常见问题：**
- SMTP服务器地址或端口错误
- 用户名或密码错误
- 未启用SMTP服务
- 防火墙阻止SMTP连接

**解决方法：**
1. 检查SMTP配置是否正确
2. 确认邮箱已启用SMTP服务
3. 使用应用专用密码（Gmail等）
4. 检查网络连接和防火墙设置

### 6.2 钉钉消息发送失败

**常见问题：**
- Webhook URL错误
- 安全验证失败
- 消息格式不正确
- 机器人被禁用

**解决方法：**
1. 检查Webhook URL是否正确
2. 确认安全设置配置正确
3. 检查消息是否包含必要的关键词
4. 确认机器人未被群管理员禁用

### 6.3 查看日志

系统日志会记录通知发送的详细信息：

```bash
# 查看Go后端日志
docker-compose logs go-backend | grep -i "alert\|notification"

# 查看实时日志
docker-compose logs -f go-backend
```

## 7. 高级配置

### 7.1 自定义邮件模板

可以修改 `go-backend/internal/alert/email_notifier.go` 中的 `buildEmailBody` 方法来自定义邮件模板。

### 7.2 添加更多通知渠道

系统架构支持扩展更多通知渠道：
1. 实现 `Notifier` 接口
2. 在 `NotificationManager` 中注册新的通知器
3. 更新前端界面添加新的通知渠道选项

### 7.3 告警抑制

系统内置告警抑制机制，防止短时间内重复发送相同告警。默认抑制时间为5分钟，可在 `alert/engine.go` 中调整。

## 8. 安全建议

1. **邮箱安全**：
   - 使用应用专用密码而非主密码
   - 定期更换密码
   - 启用两步验证

2. **钉钉安全**：
   - 使用加签验证
   - 定期更换Webhook密钥
   - 限制机器人权限

3. **网络安全**：
   - 使用TLS/SSL加密连接
   - 配置防火墙规则
   - 监控异常访问

## 9. 监控和维护

### 9.1 监控告警发送状态

系统会记录每次告警发送的结果，可以通过以下方式监控：

```bash
# 查看告警发送统计
curl http://localhost:8080/api/v1/alert-stats

# 查看失败的告警
curl http://localhost:8080/api/v1/alert-failures
```

### 9.2 定期维护

- 定期检查邮箱和钉钉配置是否有效
- 清理过期的告警记录
- 更新告警规则以适应业务变化
- 备份告警规则配置

通过以上配置，你就可以成功设置智能日志异常分析系统的告警通知功能了。如果遇到问题，请参考故障排除部分或查看系统日志获取更多信息。