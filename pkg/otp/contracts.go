package otp

type Generator interface {
	Generate() (string, error)
}
