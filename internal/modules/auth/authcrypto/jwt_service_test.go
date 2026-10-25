package authcrypto_test

import (
	"testing"
	"time"

	"komecore/internal/modules/auth"
	"komecore/internal/modules/auth/authcrypto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateAndValidate_WithAccountID(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	svc := authcrypto.NewJWTService(secret)

	userID := uuid.New()
	accountID := uuid.New()
	sessionID := uuid.New()

	token, err := svc.Generate(auth.GenerateTokenParams{
		UserID:    userID,
		AccountID: accountID,
		SessionID: sessionID,
		Type:      auth.TokenTypeAccess,
		Duration:  15 * time.Minute,
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Token)

	claims, err := svc.Validate(token.Token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, accountID, claims.AccountID)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, auth.TokenTypeAccess, claims.Type)
}

func TestJWTService_Validate_MissingIssuedAtOrExpiresAt_DoesNotPanic(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")
	svc := authcrypto.NewJWTService(string(secret))

	userID := uuid.New()
	sessionID := uuid.New()

	// Token with neither IssuedAt nor ExpiresAt
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    userID.String(),
		"session_id": sessionID.String(),
		"type":       string(auth.TokenTypeAccess),
	})
	signed, err := token.SignedString(secret)
	require.NoError(t, err)

	claims, err := svc.Validate(signed)
	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.True(t, claims.IssuedAt.IsZero())
	assert.True(t, claims.ExpiresAt.IsZero())
}
