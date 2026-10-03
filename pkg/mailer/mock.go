package mailer

import "sync"

// MockSender is a thread-safe mock implementation of Sender for tests.
type MockSender struct {
	mu        sync.Mutex
	SentMails []SendInput
	Err       error
}

func NewMockSender() *MockSender {
	return &MockSender{
		SentMails: make([]SendInput, 0),
	}
}

func (m *MockSender) Send(input SendInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Err != nil {
		return m.Err
	}

	m.SentMails = append(m.SentMails, input)
	return nil
}

func (m *MockSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMails = m.SentMails[:0]
	m.Err = nil
}
