package mailer

type SendInput struct {
	To      string
	Subject string
	Text    string
	HTML    *string
}

type Sender interface {
	Send(input SendInput) error
}
