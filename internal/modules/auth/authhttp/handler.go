package authhttp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"komecore/internal/apperror"
	appconfig "komecore/internal/config"
	"komecore/internal/httpx"
	appcookie "komecore/internal/httpx/cookie"
	"komecore/internal/modules/auth/authdomain"
	"komecore/internal/modules/auth/authusecase"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler = authHandler

type authHandler struct {
	me                   *authusecase.MeUsecase
	logout               *authusecase.LogoutUsecase
	loginCustomer        *authusecase.LoginCustomerUsecase
	loginStaff           *authusecase.LoginStaffUsecase
	registerCustomer     *authusecase.RegisterCustomerUsecase
	verifyAccount        *authusecase.VerifyAccountUsecase
	getAccount           *authusecase.GetAccountUsecase
	authenticateOAuth    *authusecase.AuthenticateOAuthUsecase
	requestPasswordReset *authusecase.RequestPasswordResetUsecase
	verifyPasswordReset  *authusecase.VerifyPasswordResetUsecase
	resetPassword        *authusecase.ResetPasswordUsecase
	refreshToken         *authusecase.RefreshTokenUsecase
	deleteAccount        *authusecase.DeleteAccountUsecase
	googleCfg            appconfig.GoogleOAuthConfig
}

func NewAuthHandler(
	me *authusecase.MeUsecase,
	logout *authusecase.LogoutUsecase,
	loginCustomer *authusecase.LoginCustomerUsecase,
	loginStaff *authusecase.LoginStaffUsecase,
	registerCustomer *authusecase.RegisterCustomerUsecase,
	verifyAccount *authusecase.VerifyAccountUsecase,
	getAccount *authusecase.GetAccountUsecase,
	authenticateOAuth *authusecase.AuthenticateOAuthUsecase,
	requestPasswordReset *authusecase.RequestPasswordResetUsecase,
	verifyPasswordReset *authusecase.VerifyPasswordResetUsecase,
	resetPassword *authusecase.ResetPasswordUsecase,
	refreshToken *authusecase.RefreshTokenUsecase,
	deleteAccount *authusecase.DeleteAccountUsecase,
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
	authCtx, err := httpx.RequireAuth(r)
	if err != nil {
		return err
	}

	acc, err := h.getAccount.Execute(r.Context(), authCtx.UserID)
	if err != nil {
		return err
	}
	if acc == nil {
		return apperror.NewNotFound("account not found")
	}

	response := map[string]any{
		"id":            acc.ID,
		"email":         acc.Email,
		"last_login_at": acc.LastLoginAt,
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) Me(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := httpx.RequireAuth(r)
	if err != nil {
		return err
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

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) SignInEmail(w http.ResponseWriter, r *http.Request) error {
	var req signInEmailRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperror.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperror.NewBadRequest("invalid password")
	}

	input := authusecase.LoginCustomerParams{
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

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) SignUpAccount(w http.ResponseWriter, r *http.Request) error {
	var req signUpRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperror.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperror.NewBadRequest("invalid password")
	}
	if req.Username == "" {
		return apperror.NewBadRequest("invalid user name")
	}

	input := authusecase.RegisterCustomerParams{
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

	httpx.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *authHandler) VerifyAccount(w http.ResponseWriter, r *http.Request) error {
	var req verifyAccountRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.ChallengeID == "" {
		return apperror.NewBadRequest("invalid challenge id")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperror.NewBadRequest("invalid challenge id")
	}
	if len(req.OTP) != 6 {
		return apperror.NewBadRequest("invalid otp")
	}

	input := authusecase.VerifyAccountParams{
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

	httpx.WriteJSON(w, http.StatusCreated, response)
	return nil
}

func (h *authHandler) SignInStaffEmail(w http.ResponseWriter, r *http.Request) error {
	var req signInEmailRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}

	if req.Email == "" {
		return apperror.NewBadRequest("invalid email")
	}
	if req.Password == "" {
		return apperror.NewBadRequest("invalid password")
	}

	input := authusecase.LoginStaffParams{
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

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := httpx.RequireAuth(r)
	if err != nil {
		return err
	}

	err = h.logout.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.Clear(w, appcookie.CookieCustomer)
	appcookie.Clear(w, appcookie.CookieCustomerRefresh)

	response := map[string]string{
		"message": "logout success",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) LogoutStaff(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := httpx.RequireAuth(r)
	if err != nil {
		return err
	}

	err = h.logout.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.Clear(w, appcookie.CookieStaff)
	appcookie.Clear(w, appcookie.CookieStaffRefresh)

	response := map[string]string{
		"message": "logout success",
	}

	httpx.WriteJSON(w, http.StatusOK, response)
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
		return apperror.NewInternal(fmt.Errorf("failed to generate state: %w", err))
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
		return apperror.NewBadRequest("missing oauth state cookie")
	}

	stateParam := httpx.Query(r, "state")
	if stateParam == "" || stateParam != cookie {
		return apperror.NewBadRequest("invalid oauth state")
	}

	appcookie.Clear(w, appcookie.CookieOAuthState)

	code := httpx.Query(r, "code")
	if code == "" {
		return apperror.NewBadRequest("missing oauth code")
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
		return apperror.NewUnauthorized(fmt.Sprintf("failed to exchange code: %v", err))
	}

	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return apperror.NewInternal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return apperror.NewInternal(fmt.Errorf("google userinfo returned status code %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apperror.NewInternal(err)
	}

	var googleUser struct {
		Sub     string  `json:"sub"`
		Name    string  `json:"name"`
		Email   string  `json:"email"`
		Picture *string `json:"picture"`
	}
	if err := json.Unmarshal(body, &googleUser); err != nil {
		return apperror.NewInternal(err)
	}

	if googleUser.Email == "" {
		return apperror.NewBadRequest("google did not provide email address")
	}

	userAgent := r.UserAgent()
	ipAddress := r.RemoteAddr

	params := authusecase.AuthenticateOAuthParams{
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
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.Email == "" {
		return apperror.NewBadRequest("email is required")
	}

	input := authusecase.RequestPasswordResetParams{
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

	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) ForgotPasswordStaff(w http.ResponseWriter, r *http.Request) error {
	var req forgotPasswordCustomerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.Email == "" {
		return apperror.NewBadRequest("email is required")
	}

	input := authusecase.RequestPasswordResetParams{
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
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) VerifyPasswordReset(w http.ResponseWriter, r *http.Request) error {
	var req verifyPasswordResetRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.ChallengeID == "" {
		return apperror.NewBadRequest("challenge_id is required")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperror.NewBadRequest("invalid challenge_id")
	}
	if len(req.OTP) != 6 {
		return apperror.NewBadRequest("invalid otp")
	}

	input := authusecase.VerifyPasswordResetParams{
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
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	var req resetPasswordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return apperror.NewBadRequest("invalid body request")
	}
	if req.ChallengeID == "" {
		return apperror.NewBadRequest("challenge_id is required")
	}
	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperror.NewBadRequest("invalid challenge_id")
	}
	if req.NewPassword == "" {
		return apperror.NewBadRequest("new_password is required")
	}

	input := authusecase.ResetPasswordParams{
		ChallengeID: challengeID,
		NewPassword: req.NewPassword,
	}
	if err := h.resetPassword.Execute(r.Context(), input); err != nil {
		return err
	}

	response := map[string]string{
		"message": "password reset successful",
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}

func (h *authHandler) RefreshCustomer(w http.ResponseWriter, r *http.Request) error {
	var req refreshTokenRequest
	_ = httpx.DecodeJSON(r, &req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookieVal, err := appcookie.Extract(r, appcookie.CookieRefreshToken); err == nil {
			refreshToken = cookieVal
		}
	}
	if refreshToken == "" {
		return apperror.NewUnauthorized("refresh token is required")
	}

	res, err := h.refreshToken.Execute(r.Context(), authusecase.RefreshTokenParams{
		RefreshToken: refreshToken,
		AccountType:  authdomain.AccountTypeCustomer,
	})
	if err != nil {
		return err
	}

	appcookie.Bind(w, appcookie.CookieAccessToken, res.AccessToken.Token, res.AccessToken.ExpiresAt)
	appcookie.Bind(w, appcookie.CookieRefreshToken, res.RefreshToken.Token, res.RefreshToken.ExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, refreshTokenResponse{
		AccessToken: res.AccessToken.Token,
	})
	return nil
}

func (h *authHandler) RefreshStaff(w http.ResponseWriter, r *http.Request) error {
	var req refreshTokenRequest
	_ = httpx.DecodeJSON(r, &req)

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		if cookieVal, err := appcookie.Extract(r, appcookie.CookieStaffRefreshToken); err == nil {
			refreshToken = cookieVal
		}
	}
	if refreshToken == "" {
		return apperror.NewUnauthorized("refresh token is required")
	}

	res, err := h.refreshToken.Execute(r.Context(), authusecase.RefreshTokenParams{
		RefreshToken: refreshToken,
		AccountType:  authdomain.AccountTypeStaff,
	})
	if err != nil {
		return err
	}

	appcookie.Bind(w, appcookie.CookieStaffAccessToken, res.AccessToken.Token, res.AccessToken.ExpiresAt)
	appcookie.Bind(w, appcookie.CookieStaffRefreshToken, res.RefreshToken.Token, res.RefreshToken.ExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, refreshTokenResponse{
		AccessToken: res.AccessToken.Token,
	})
	return nil
}

func (h *authHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) error {
	authCtx, err := httpx.RequireAuth(r)
	if err != nil {
		return err
	}

	err = h.deleteAccount.Execute(r.Context(), *authCtx)
	if err != nil {
		return err
	}

	appcookie.ClearAll(w)

	response := map[string]string{
		"message": "account deleted successfully",
	}
	httpx.WriteJSON(w, http.StatusOK, response)
	return nil
}
