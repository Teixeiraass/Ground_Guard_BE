package mailer

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"strings"

	"github.com/Teixeiraass/ground_guard_be/util"
)

// Mailer sends transactional emails (verification, password reset) over SMTP.
// When no SMTP host is configured, Send logs the email instead of sending it,
// so local/dev environments keep working without real credentials.
type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewMailer(config util.Config) *Mailer {
	return &Mailer{
		host:     config.SMTPHost,
		port:     config.SMTPPort,
		username: config.SMTPUsername,
		password: config.SMTPPassword,
		from:     config.SMTPFrom,
	}
}

func (m *Mailer) SendVerificationEmail(to, link string) error {
	body, err := renderEmail(verificationEmailTemplate, map[string]string{"Link": link})
	if err != nil {
		return err
	}
	return m.send(to, "Confirme seu email - Ground Guard", body)
}

func (m *Mailer) SendPasswordResetEmail(to, link string) error {
	body, err := renderEmail(passwordResetEmailTemplate, map[string]string{"Link": link})
	if err != nil {
		return err
	}
	return m.send(to, "Redefinir senha - Ground Guard", body)
}

func renderEmail(tmplStr string, data map[string]string) (string, error) {
	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var body strings.Builder
	if err := tmpl.Execute(&body, data); err != nil {
		return "", err
	}

	return body.String(), nil
}

func (m *Mailer) send(to, subject, htmlBody string) error {
	if m.host == "" {
		log.Printf("[mailer] SMTP não configurado, email não enviado. to=%s subject=%s", to, subject)
		return nil
	}

	msg := buildMimeMessage(m.from, to, subject, htmlBody)
	addr := net.JoinHostPort(m.host, m.port)
	auth := smtp.PlainAuth("", m.username, m.password, m.host)

	if m.port == "465" {
		return sendMailImplicitTLS(addr, m.host, auth, m.from, to, msg)
	}

	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}

// sendMailImplicitTLS supports SMTP providers that require TLS from the first
// byte (port 465), which net/smtp.SendMail (STARTTLS-only) can't dial directly.
func sendMailImplicitTLS(addr, serverName string, auth smtp.Auth, from, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: serverName})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return client.Quit()
}

func buildMimeMessage(from, to, subject, htmlBody string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(htmlBody)
	return sb.String()
}
