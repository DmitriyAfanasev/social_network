// Package notifications содержит адаптеры отправки уведомлений identity-сервиса.
package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"net/url"
	"strings"

	"general-project/identity/internal/ports"
)

// SMTP отправляет identity-уведомления через SMTP без хранения токенов.
type SMTP struct {
	addr string
	from string
}

// NewSMTP создаёт SMTP-адаптер уведомлений.
func NewSMTP(addr string, from string) *SMTP {
	return &SMTP{addr: addr, from: from}
}

// Send отправляет confirmation или password reset ссылку.
func (s *SMTP) Send(ctx context.Context, notification ports.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.addr == "" || s.from == "" || notification.Email == "" || notification.Token == "" {
		return fmt.Errorf("SMTP notification is not configured")
	}
	path := "confirm"
	subject := "Подтверждение регистрации"
	if notification.Purpose == "password_reset" {
		path = "password-reset"
		subject = "Сброс пароля"
	}
	link := strings.TrimRight(notification.BaseURL, "/") + "/" + path + "?token=" + url.QueryEscape(notification.Token)
	body := fmt.Sprintf("Здравствуйте!\n\nПерейдите по ссылке: %s\n", link)
	message := []byte("From: " + s.from + "\r\n" +
		"To: " + notification.Email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	return smtp.SendMail(s.addr, nil, s.from, []string{notification.Email}, message)
}
