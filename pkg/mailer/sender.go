package mailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPSender struct {
	host          string
	port          string
	username      string
	password      string
	senderAddress string
}

func NewSMTPSender(
	host string,
	port string,
	username string,
	password string,
	senderAddress string,
) Sender {
	return &SMTPSender{
		host:          host,
		port:          port,
		username:      username,
		password:      password,
		senderAddress: senderAddress,
	}
}

func (s *SMTPSender) Send(input SendInput) error {
	addr := fmt.Sprintf("%s:%s",
		strings.TrimRight(s.host, "/"),
		strings.TrimLeft(s.port, "/"),
	)
	auth := smtp.PlainAuth(
		"",
		s.username,
		s.password,
		s.host,
	)
	message := s.buildMessage(input)

	if err := smtp.SendMail(
		addr,
		auth,
		s.username,
		[]string{input.To},
		[]byte(message),
	); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *SMTPSender) buildMessage(input SendInput) string {
	var message strings.Builder

	message.WriteString(fmt.Sprintf("From: %s\r\n", s.senderAddress))
	message.WriteString(fmt.Sprintf("To: %s\r\n", input.To))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", input.Subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	if input.HTML != nil {
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	message.WriteString("\r\n")

	if input.HTML != nil {
		message.WriteString(*input.HTML)
	} else {
		message.WriteString(input.Text)
	}

	return message.String()
}
