package notify

import (
	"context"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

// EmailChannel sends notifications via SMTP.
type EmailChannel struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
	Name     string
}

// NewEmailChannel creates an email notification channel.
func NewEmailChannel(name, host string, port int, username, password, from string, to []string) *EmailChannel {
	return &EmailChannel{
		SMTPHost: host,
		SMTPPort: port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
		Name:     name,
	}
}

func (e *EmailChannel) Type() string { return "email" }

// Send delivers a notification via SMTP email.
func (e *EmailChannel) Send(ctx context.Context, n Notification) error {
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(n.Severity), n.Title)
	body := e.buildHTML(n)

	msg := "From: " + e.From + "\r\n" +
		"To: " + strings.Join(e.To, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		body

	addr := e.SMTPHost + ":" + strconv.Itoa(e.SMTPPort)

	var auth smtp.Auth
	if e.Username != "" {
		auth = smtp.PlainAuth("", e.Username, e.Password, e.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, e.From, e.To, []byte(msg)); err != nil {
		return fmt.Errorf("email: send: %w", err)
	}
	return nil
}

func (e *EmailChannel) buildHTML(n Notification) string {
	color := severityColorHex(n.Severity)

	var fieldsHTML string
	if n.Service != "" {
		fieldsHTML += "<tr><td style=\"padding:4px 8px;font-weight:bold;\">Service</td><td style=\"padding:4px 8px;\">" + htmlEscape(n.Service) + "</td></tr>"
	}
	if n.Type != "" {
		fieldsHTML += "<tr><td style=\"padding:4px 8px;font-weight:bold;\">Type</td><td style=\"padding:4px 8px;\">" + htmlEscape(n.Type) + "</td></tr>"
	}
	fieldsHTML += "<tr><td style=\"padding:4px 8px;font-weight:bold;\">Severity</td><td style=\"padding:4px 8px;\">" + htmlEscape(n.Severity) + "</td></tr>"
	fieldsHTML += "<tr><td style=\"padding:4px 8px;font-weight:bold;\">Time</td><td style=\"padding:4px 8px;\">" + htmlEscape(n.Timestamp.UTC().String()) + "</td></tr>"

	for k, v := range n.Fields {
		fieldsHTML += "<tr><td style=\"padding:4px 8px;font-weight:bold;\">" + htmlEscape(k) + "</td><td style=\"padding:4px 8px;\">" + htmlEscape(v) + "</td></tr>"
	}

	var actionsHTML string
	if n.URL != "" {
		actionsHTML += "<a href=\"" + n.URL + "\" style=\"display:inline-block;padding:8px 16px;background:" + color + ";color:#fff;text-decoration:none;border-radius:4px;margin-right:8px;\">View in DevTools</a>"
	}
	for _, a := range n.Actions {
		btnColor := color
		if a.Style == "danger" {
			btnColor = "#E01E5A"
		}
		actionsHTML += "<a href=\"" + a.URL + "\" style=\"display:inline-block;padding:8px 16px;background:" + btnColor + ";color:#fff;text-decoration:none;border-radius:4px;margin-right:8px;\">" + htmlEscape(a.Label) + "</a>"
	}

	return `<!DOCTYPE html>
<html>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;margin:0;padding:0;background:#f5f5f5;">
<table width="100%" cellpadding="0" cellspacing="0" style="max-width:600px;margin:20px auto;">
<tr>
<td style="background:` + color + `;height:4px;"></td>
</tr>
<tr>
<td style="background:#fff;padding:24px;">
<h2 style="margin:0 0 16px 0;color:#1a1a1a;">` + htmlEscape(n.Title) + `</h2>
` + func() string {
		if n.Message != "" {
			return `<p style="color:#555;margin:0 0 16px 0;">` + htmlEscape(n.Message) + `</p>`
		}
		return ""
	}() + `
<table style="width:100%;border-collapse:collapse;margin-bottom:16px;">
` + fieldsHTML + `
</table>
` + func() string {
		if actionsHTML != "" {
			return `<div style="margin-top:16px;">` + actionsHTML + `</div>`
		}
		return ""
	}() + `
</td>
</tr>
<tr>
<td style="background:#f9f9f9;padding:12px 24px;color:#999;font-size:12px;">
CloudMock Alert Notification
</td>
</tr>
</table>
</body>
</html>`
}

func severityColorHex(severity string) string {
	switch severity {
	case "critical":
		return "#E01E5A"
	case "warning":
		return "#ECB22E"
	case "info":
		return "#2EB67D"
	default:
		return "#36C5F0"
	}
}

// htmlEscape performs basic HTML escaping without importing html package.
func htmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
