package domain

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}
