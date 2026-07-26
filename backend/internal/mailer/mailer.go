// Package mailer sends plain-text email via SMTP (stdlib net/smtp) - no
// external mail API, consistent with this project's minimal-dependency style
// (the same reasoning that chose a small JWKS library over Firebase for
// Google sign-in - see documentations/features/ entry).
package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

// Client is safe to use with no host configured - Send becomes a logged no-op
// instead of failing, so local dev without SMTP credentials doesn't break
// registration (see documentations/guides/google-maps-setup.md step 8 for how
// to actually get credentials).
type Client struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func New(host, port, username, password, from string) *Client {
	return &Client{Host: host, Port: port, Username: username, Password: password, From: from}
}

// Enabled reports whether Send will actually attempt delivery.
func (c *Client) Enabled() bool {
	return c.Host != ""
}

// stripCRLF prevents SMTP header injection - to/subject can contain
// user-controlled data (a driver's own registered email address), and this
// message is built as a raw header block rather than going through a MIME
// library that would otherwise encode/escape them.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

func (c *Client) Send(to, subject, body string) error {
	if !c.Enabled() {
		return nil
	}

	to, subject = stripCRLF(to), stripCRLF(subject)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		c.From, to, subject, body)

	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
	addr := c.Host + ":" + c.Port
	if err := smtp.SendMail(addr, auth, c.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
