package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Mailer sends plain-text emails via SMTP.
type Mailer struct {
	host string
	port string
	user string
	pass string
	from string
}

func New(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

func (m *Mailer) Send(to, subject, body string) error {
	// PlainAuth is skipped when no credentials are configured (e.g. local Mailpit).
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	msg := strings.Join([]string{
		"From: " + m.from,
		"To: " + to,
		"Subject: " + subject,
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(fmt.Sprintf("%s:%s", m.host, m.port), auth, m.from, []string{to}, []byte(msg))
}
