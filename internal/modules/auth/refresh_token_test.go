package auth

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	transaction "komecore/internal/infra/transactor"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type refreshTokenTestSessionRepo struct {
	mu        sync.RWMutex
	sessions  map[uuid.UUID]*Session
	saveCount int32
}

func newRefreshTokenTestSessionRepo() *refreshTokenTestSessionRepo {
	return &refreshTokenTestSessionRepo{
		sessions: make(map[uuid.UUID]*Session),
	}
}

func (r *refreshTokenTestSessionRepo) GetByID(_ context.Context, _ transaction.Executor, id uuid.UUID) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sess, ok := r.sessions[id]
	if !ok || sess == nil {
		return nil, nil
	}
	copied := *sess
	return &copied, nil
}

func (r *refreshTokenTestSessionRepo) Save(_ context.Context, _ transaction.Executor, session Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	atomic.AddInt32(&r.saveCount, 1)
	copied := session
	r.sessions[session.ID] = &copied
	return nil
}

func (r *refreshTokenTestSessionRepo) RevokeByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

func (r *refreshTokenTestSessionRepo) RevokeAllByUserID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

func (r *refreshTokenTestSessionRepo) UpdateLastActivityByID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

type refreshTokenTestTokenRepo struct {
	mu          sync.RWMutex
	tokens      map[uuid.UUID]*RefreshToken
	saveCount   int32
	delayOnSave time.Duration
}

func newRefreshTokenTestTokenRepo() *refreshTokenTestTokenRepo {
	return &refreshTokenTestTokenRepo{
		tokens: make(map[uuid.UUID]*RefreshToken),
	}
}

func (r *refreshTokenTestTokenRepo) GetBySessionID(_ context.Context, _ transaction.Executor, sessionID uuid.UUID) (*RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tok, ok := r.tokens[sessionID]
	if !ok || tok == nil {
		return nil, nil
	}
	copied := *tok
	return &copied, nil
}

func (r *refreshTokenTestTokenRepo) Save(_ context.Context, _ transaction.Executor, rt RefreshToken) error {
	if r.delayOnSave > 0 {
		time.Sleep(r.delayOnSave)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	atomic.AddInt32(&r.saveCount, 1)
	copied := rt
	r.tokens[rt.SessionID] = &copied
	return nil
}

func (r *refreshTokenTestTokenRepo) RevokeBySessionID(_ context.Context, _ transaction.Executor, _ uuid.UUID) error {
	return nil
}

type refreshTokenTestTokenService struct {
	mu     sync.Mutex
	tokens map[string]*TokenClaims
	genSeq int
}

func newRefreshTokenTestTokenService() *refreshTokenTestTokenService {
	return &refreshTokenTestTokenService{
		tokens: make(map[string]*TokenClaims),
	}
}

func (s *refreshTokenTestTokenService) registerToken(token string, claims *TokenClaims) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = claims
}

func (s *refreshTokenTestTokenService) Validate(token string) (*TokenClaims, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claims, ok := s.tokens[token]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	return claims, nil
}

func (s *refreshTokenTestTokenService) Generate(params GenerateTokenParams) (GeneratedToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.genSeq++
	tokenStr := fmt.Sprintf("token_%s_%d", params.Type, s.genSeq)
	exp := time.Now().Add(params.Duration)

	roleStrs := make([]string, len(params.Roles))
	for i, r := range params.Roles {
		roleStrs[i] = string(r)
	}

	claims := &TokenClaims{
		UserID:     params.UserID,
		SessionID:  params.SessionID,
		StaffID:    params.StaffID,
		CustomerID: params.CustomerID,
		Type:       params.Type,
		Roles:      roleStrs,
		ExpiresAt:  exp,
	}
	s.tokens[tokenStr] = claims

	return GeneratedToken{
		Token:     tokenStr,
		ExpiresAt: exp,
	}, nil
}

type refreshTokenTestHasher struct{}

func (h *refreshTokenTestHasher) Hash(token string) string {
	return "hash:" + token
}

func (h *refreshTokenTestHasher) Compare(hash string, token string) bool {
	return hash == "hash:"+token
}

func setupRefreshTokenService(
	sessionRepo SessionRepository,
	refreshTokenRepo RefreshTokenRepository,
	tokenSvc TokenService,
	tokenHasher TokenHasher,
) *Service {
	return NewService(
		&mockExecutor{},
		&mockTransactor{},
		nil,
		sessionRepo,
		refreshTokenRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		tokenHasher,
		tokenSvc,
		nil,
		nil,
		nil,
		&mockAuditLogger{},
	)
}

func TestRefreshToken_Success(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)

	userID := uuid.New()
	custID := uuid.New()
	sessionID := uuid.New()
	initialToken := "initial_refresh_token"

	now := time.Now()
	sessionRepo.sessions[sessionID] = &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	refreshTokenRepo.tokens[sessionID] = &RefreshToken{
		ID:        uuid.New(),
		SessionID: sessionID,
		TokenHash: hasher.Hash(initialToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	tokenSvc.registerToken(initialToken, &TokenClaims{
		UserID:     userID,
		SessionID:  sessionID,
		CustomerID: &custID,
		Type:       TokenTypeRefresh,
		Roles:      []string{"customer"},
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
	})

	output, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
		RefreshToken: initialToken,
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.NotEmpty(t, output.AccessToken.Token)
	assert.NotEmpty(t, output.RefreshToken.Token)
	assert.NotEqual(t, initialToken, output.RefreshToken.Token)

	// Verify DB state updated
	assert.Equal(t, int32(1), atomic.LoadInt32(&sessionRepo.saveCount))
	assert.Equal(t, int32(1), atomic.LoadInt32(&refreshTokenRepo.saveCount))

	savedToken := refreshTokenRepo.tokens[sessionID]
	assert.Equal(t, hasher.Hash(output.RefreshToken.Token), savedToken.TokenHash)
}

func TestRefreshToken_ConcurrentRequests_GraceWindowAndSingleflight(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	// Add slight latency to simulate real DB work so goroutines contend simultaneously
	refreshTokenRepo.delayOnSave = 15 * time.Millisecond
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)
	svc.withGraceWindow(5 * time.Second)

	userID := uuid.New()
	custID := uuid.New()
	sessionID := uuid.New()
	sharedToken := "shared_concurrent_refresh_token"

	now := time.Now()
	sessionRepo.sessions[sessionID] = &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	refreshTokenRepo.tokens[sessionID] = &RefreshToken{
		ID:        uuid.New(),
		SessionID: sessionID,
		TokenHash: hasher.Hash(sharedToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	tokenSvc.registerToken(sharedToken, &TokenClaims{
		UserID:     userID,
		SessionID:  sessionID,
		CustomerID: &custID,
		Type:       TokenTypeRefresh,
		Roles:      []string{"customer"},
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
	})

	const numWorkers = 25
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	outputs := make([]*RefreshTokenOutput, numWorkers)
	errors := make([]error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startBarrier
			out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
				RefreshToken: sharedToken,
			})
			outputs[idx] = out
			errors[idx] = err
		}(i)
	}

	// Release all workers simultaneously
	close(startBarrier)
	wg.Wait()

	// All workers must succeed with 0 errors
	for i := 0; i < numWorkers; i++ {
		require.NoError(t, errors[i], "worker %d failed", i)
		require.NotNil(t, outputs[i], "worker %d returned nil output", i)
	}

	// All workers must receive the exact identical newly minted tokens
	firstAccess := outputs[0].AccessToken.Token
	firstRefresh := outputs[0].RefreshToken.Token
	for i := 1; i < numWorkers; i++ {
		assert.Equal(t, firstAccess, outputs[i].AccessToken.Token, "worker %d got different access token", i)
		assert.Equal(t, firstRefresh, outputs[i].RefreshToken.Token, "worker %d got different refresh token", i)
	}

	// Crucial: Singleflight coalesced the DB write so only 1 DB rotation occurred
	assert.Equal(t, int32(1), atomic.LoadInt32(&refreshTokenRepo.saveCount), "expected exactly 1 DB refresh token save")
	assert.Equal(t, int32(1), atomic.LoadInt32(&sessionRepo.saveCount), "expected exactly 1 DB session save")
}

func TestRefreshToken_GraceWindow_SuccessiveCallsWithinWindow(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)
	svc.withGraceWindow(1 * time.Second)

	userID := uuid.New()
	custID := uuid.New()
	sessionID := uuid.New()
	originalToken := "initial_token_for_grace"

	now := time.Now()
	sessionRepo.sessions[sessionID] = &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	refreshTokenRepo.tokens[sessionID] = &RefreshToken{
		ID:        uuid.New(),
		SessionID: sessionID,
		TokenHash: hasher.Hash(originalToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	tokenSvc.registerToken(originalToken, &TokenClaims{
		UserID:     userID,
		SessionID:  sessionID,
		CustomerID: &custID,
		Type:       TokenTypeRefresh,
		Roles:      []string{"customer"},
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
	})

	// 1st request succeeds and updates DB
	out1, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
		RefreshToken: originalToken,
	})
	require.NoError(t, err)
	require.NotNil(t, out1)
	assert.Equal(t, int32(1), atomic.LoadInt32(&refreshTokenRepo.saveCount))

	// Wait 20ms (well within 1-second grace window)
	time.Sleep(20 * time.Millisecond)

	// Sibling request arrives with the original token (e.g. parallel tab)
	out2, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
		RefreshToken: originalToken,
	})
	require.NoError(t, err)
	require.NotNil(t, out2)

	// Identical tokens returned from grace store
	assert.Equal(t, out1.AccessToken.Token, out2.AccessToken.Token)
	assert.Equal(t, out1.RefreshToken.Token, out2.RefreshToken.Token)

	// DB was NOT touched again
	assert.Equal(t, int32(1), atomic.LoadInt32(&refreshTokenRepo.saveCount))
	assert.Equal(t, int32(1), atomic.LoadInt32(&sessionRepo.saveCount))
}

func TestRefreshToken_GraceWindow_ExpiredGraceRejectsOldToken(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)
	// Set very short grace window of 30ms
	svc.withGraceWindow(30 * time.Millisecond)

	userID := uuid.New()
	custID := uuid.New()
	sessionID := uuid.New()
	originalToken := "token_that_will_expire_grace"

	now := time.Now()
	sessionRepo.sessions[sessionID] = &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	refreshTokenRepo.tokens[sessionID] = &RefreshToken{
		ID:        uuid.New(),
		SessionID: sessionID,
		TokenHash: hasher.Hash(originalToken),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	tokenSvc.registerToken(originalToken, &TokenClaims{
		UserID:     userID,
		SessionID:  sessionID,
		CustomerID: &custID,
		Type:       TokenTypeRefresh,
		Roles:      []string{"customer"},
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
	})

	// 1st request rotates the token
	out1, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
		RefreshToken: originalToken,
	})
	require.NoError(t, err)
	require.NotNil(t, out1)

	// Wait 60ms so the 30ms grace window expires
	time.Sleep(60 * time.Millisecond)

	// Stale attempt with original token after grace period has expired -> rejected!
	out2, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
		RefreshToken: originalToken,
	})
	assert.Error(t, err)
	assert.Nil(t, out2)
	assert.Contains(t, err.Error(), "refresh token mismatch")
}

func TestRefreshToken_GraceStorePruning(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)
	svc.withGraceWindow(15 * time.Millisecond)

	userID := uuid.New()
	custID := uuid.New()
	sessionID1 := uuid.New()
	sessionID2 := uuid.New()

	tkn1 := "prune_test_token_1"
	tkn2 := "prune_test_token_2"

	now := time.Now()
	sessionRepo.sessions[sessionID1] = &Session{ID: sessionID1, UserID: userID, ExpiresAt: now.Add(time.Hour)}
	sessionRepo.sessions[sessionID2] = &Session{ID: sessionID2, UserID: userID, ExpiresAt: now.Add(time.Hour)}

	refreshTokenRepo.tokens[sessionID1] = &RefreshToken{ID: uuid.New(), SessionID: sessionID1, TokenHash: hasher.Hash(tkn1), ExpiresAt: now.Add(time.Hour)}
	refreshTokenRepo.tokens[sessionID2] = &RefreshToken{ID: uuid.New(), SessionID: sessionID2, TokenHash: hasher.Hash(tkn2), ExpiresAt: now.Add(time.Hour)}

	tokenSvc.registerToken(tkn1, &TokenClaims{UserID: userID, SessionID: sessionID1, CustomerID: &custID, Type: TokenTypeRefresh, ExpiresAt: now.Add(time.Hour)})
	tokenSvc.registerToken(tkn2, &TokenClaims{UserID: userID, SessionID: sessionID2, CustomerID: &custID, Type: TokenTypeRefresh, ExpiresAt: now.Add(time.Hour)})

	// Rotate session 1
	_, err := svc.RefreshToken(context.Background(), RefreshTokenInput{RefreshToken: tkn1})
	require.NoError(t, err)

	svc.graceMu.RLock()
	_, exists1 := svc.graceStore[sessionID1]
	svc.graceMu.RUnlock()
	assert.True(t, exists1, "session 1 should be in graceStore")

	// Sleep 30ms to let session 1's grace expire
	time.Sleep(30 * time.Millisecond)

	// Rotate session 2 (this triggers pruning of expired entries)
	_, err = svc.RefreshToken(context.Background(), RefreshTokenInput{RefreshToken: tkn2})
	require.NoError(t, err)

	svc.graceMu.RLock()
	_, exists1After := svc.graceStore[sessionID1]
	_, exists2After := svc.graceStore[sessionID2]
	svc.graceMu.RUnlock()

	assert.False(t, exists1After, "expired session 1 should be pruned from graceStore")
	assert.True(t, exists2After, "session 2 should be active in graceStore")
}

func TestRefreshToken_ValidationAndAuthorizationErrors(t *testing.T) {
	sessionRepo := newRefreshTokenTestSessionRepo()
	refreshTokenRepo := newRefreshTokenTestTokenRepo()
	tokenSvc := newRefreshTokenTestTokenService()
	hasher := &refreshTokenTestHasher{}

	svc := setupRefreshTokenService(sessionRepo, refreshTokenRepo, tokenSvc, hasher)

	userID := uuid.New()
	custID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	t.Run("invalid unparseable token", func(t *testing.T) {
		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "non_existent_token",
		})
		assert.Error(t, err)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), ErrInvalidToken.Error())
	})

	t.Run("token is access token instead of refresh token", func(t *testing.T) {
		tokenSvc.registerToken("access_token", &TokenClaims{
			UserID:     userID,
			SessionID:  sessionID,
			CustomerID: &custID,
			Type:       TokenTypeAccess, // wrong type
			ExpiresAt:  now.Add(time.Hour),
		})
		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "access_token",
		})
		assert.Error(t, err)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "invalid token type")
	})

	t.Run("account type mismatch returns forbidden", func(t *testing.T) {
		staffID := uuid.New()
		tokenSvc.registerToken("staff_token", &TokenClaims{
			UserID:    userID,
			SessionID: sessionID,
			StaffID:   &staffID,
			Type:      TokenTypeRefresh,
			ExpiresAt: now.Add(time.Hour),
		})
		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "staff_token",
			AccountType:  AccountTypeCustomer,
		})
		assert.Error(t, err)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "customer account required")
	})

	t.Run("session revoked", func(t *testing.T) {
		revokedSessionID := uuid.New()
		revokedAt := now.Add(-time.Minute)
		sessionRepo.sessions[revokedSessionID] = &Session{
			ID:        revokedSessionID,
			UserID:    userID,
			ExpiresAt: now.Add(time.Hour),
			RevokedAt: &revokedAt,
		}
		refreshTokenRepo.tokens[revokedSessionID] = &RefreshToken{
			ID:        uuid.New(),
			SessionID: revokedSessionID,
			TokenHash: hasher.Hash("revoked_session_token"),
			ExpiresAt: now.Add(time.Hour),
		}
		tokenSvc.registerToken("revoked_session_token", &TokenClaims{
			UserID:     userID,
			SessionID:  revokedSessionID,
			CustomerID: &custID,
			Type:       TokenTypeRefresh,
			ExpiresAt:  now.Add(time.Hour),
		})

		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "revoked_session_token",
		})
		assert.Error(t, err)
		assert.Nil(t, out)
	})

	t.Run("session expired", func(t *testing.T) {
		expiredSessionID := uuid.New()
		sessionRepo.sessions[expiredSessionID] = &Session{
			ID:        expiredSessionID,
			UserID:    userID,
			ExpiresAt: now.Add(-time.Hour), // expired
		}
		refreshTokenRepo.tokens[expiredSessionID] = &RefreshToken{
			ID:        uuid.New(),
			SessionID: expiredSessionID,
			TokenHash: hasher.Hash("expired_session_token"),
			ExpiresAt: now.Add(time.Hour),
		}
		tokenSvc.registerToken("expired_session_token", &TokenClaims{
			UserID:     userID,
			SessionID:  expiredSessionID,
			CustomerID: &custID,
			Type:       TokenTypeRefresh,
			ExpiresAt:  now.Add(time.Hour),
		})

		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "expired_session_token",
		})
		assert.Error(t, err)
		assert.Nil(t, out)
	})

	t.Run("refresh token in DB is revoked", func(t *testing.T) {
		sessID := uuid.New()
		sessionRepo.sessions[sessID] = &Session{
			ID:        sessID,
			UserID:    userID,
			ExpiresAt: now.Add(time.Hour),
		}
		revokedAt := now.Add(-time.Minute)
		refreshTokenRepo.tokens[sessID] = &RefreshToken{
			ID:        uuid.New(),
			SessionID: sessID,
			TokenHash: hasher.Hash("revoked_db_token"),
			ExpiresAt: now.Add(time.Hour),
			RevokedAt: &revokedAt,
		}
		tokenSvc.registerToken("revoked_db_token", &TokenClaims{
			UserID:     userID,
			SessionID:  sessID,
			CustomerID: &custID,
			Type:       TokenTypeRefresh,
			ExpiresAt:  now.Add(time.Hour),
		})

		out, err := svc.RefreshToken(context.Background(), RefreshTokenInput{
			RefreshToken: "revoked_db_token",
		})
		assert.Error(t, err)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "refresh token is invalid or expired")
	})
}
