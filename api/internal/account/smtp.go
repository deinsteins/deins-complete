package account

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type SMTPMailer struct{ Addr, From, Username, Password string }

func (m SMTPMailer) SendMagicCode(ctx context.Context, to, code string) error {
	_ = ctx
	host, _, err := net.SplitHostPort(m.Addr)
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, host)
	}
	body := []byte("To: " + to + "\r\nFrom: " + m.From + "\r\nSubject: Your DeinsComplete sign-in code\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour DeinsComplete sign-in code is: " + code + "\r\n\r\nThis code expires shortly.\r\n")
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("recipient required")
	}
	return smtp.SendMail(m.Addr, auth, m.From, []string{to}, body)
}
