package alert

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DingTalkNotifier implements the Notifier interface for DingTalk notifications
type DingTalkNotifier struct {
	config DingTalkConfig
	client *http.Client
}

// DingTalkMessage represents a DingTalk message
type DingTalkMessage struct {
	MsgType  string            `json:"msgtype"`
	Markdown *DingTalkMarkdown `json:"markdown,omitempty"`
	At       *DingTalkAt       `json:"at,omitempty"`
}

// DingTalkMarkdown represents markdown content for DingTalk
type DingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// DingTalkAt represents @mentions in DingTalk
type DingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

// DingTalkResponse represents DingTalk API response
type DingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// NewDingTalkNotifier creates a new DingTalk notifier
func NewDingTalkNotifier(config DingTalkConfig) *DingTalkNotifier {
	return &DingTalkNotifier{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendAlert sends an alert via DingTalk
func (d *DingTalkNotifier) SendAlert(ctx context.Context, alert *Alert) error {
	// Build webhook URL with signature if secret is provided
	webhookURL := d.config.WebhookURL
	if d.config.Secret != "" {
		timestamp := time.Now().UnixNano() / 1e6
		sign := d.generateSignature(timestamp, d.config.Secret)

		u, err := url.Parse(webhookURL)
		if err != nil {
			return fmt.Errorf("invalid webhook URL: %w", err)
		}

		query := u.Query()
		query.Set("timestamp", strconv.FormatInt(timestamp, 10))
		query.Set("sign", sign)
		u.RawQuery = query.Encode()
		webhookURL = u.String()
	}

	// Build message
	message := &DingTalkMessage{
		MsgType:  "markdown",
		Markdown: d.buildMarkdownContent(alert),
		At: &DingTalkAt{
			IsAtAll: false, // 可以根据需要配置
		},
	}

	// Marshal message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var dingResp DingTalkResponse
	if err := json.NewDecoder(resp.Body).Decode(&dingResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if dingResp.ErrCode != 0 {
		return fmt.Errorf("DingTalk API error: %s (code: %d)", dingResp.ErrMsg, dingResp.ErrCode)
	}

	return nil
}

// generateSignature generates HMAC-SHA256 signature for DingTalk webhook
func (d *DingTalkNotifier) generateSignature(timestamp int64, secret string) string {
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// buildMarkdownContent builds markdown content for DingTalk message
func (d *DingTalkNotifier) buildMarkdownContent(alert *Alert) *DingTalkMarkdown {
	title := fmt.Sprintf("🚨 日志异常告警 - %s", alert.RuleName)

	text := fmt.Sprintf(`## %s

**告警规则:** %s

**日志来源:** %s

**日志级别:** %s

**异常评分:** %.2f

**日志消息:**
> %s

`,
		title,
		alert.RuleName,
		alert.Source,
		alert.Level,
		alert.AnomalyScore,
		alert.Message,
	)

	// Add root causes if available
	if len(alert.RootCauses) > 0 {
		text += "**根本原因:**\n"
		for _, cause := range alert.RootCauses {
			text += fmt.Sprintf("- %s\n", cause)
		}
		text += "\n"
	}

	// Add recommendations if available
	if len(alert.Recommendations) > 0 {
		text += "**建议措施:**\n"
		for _, rec := range alert.Recommendations {
			text += fmt.Sprintf("- %s\n", rec)
		}
		text += "\n"
	}

	// Add timestamp
	text += fmt.Sprintf("**告警时间:** %s\n", alert.Timestamp.Format("2006-01-02 15:04:05"))

	return &DingTalkMarkdown{
		Title: title,
		Text:  text,
	}
}
