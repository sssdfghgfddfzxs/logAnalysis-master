package alert

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// EmailNotifier implements the Notifier interface for email notifications
type EmailNotifier struct {
	config EmailConfig
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		config: config,
	}
}

// SendAlert sends an alert via email
func (e *EmailNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	// Build email content
	subject := fmt.Sprintf("[告警] %s - %s", alert.RuleName, alert.Source)
	body := e.buildEmailBody(alert)

	// Create message
	message := fmt.Sprintf("From: %s\r\n", e.config.From)
	message += fmt.Sprintf("To: %s\r\n", strings.Join(e.config.To, ","))
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// Setup authentication
	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPHost)

	// Setup TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         e.config.SMTPHost,
	}

	// Connect to server
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	if e.config.UseTLS {
		// Direct TLS connection
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, e.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}

		return e.sendMessage(client, message)
	} else {
		// Plain connection with optional STARTTLS
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer client.Quit()

		if e.config.UseStartTLS {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}

		return e.sendMessage(client, message)
	}
}

// sendMessage sends the email message using the SMTP client
func (e *EmailNotifier) sendMessage(client *smtp.Client, message string) error {
	// Set sender
	if err := client.Mail(e.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, to := range e.config.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	if _, err := writer.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// buildEmailBody builds the HTML email body
func (e *EmailNotifier) buildEmailBody(alert *Alert) string {
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .alert-container { border: 1px solid #ddd; border-radius: 5px; padding: 20px; }
        .alert-header { background-color: #f44336; color: white; padding: 10px; margin: -20px -20px 20px -20px; border-radius: 5px 5px 0 0; }
        .alert-info { margin-bottom: 15px; }
        .alert-label { font-weight: bold; color: #333; }
        .alert-value { margin-left: 10px; }
        .anomaly-score { color: #f44336; font-weight: bold; }
        .root-causes, .recommendations { background-color: #f9f9f9; padding: 10px; border-radius: 3px; margin-top: 10px; }
        .list-item { margin: 5px 0; }
        .timestamp { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <div class="alert-container">
        <div class="alert-header">
            <h2>🚨 日志异常告警</h2>
        </div>
        
        <div class="alert-info">
            <span class="alert-label">告警规则:</span>
            <span class="alert-value">%s</span>
        </div>
        
        <div class="alert-info">
            <span class="alert-label">日志来源:</span>
            <span class="alert-value">%s</span>
        </div>
        
        <div class="alert-info">
            <span class="alert-label">日志级别:</span>
            <span class="alert-value">%s</span>
        </div>
        
        <div class="alert-info">
            <span class="alert-label">异常评分:</span>
            <span class="alert-value anomaly-score">%.2f</span>
        </div>
        
        <div class="alert-info">
            <span class="alert-label">日志消息:</span>
            <div class="alert-value">%s</div>
        </div>
        
        %s
        
        %s
        
        <div class="alert-info timestamp">
            <span class="alert-label">告警时间:</span>
            <span class="alert-value">%s</span>
        </div>
    </div>
</body>
</html>`

	// Build root causes section
	rootCausesHTML := ""
	if len(alert.RootCauses) > 0 {
		rootCausesHTML = `
        <div class="alert-info">
            <span class="alert-label">根本原因:</span>
            <div class="root-causes">`
		for _, cause := range alert.RootCauses {
			rootCausesHTML += fmt.Sprintf(`<div class="list-item">• %s</div>`, cause)
		}
		rootCausesHTML += `</div></div>`
	}

	// Build recommendations section
	recommendationsHTML := ""
	if len(alert.Recommendations) > 0 {
		recommendationsHTML = `
        <div class="alert-info">
            <span class="alert-label">建议措施:</span>
            <div class="recommendations">`
		for _, rec := range alert.Recommendations {
			recommendationsHTML += fmt.Sprintf(`<div class="list-item">• %s</div>`, rec)
		}
		recommendationsHTML += `</div></div>`
	}

	return fmt.Sprintf(html,
		alert.RuleName,
		alert.Source,
		alert.Level,
		alert.AnomalyScore,
		alert.Message,
		rootCausesHTML,
		recommendationsHTML,
		alert.Timestamp.Format("2006-01-02 15:04:05"),
	)
}
