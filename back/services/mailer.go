package services

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// Mailer sends verification emails.
type Mailer interface {
	SendVerificationCode(email, code string, expiresIn time.Duration) error
}

type noopMailer struct{}

func (noopMailer) SendVerificationCode(email, code string, expiresIn time.Duration) error {
	return nil
}

// SMTPConfig stores SMTP connection settings.
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
	UseTLS      bool
}

// SMTPMailer sends emails through an SMTP server.
type SMTPMailer struct {
	config SMTPConfig
}

// LoadSMTPConfigFromEnv reads SMTP settings from env vars.
func LoadSMTPConfigFromEnv() (SMTPConfig, error) {
	port := 994
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return SMTPConfig{}, fmt.Errorf("invalid SMTP_PORT")
		}
		port = parsed
	}

	cfg := SMTPConfig{
		Host:        strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:        port,
		Username:    strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		Password:    strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		FromAddress: strings.TrimSpace(os.Getenv("SMTP_FROM")),
		FromName:    strings.TrimSpace(os.Getenv("SMTP_FROM_NAME")),
		UseTLS:      parseBoolEnv("SMTP_USE_TLS"),
	}

	if cfg.Host == "" || cfg.Username == "" || cfg.Password == "" || cfg.FromAddress == "" {
		return SMTPConfig{}, fmt.Errorf("SMTP config is incomplete")
	}
	return cfg, nil
}

// NewSMTPMailer creates a mailer backed by SMTP.
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) SendVerificationCode(email, code string, expiresIn time.Duration) error {
	subject := "AIRAWeb 注册验证码"
	body := buildVerificationEmailBody(code, expiresIn)
	msg := buildRFC822Message(formatFromHeader(m.config.FromName, m.config.FromAddress), m.config.FromAddress, email, subject, body)

	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)

	if m.config.UseTLS || m.config.Port == 465 {
		return m.sendWithImplicitTLS(addr, auth, email, msg)
	}
	return smtp.SendMail(addr, auth, m.config.FromAddress, []string{email}, []byte(msg))
}

func (m *SMTPMailer) sendWithImplicitTLS(addr string, auth smtp.Auth, email, message string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName:         m.config.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.config.FromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(email); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func buildVerificationEmailBody(code string, expiresIn time.Duration) string {
	minutes := int(expiresIn.Minutes())
	if minutes <= 0 {
		minutes = 10
	}
	return strings.Join([]string{
		"您好，",
		"",
		fmt.Sprintf("您的 AIRAWeb 注册验证码是：%s", code),
		fmt.Sprintf("验证码 %d 分钟内有效，请尽快完成注册。", minutes),
		"",
		"如果这不是您的操作，请忽略此邮件。",
	}, "\r\n")
}

func buildRFC822Message(fromHeader, fromAddress, to, subject, body string) string {
	headers := []string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	return strings.Join(headers, "\r\n") + "\r\n\r\n" + body
}

func formatFromHeader(name, address string) string {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

func parseBoolEnv(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
