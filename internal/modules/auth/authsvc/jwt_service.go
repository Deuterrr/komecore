package authsvc

import (
	"fmt"
	"strings"

	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authrepo"
	appclock "komecore/pkg/clock"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTService struct {
	secretKey []byte
}

func NewJWTService(secret string) authrepo.TokenService {
	return &JWTService{
		secretKey: []byte(secret),
	}
}

type jwtClaims struct {
	UserID     string `json:"user_id"`
	SessionID  string `json:"session_id"`
	Type       string `json:"type"`
	StaffID    string `json:"staff_id,omitempty"`
	CustomerID string `json:"customer_id,omitempty"`
	Role       string `json:"roles,omitempty"` // comma-separated

	jwt.RegisteredClaims
}

func (j *JWTService) Generate(params authrepo.GenerateTokenParams) (authrepo.GeneratedToken, error) {
	now := appclock.Now()
	exp := now.Add(params.Duration)

	staffIDStr := ""
	if params.StaffID != nil {
		staffIDStr = params.StaffID.String()
	}

	customerIDStr := ""
	if params.CustomerID != nil {
		customerIDStr = params.CustomerID.String()
	}

	roles := make([]string, len(params.Roles))
	for i, role := range params.Roles {
		roles[i] = string(role)
	}
	rolesStr := strings.Join(roles, ",")

	claims := jwtClaims{
		UserID:     params.UserID.String(),
		SessionID:  params.SessionID.String(),
		Type:       string(params.Type),
		StaffID:    staffIDStr,
		CustomerID: customerIDStr,
		Role:       rolesStr,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signed, err := token.SignedString(j.secretKey)
	if err != nil {
		return authrepo.GeneratedToken{}, fmt.Errorf("sign jwt token failed: %w", err)
	}

	return authrepo.GeneratedToken{
		Token:     signed,
		ExpiresAt: exp,
		Type:      params.Type,
	}, nil
}

func (j *JWTService) Validate(tokenStr string) (*authdomain.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&jwtClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected jwt signing method")
			}

			return j.secretKey, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("parse jwt token failed: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, fmt.Errorf("invalid jwt claims type")
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid jwt token")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id claim: %w", err)
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session id claim: %w", err)
	}

	tknClaim := &authdomain.TokenClaims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      authdomain.TokenType(claims.Type),
		IssuedAt:  claims.IssuedAt.Time,
		ExpiresAt: claims.ExpiresAt.Time,
	}

	if claims.StaffID != "" {
		mid, err := uuid.Parse(claims.StaffID)
		if err != nil {
			return nil, fmt.Errorf("invalid staff_id claim: %w", err)
		}
		tknClaim.StaffID = &mid
	}
	if claims.CustomerID != "" {
		cid, err := uuid.Parse(claims.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("invalid customer_id claim: %w", err)
		}
		tknClaim.CustomerID = &cid
	}
	if claims.Role != "" {
		tknClaim.Roles = strings.Split(claims.Role, ",")
	}

	return tknClaim, nil
}
