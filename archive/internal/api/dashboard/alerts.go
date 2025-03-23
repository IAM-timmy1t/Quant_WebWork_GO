package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"text/template"
	"time"
)

// AlertManager handles alert generation and notification
type AlertManager struct {
	mu sync.RWMutex

	// Alert configuration
	config AlertConfig

	// Active alerts
	activeAlerts map[string]*Alert

	// Alert history
	alertHistory []Alert

	// Alert channels
	notifiers []AlertNotifier

	// Templates
	templates map[string]*template.Template
}

// AlertConfig defines alert configuration
type AlertConfig struct {
	Enabled           bool          `json:"enabled"`
	RetentionPeriod   time.Duration `json:"retentionPeriod"`
	MinInterval       time.Duration `json:"minInterval"`
	DefaultPriority   string        `json:"defaultPriority"`
	WebhookURL        string        `json:"webhookUrl,omitempty"`
	EmailConfig       EmailConfig   `json:"emailConfig,omitempty"`
	SlackConfig       SlackConfig   `json:"slackConfig,omitempty"`
}

// EmailConfig defines email notification settings
type EmailConfig struct {
	Enabled    bool   `json:"enabled"`
	SMTPServer string `json:"smtpServer"`
	SMTPPort   int    `json:"smtpPort"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"fromEmail"`
	ToEmails   []string `json:"toEmails"`
}

// SlackConfig defines Slack notification settings
type SlackConfig struct {
	Enabled    bool   `json:"enabled"`
	WebhookURL string `json:"webhookUrl"`
	Channel    string `json:"channel"`
}

// Alert represents a system alert
type Alert struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Priority    string    `json:"priority"`
	Message     string    `json:"message"`
	Details     string    `json:"details"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
	ResolvedAt  time.Time `json:"resolvedAt,omitempty"`
	Acknowledged bool      `json:"acknowledged"`
}

// AlertNotifier interface for different notification methods
type AlertNotifier interface {
	Notify(alert Alert) error
}

// NewAlertManager creates a new alert manager
func NewAlertManager(config AlertConfig) *AlertManager {
	am := &AlertManager{
		config:       config,
		activeAlerts: make(map[string]*Alert),
		alertHistory: make([]Alert, 0),
		notifiers:    make([]AlertNotifier, 0),
		templates:    make(map[string]*template.Template),
	}

	// Initialize notifiers
	if config.EmailConfig.Enabled {
		am.notifiers = append(am.notifiers, NewEmailNotifier(config.EmailConfig))
	}
	if config.SlackConfig.Enabled {
		am.notifiers = append(am.notifiers, NewSlackNotifier(config.SlackConfig))
	}
	if config.WebhookURL != "" {
		am.notifiers = append(am.notifiers, NewWebhookNotifier(config.WebhookURL))
	}

	// Initialize templates
	am.initializeTemplates()

	return am
}

// CreateAlert creates a new alert
func (am *AlertManager) CreateAlert(source, priority, message, details string) *Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert := &Alert{
		ID:        fmt.Sprintf("%s-%d", source, time.Now().UnixNano()),
		Source:    source,
		Priority:  priority,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
		Status:    "active",
	}

	// Check for duplicate active alerts
	for _, existing := range am.activeAlerts {
		if existing.Source == source && existing.Message == message && 
		   time.Since(existing.Timestamp) < am.config.MinInterval {
			return existing
		}
	}

	am.activeAlerts[alert.ID] = alert
	am.alertHistory = append(am.alertHistory, *alert)

	// Send notifications
	go am.notify(*alert)

	return alert
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Status = "resolved"
	alert.ResolvedAt = time.Now()
	delete(am.activeAlerts, alertID)

	return nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.activeAlerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Acknowledged = true
	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]Alert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, *alert)
	}
	return alerts
}

// GetAlertHistory returns historical alerts within the retention period
func (am *AlertManager) GetAlertHistory() []Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	cutoff := time.Now().Add(-am.config.RetentionPeriod)
	var alerts []Alert
	for _, alert := range am.alertHistory {
		if alert.Timestamp.After(cutoff) {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// notify sends alert notifications through configured channels
func (am *AlertManager) notify(alert Alert) {
	for _, notifier := range am.notifiers {
		go func(n AlertNotifier) {
			if err := n.Notify(alert); err != nil {
				// Log error
			}
		}(notifier)
	}
}

// EmailNotifier implements email notifications
type EmailNotifier struct {
	config EmailConfig
}

func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{config: config}
}

func (n *EmailNotifier) Notify(alert Alert) error {
	// Implement email sending logic
	return nil
}

// SlackNotifier implements Slack notifications
type SlackNotifier struct {
	config SlackConfig
}

func NewSlackNotifier(config SlackConfig) *SlackNotifier {
	return &SlackNotifier{config: config}
}

func (n *SlackNotifier) Notify(alert Alert) error {
	payload := map[string]interface{}{
		"channel": n.config.Channel,
		"text":    fmt.Sprintf("[%s] %s: %s", alert.Priority, alert.Source, alert.Message),
		"attachments": []map[string]interface{}{
			{
				"color": getAlertColor(alert.Priority),
				"fields": []map[string]string{
					{
						"title": "Details",
						"value": alert.Details,
					},
					{
						"title": "Time",
						"value": alert.Timestamp.Format(time.RFC3339),
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(n.config.WebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed: %d", resp.StatusCode)
	}

	return nil
}

// WebhookNotifier implements generic webhook notifications
type WebhookNotifier struct {
	webhookURL string
}

func NewWebhookNotifier(webhookURL string) *WebhookNotifier {
	return &WebhookNotifier{webhookURL: webhookURL}
}

func (n *WebhookNotifier) Notify(alert Alert) error {
	jsonPayload, err := json.Marshal(alert)
	if err != nil {
		return err
	}

	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook notification failed: %d", resp.StatusCode)
	}

	return nil
}

// Helper functions

func getAlertColor(priority string) string {
	switch priority {
	case "critical":
		return "#FF0000"
	case "high":
		return "#FFA500"
	case "medium":
		return "#FFFF00"
	case "low":
		return "#00FF00"
	default:
		return "#808080"
	}
}

func (am *AlertManager) initializeTemplates() {
	// Initialize alert templates
	emailTemplate := `
Subject: [{{.Priority}}] Alert from {{.Source}}

Alert Details:
Priority: {{.Priority}}
Source: {{.Source}}
Message: {{.Message}}
Details: {{.Details}}
Time: {{.Timestamp}}
Status: {{.Status}}
`

	slackTemplate := `
*[{{.Priority}}] Alert from {{.Source}}*
>Message: {{.Message}}
>Details: {{.Details}}
>Time: {{.Timestamp}}
>Status: {{.Status}}
`

	am.templates["email"] = template.Must(template.New("email").Parse(emailTemplate))
	am.templates["slack"] = template.Must(template.New("slack").Parse(slackTemplate))
}
