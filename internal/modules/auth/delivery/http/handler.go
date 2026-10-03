package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	apperrors "komecore/internal/common/errors"
	apphttp "komecore/internal/common/http"
	appcookie "komecore/internal/common/http/cookie"
	appconfig "komecore/internal/config"
	authdomain "komecore/internal/modules/auth/domain"
	"komecore/internal/modules/auth/usecase"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler = authHandler

type authHandler struct {
	me                   *usecase.MeUsecase
	logout               *usecase.LogoutUsecase
	loginCustomer        *usecase.LoginCustomerUsecase
	loginStaff           *usecase.LoginStaffUsecase
	registerCustomer     *usecase.RegisterCustomerUsecase
	verifyAccount        *usecase.VerifyAccountUsecase
	getAccount           *usecase.GetAccountUsecase
	authenticateOAuth    *usecase.AuthenticateOAuthUsecase
	requestPasswordReset *usecase.RequestPasswordResetUsecase
	verifyPasswordReset  *usecase.VerifyPasswordResetUsecase
	resetPassword        *usecase.ResetPasswordUsecase
	refreshToken         *usecase.RefreshTokenUsecase
	deleteAccount        *usecase.DeleteAccountUsecase
	googleCfg            appconfig.GoogleOAuthConfig
}

func NewAuthHandler(
	me *usecase.MeUsecase,
	logout *usecase.LogoutUsecase,
	loginCustomer *usecase.LoginCustomerUsecase,
	loginStaff *usecase.LoginStaffUsecase,
	registerCustomer *usecase.RegisterCustomerUsecase,
	verifyAccount *usecase.VerifyAccountUsecase,
	getAccount *usecase.GetAccountUsecase,
	authenticateOAuth *usecase.AuthenticateOAuthUsecase,
	requestPasswordReset *usecase.RequestPasswordResetUsecase,
	verifyPasswordReset *usecase.VerifyPasswordResetUsecase,
	resetPassword *usecase.ResetPasswordUsecase,
	refreshToken *usecase.RefreshTokenUsecase,
	deleteAccount *usecase.DeleteAccountUsecase,
	googleCfg appconfig.GoogleOAuthConfig,
) *authHandler {
	return &authHandler{
		me:                   me,
		logout:               logout,
		loginCustomer:        loginCustomer,
		loginStaff:           loginStaff,
		registerCustomer:     registerCustomer,
		verifyAccount:        verifyAccount,
		getAccount:           getAccount,
		authenticateOAuth:    authenticateOAuth,
		requestPasswordReset: requestPasswordReset,
		verifyPasswordReset:  verifyPasswordReset,
		resetPassword:        resetPassword,
		refreshToken:         refreshToken,
		deleteAccount:        deleteAccount,
		googleCfg:            googleCfg,
	}
}

func (h *authHandler) GetByID(w http.ResponseWriter, r *http.Request) error {
	authCtx, ok := authdomain.GetAuthContext(r.Context())
	if !ok {
		return apperrors.NewUnauthorized("authentication required")
	}

	acc, err := h.getAccount.Execute(r.Context(), authCtx.UserID)
	if err != nil {
		return err
	}
	if acc == nil {
		return apperrors.NewNotFound("account not found")
	}

	response := map[string]any{
		"id":            acc.ID,
		"email":         acc.Email,
		"last_login_at": acc.LastLoginAt,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) Me(w http.ResponseWriter, r *http.Request) error {
	authCtx, ok := authdomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	me, err := h.me.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	roles := make([]roleResponse, 0, len(me.Actor.Roles))
	for _, role := range me.Actor.Roles {
		roles = append(roles, roleResponse{
			Code: string(role.Code),
			Name: role.Name,
		})
	}

	permissions := make([]permissionResponse, 0, len(me.Actor.Roles))
	for _, role := range me.Actor.Roles {
		permissions = append(permissions, permissionResponse{
			Code: string(role.Code),
		})
	}

	var avatarURL *string
	if me.User != nil {
		avatarURL = me.User.AvatarURL
	}

	var oauthProvider *string
	if me.OAuth != nil {
		providerStr := string(me.OAuth.Provider)
		oauthProvider = &providerStr
	}

	response := meResponse{
		AccountID:       me.Account.ID,
		AccountType:     string(me.Account.Type),
		IsAuthenticated: true,
		AvatarURL:       avatarURL,
		StaffID:         me.Actor.StaffID,
		Roles:           roles,
		Permissions:     permissions,
		OAuthProvider:   oauthProvider,
		LastLoginAt:     me.Account.LastLoginAt,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) SignInEmail(w http.ResponseWriter, r *http.Request) error {
	var req signInEmailRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperrors.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperrors.NewBadRequest("invalid password")
	}

	input := usecase.LoginCustomerParams{
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		Email:     req.Email,
		Password:  req.Password,
	}

	tokens, err := h.loginCustomer.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	var accessExpiry time.Time
	var refreshExpiry time.Time
	if req.RememberMe {
		accessExpiry = tokens.AccessToken.ExpiresAt
		refreshExpiry = tokens.RefreshToken.ExpiresAt
	}

	// Access token is bound to the response cookie
	// so it can be used for authenticated API requests
	appcookie.Bind(
		w,
		appcookie.CookieCustomer,
		tokens.AccessToken.Token,
		accessExpiry,
	)

	// Refresh token is bound to the response cookie
	// so it can be used to obtain a new access token
	appcookie.Bind(
		w,
		appcookie.CookieCustomerRefresh,
		tokens.RefreshToken.Token,
		refreshExpiry,
	)

	response := map[string]string{
		"message": "login success",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) SignUpAccount(w http.ResponseWriter, r *http.Request) error {
	var req signUpRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperrors.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperrors.NewBadRequest("invalid password")
	}
	if req.Username == "" {
		return apperrors.NewBadRequest("invalid user name")
	}

	input := usecase.RegisterCustomerParams{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Username: req.Username,
		Phone:    req.Phone,
	}

	challengeID, err := h.registerCustomer.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := signUpResponse{
		Message:     "verification code sent",
		ChallengeID: *challengeID,
	}

	apphttp.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *authHandler) VerifyAccount(w http.ResponseWriter, r *http.Request) error {
	var req verifyAccountRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	if req.ChallengeID == "" {
		return apperrors.NewBadRequest("invalid challenge id")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperrors.NewBadRequest("invalid challenge id")
	}
	if len(req.OTP) != 6 {
		return apperrors.NewBadRequest("invalid otp")
	}

	input := usecase.VerifyAccountParams{
		UserAgent:   req.UserAgent,
		IPAddress:   req.IPAddress,
		ChallengeID: challengeID,
		OTP:         req.OTP,
	}

	tokens, err := h.verifyAccount.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	// Access token is bound to the response cookie
	// so it can be used for authenticated API requests
	appcookie.Bind(
		w,
		appcookie.CookieCustomer,
		tokens.AccessToken.Token,
		tokens.AccessToken.ExpiresAt,
	)

	// Refresh token is bound to the response cookie
	// so it can be used to obtain a new access token
	appcookie.Bind(
		w,
		appcookie.CookieCustomerRefresh,
		tokens.RefreshToken.Token,
		tokens.RefreshToken.ExpiresAt,
	)

	response := map[string]string{
		"message": "verify success",
	}

	apphttp.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *authHandler) SignInStaffEmail(w http.ResponseWriter, r *http.Request) error {
	var req signInEmailRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperrors.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperrors.NewBadRequest("invalid password")
	}

	input := usecase.LoginStaffParams{
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		Email:     req.Email,
		Password:  req.Password,
	}

	tokens, err := h.loginStaff.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	var (
		accessExpiry  time.Time
		refreshExpiry time.Time
	)

	if req.RememberMe {
		accessExpiry = tokens.AccessToken.ExpiresAt
		refreshExpiry = tokens.RefreshToken.ExpiresAt
	}

	// Access token is bound to the response cookie
	// so it can be used for authenticated API requests
	appcookie.Bind(
		w,
		appcookie.CookieStaff,
		tokens.AccessToken.Token,
		accessExpiry,
	)

	// Refresh token is bound to the response cookie
	// so it can be used to obtain a new access token
	appcookie.Bind(
		w,
		appcookie.CookieStaffRefresh,
		tokens.RefreshToken.Token,
		refreshExpiry,
	)

	response := map[string]string{
		"message": "login success",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	authCtx, ok := authdomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	err := h.logout.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.Clear(w, appcookie.CookieCustomer)
	appcookie.Clear(w, appcookie.CookieCustomerRefresh)

	response := map[string]string{
		"message": "logout success",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) LogoutStaff(w http.ResponseWriter, r *http.Request) error {
	authCtx, ok := authdomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	err := h.logout.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.Clear(w, appcookie.CookieStaff)
	appcookie.Clear(w, appcookie.CookieStaffRefresh)

	response := map[string]string{
		"message": "logout success",
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) error {
	oauth2Config := &oauth2.Config{
		ClientID:     h.googleCfg.ClientID,
		ClientSecret: h.googleCfg.ClientSecret,
		RedirectURL:  h.googleCfg.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return apperrors.NewInternal(fmt.Errorf("failed to generate state: %w", err))
	}
	state := base64.URLEncoding.EncodeToString(b)

	appcookie.Bind(
		w,
		appcookie.CookieOAuthState,
		state,
		appclock.Now().Add(10*time.Minute),
	)

	url := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

func (h *authHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) error {
	cookie, err := appcookie.Extract(r, appcookie.CookieOAuthState)
	if err != nil {
		return apperrors.NewBadRequest("missing oauth state cookie")
	}

	stateParam := apphttp.Query(r, "state")
	if stateParam == "" || stateParam != cookie {
		return apperrors.NewBadRequest("invalid oauth state")
	}

	appcookie.Clear(w, appcookie.CookieOAuthState)

	code := apphttp.Query(r, "code")
	if code == "" {
		return apperrors.NewBadRequest("missing oauth code")
	}

	oauth2Config := &oauth2.Config{
		ClientID:     h.googleCfg.ClientID,
		ClientSecret: h.googleCfg.ClientSecret,
		RedirectURL:  h.googleCfg.RedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		return apperrors.NewUnauthorized(fmt.Sprintf("failed to exchange code: %v", err))
	}

	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return apperrors.NewInternal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperrors.NewInternal(fmt.Errorf("google userinfo returned status code %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apperrors.NewInternal(err)
	}

	var googleUser struct {
		Sub     string  `json:"sub"`
		Name    string  `json:"name"`
		Email   string  `json:"email"`
		Picture *string `json:"picture"`
	}
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return apperrors.NewInternal(err)
	}

	if googleUser.Email == "" {
		return apperrors.NewBadRequest("google did not provide email address")
	}

	userAgent := r.UserAgent()
	ipAddress := r.RemoteAddr

	params := usecase.AuthenticateOAuthParams{
		UserAgent: &userAgent,
		IPAddress: &ipAddress,
		Provider:  authdomain.OAuthProviderGoogle,
		Subject:   googleUser.Sub,
		Email:     googleUser.Email,
		Name:      googleUser.Name,
		AvatarURL: googleUser.Picture,
	}

	result, err := h.authenticateOAuth.Execute(r.Context(), params)
	if err != nil {
		return err
	}

	// Access token is bound to the response cookie
	// so it can be used for authenticated API requests
	appcookie.Bind(
		w,
		appcookie.CookieCustomer,
		result.AccessToken.Token,
		result.AccessToken.ExpiresAt,
	)

	// Refresh token is bound to the response cookie
	// so it can be used to obtain a new access token
	appcookie.Bind(
		w,
		appcookie.CookieCustomerRefresh,
		result.RefreshToken.Token,
		result.RefreshToken.ExpiresAt,
	)

	successRedirectURL := h.googleCfg.SuccessRedirectURL
	if successRedirectURL == "" {
		successRedirectURL = "/"
	}
	http.Redirect(w, r, successRedirectURL, http.StatusTemporaryRedirect)
	return nil
}

func (h *authHandler) ForgotPasswordCustomer(w http.ResponseWriter, r *http.Request) error {
	var req forgotPasswordCustomerRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}
	if req.Email == "" {
		return apperrors.NewBadRequest("email is required")
	}

	input := usecase.RequestPasswordResetParams{
		Email:       req.Email,
		AccountType: authdomain.AccountTypeCustomer,
	}
	challengeID, err := h.requestPasswordReset.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := forgotPasswordResponse{
		Message:     "if the email is registered you will receive a reset code shortly",
		ChallengeID: challengeID,
	}

	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) ForgotPasswordStaff(w http.ResponseWriter, r *http.Request) error {
	var req forgotPasswordCustomerRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}
	if req.Email == "" {
		return apperrors.NewBadRequest("email is required")
	}

	input := usecase.RequestPasswordResetParams{
		Email:       req.Email,
		AccountType: authdomain.AccountTypeStaff,
	}
	challengeID, err := h.requestPasswordReset.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := forgotPasswordResponse{
		Message:     "if the email is registered you will receive a reset code shortly",
		ChallengeID: challengeID,
	}
	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) VerifyPasswordReset(w http.ResponseWriter, r *http.Request) error {
	var req verifyPasswordResetRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}
	if req.ChallengeID == "" {
		return apperrors.NewBadRequest("challenge_id is required")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperrors.NewBadRequest("invalid challenge_id")
	}
	if len(req.OTP) != 6 {
		return apperrors.NewBadRequest("invalid otp")
	}

	input := usecase.VerifyPasswordResetParams{
		ChallengeID: challengeID,
		OTP:         req.OTP,
	}
	verifiedID, err := h.verifyPasswordReset.Execute(r.Context(), input)
	if err != nil {
		return err
	}

	response := verifyPasswordResetResponse{
		Message:     "otp verified",
		ChallengeID: *verifiedID,
	}
	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var req resetPasswordRequest
	if err := apphttp.DecodeJSON(r, &req); err != nil {
		return apperrors.NewBadRequest("invalid body request")
	}
	if req.ChallengeID == "" {
		return apperrors.NewBadRequest("challenge_id is required")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperrors.NewBadRequest("invalid challenge_id")
	}
	if req.NewPassword == "" {
		return apperrors.NewBadRequest("new_password is required")
	}

	input := usecase.ResetPasswordParams{
		ChallengeID: challengeID,
		NewPassword: req.NewPassword,
	}
	if err := h.resetPassword.Execute(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "password reset successful",
	}
	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) RefreshCustomer(w http.ResponseWriter, r *http.Request) error {
	var req refreshTokenRequest
	_ = apphttp.DecodeJSON(r, &req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookieVal, err := appcookie.Extract(r, appcookie.CookieRefreshToken); err == nil {
			refreshToken = cookieVal
		}
	}
	if refreshToken == "" {
		return apperrors.NewUnauthorized("refresh token is required")
	}

	res, err := h.refreshToken.Execute(r.Context(), usecase.RefreshTokenParams{
		RefreshToken: refreshToken,
		AccountType:  authdomain.AccountTypeCustomer,
	})
	if err != nil {
		return err
	}

	appcookie.Bind(w, appcookie.CookieAccessToken, res.AccessToken.Token, res.AccessToken.ExpiresAt)
	appcookie.Bind(w, appcookie.CookieRefreshToken, res.RefreshToken.Token, res.RefreshToken.ExpiresAt)

	apphttp.WriteJSON(w, http.StatusOK, refreshTokenResponse{
		AccessToken: res.AccessToken.Token,
	})
	return nil
}

func (h *authHandler) RefreshStaff(w http.ResponseWriter, r *http.Request) error {
	var req refreshTokenRequest
	_ = apphttp.DecodeJSON(r, &req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookieVal, err := appcookie.Extract(r, appcookie.CookieStaffRefreshToken); err == nil {
			refreshToken = cookieVal
		}
	}
	if refreshToken == "" {
		return apperrors.NewUnauthorized("refresh token is required")
	}

	res, err := h.refreshToken.Execute(r.Context(), usecase.RefreshTokenParams{
		RefreshToken: refreshToken,
		AccountType:  authdomain.AccountTypeStaff,
	})
	if err != nil {
		return err
	}

	appcookie.Bind(w, appcookie.CookieStaffAccessToken, res.AccessToken.Token, res.AccessToken.ExpiresAt)
	appcookie.Bind(w, appcookie.CookieStaffRefreshToken, res.RefreshToken.Token, res.RefreshToken.ExpiresAt)

	apphttp.WriteJSON(w, http.StatusOK, refreshTokenResponse{
		AccessToken: res.AccessToken.Token,
	})
	return nil
}

func (h *authHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) error {
	authCtx, ok := authdomain.GetAuthContext(r.Context())
	if !ok || !authCtx.IsAuthenticated {
		return apperrors.NewUnauthorized("authentication required")
	}

	err := h.deleteAccount.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.ClearAll(w)

	response := map[string]string{
		"message": "account deleted successfully",
	}
	apphttp.WriteJSON(w, http.StatusOK, response)
	return nil
}
