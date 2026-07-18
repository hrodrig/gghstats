package alert

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

// SMTP delivers CanonicalText as a plain-text email (SPEC §8.5).
// Pattern matches groot internal/notifier/email.go: STARTTLS on 587, optional implicit TLS.
type SMTP struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	To         []string
	UseTLS     bool // implicit TLS (e.g. port 465)
	SkipVerify bool
}

func (s *SMTP) Type() string { return TypeSMTP }

func (s *SMTP) Send(_ context.Context, p Payload) error {
	if strings.TrimSpace(s.Host) == "" || strings.TrimSpace(s.From) == "" {
		return fmt.Errorf("smtp: host and from are required")
	}
	if len(s.To) == 0 {
		return fmt.Errorf("smtp: at least one recipient is required")
	}
	port := s.Port
	if port == 0 {
		port = 587
	}
	msg := buildSMTPMessage(s.From, s.To, p.OneLine(), p.CanonicalText())
	auth := smtpAuth(s.Host, s.Username, s.Password)
	addr := net.JoinHostPort(s.Host, strconv.Itoa(port))

	switch {
	case s.UseTLS:
		return s.sendImplicitTLS(addr, auth, msg)
	case port == 587:
		return s.sendSTARTTLS(addr, auth, msg)
	default:
		return smtp.SendMail(addr, auth, s.From, s.To, msg)
	}
}

func buildSMTPMessage(from string, recipients []string, subject, text string) []byte {
	body := strings.ReplaceAll(text, "\r\n", "\n")
	subj := strings.ReplaceAll(subject, "\n", " ")
	subj = strings.ReplaceAll(subj, "\r", "")
	if strings.TrimSpace(subj) == "" {
		subj = "gghstats alert"
	}
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + strings.Join(recipients, ", "),
		"Subject: " + subj,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return []byte(msg)
}

func smtpAuth(host, username, password string) smtp.Auth {
	if strings.TrimSpace(username) == "" {
		return nil
	}
	return smtp.PlainAuth("", username, password, host)
}

func (s *SMTP) tlsConfig() *tls.Config {
	cfg := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
	if s.SkipVerify {
		cfg.InsecureSkipVerify = true //nolint:gosec // operator opt-in for lab / self-signed
	}
	return cfg
}

func (s *SMTP) sendImplicitTLS(addr string, auth smtp.Auth, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, s.tlsConfig())
	if err != nil {
		return fmt.Errorf("smtp TLS dial: %w", err)
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	return deliverSMTP(client, s.From, s.To, msg)
}

func (s *SMTP) sendSTARTTLS(addr string, auth smtp.Auth, msg []byte) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(s.tlsConfig()); err != nil {
			return fmt.Errorf("smtp STARTTLS: %w", err)
		}
	}
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	return deliverSMTP(client, s.From, s.To, msg)
}

func deliverSMTP(client *smtp.Client, from string, recipients []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp RCPT TO %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

// splitRecipients accepts semicolon and/or comma separated addresses (groot uses ';').
func splitRecipients(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", ";")
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		u := strings.TrimSpace(p)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}
