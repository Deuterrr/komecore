package mailer_test

import (
	"errors"
	"testing"

	"komecore/pkg/mailer"
)

func TestMockSender(t *testing.T) {
	mock := mailer.NewMockSender()

	err := mock.Send(mailer.SendInput{
		To:      "customer@example.com",
		Subject: "Test Email",
		Text:    "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.SentMails) != 1 {
		t.Fatalf("expected 1 mail sent, got %d", len(mock.SentMails))
	}

	if mock.SentMails[0].To != "customer@example.com" {
		t.Errorf("expected recipient customer@example.com, got %s", mock.SentMails[0].To)
	}

	// Test error injection
	mock.Err = errors.New("smtp failure")
	err = mock.Send(mailer.SendInput{To: "fail@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Test reset
	mock.Reset()
	if len(mock.SentMails) != 0 || mock.Err != nil {
		t.Fatal("expected reset to clear mails and err")
	}
}
