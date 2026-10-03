package service

import (
	"crypto/sha256"
	"encoding/hex"
	"komecore/internal/modules/auth/repository"
)

type sHA256TokenHasher struct{}

func NewSHATokenHasher() repository.TokenHasher {
	return &sHA256TokenHasher{}
}

func (tH *sHA256TokenHasher) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (tH *sHA256TokenHasher) Compare(hash string, token string) bool {
	return hash == tH.Hash(token)
}
