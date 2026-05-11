package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/http"
	supportpath "github.com/goravel/framework/support/path"
)

const (
	defaultAuthMockConfigPath = "public/auth.mock.json"
	authMockConfigPathEnv     = "LOBSTERAI_AUTH_MOCK_CONFIG_PATH"
	authPublicBaseURLenv      = "LOBSTERAI_AUTH_PUBLIC_BASE_URL"

	authCodeTTL     = 10 * time.Minute
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

const (
	authResponseCodeSuccess = 0
	authResponseCodeFailure = 1
)

type authEnvironment string

const (
	authEnvironmentTest authEnvironment = "test"
	authEnvironmentProd authEnvironment = "prod"
)

type authMockConfig struct {
	Test authMockEnvironmentConfig `json:"test"`
	Prod authMockEnvironmentConfig `json:"prod"`
}

type authMockEnvironmentConfig struct {
	User           authMockUserProfile    `json:"user"`
	Quota          authMockQuota          `json:"quota"`
	ProfileSummary authMockProfileSummary `json:"profileSummary"`
	Models         []authMockModel        `json:"models"`
}

type authMockUserProfile struct {
	Yid       string `json:"yid"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Phone     string `json:"phone"`
	UserID    string `json:"userId"`
	ID        int    `json:"id"`
	Status    int    `json:"status"`
}

type authMockQuota struct {
	PlanName           string `json:"planName"`
	SubscriptionStatus string `json:"subscriptionStatus"`
	CreditsLimit       int    `json:"creditsLimit"`
	CreditsUsed        int    `json:"creditsUsed"`
	CreditsRemaining   int    `json:"creditsRemaining"`
}

type authMockProfileSummary struct {
	ID                    int                  `json:"id"`
	Nickname              string               `json:"nickname"`
	AvatarURL             string               `json:"avatarUrl"`
	TotalCreditsRemaining int                  `json:"totalCreditsRemaining"`
	CreditItems           []authMockCreditItem `json:"creditItems"`
}

type authMockCreditItem struct {
	Type             string `json:"type"`
	Label            string `json:"label"`
	LabelEn          string `json:"labelEn"`
	CreditsRemaining int    `json:"creditsRemaining"`
	ExpiresAt        string `json:"expiresAt"`
}

type authMockModel struct {
	ModelID       string `json:"modelId"`
	ModelName     string `json:"modelName"`
	Provider      string `json:"provider"`
	APIFormat     string `json:"apiFormat"`
	SupportsImage bool   `json:"supportsImage"`
}

type authExchangeRequest struct {
	AuthCode string `json:"authCode"`
}

type authRefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authResponseEnvelope struct {
	Code int    `json:"code"`
	Data any    `json:"data,omitempty"`
	Msg  string `json:"message,omitempty"`
}

type authCodeEntry struct {
	Environment authEnvironment
	ExpiresAt   time.Time
}

type authSession struct {
	Environment   authEnvironment
	AccessToken   string
	RefreshToken  string
	AccessExpiry  time.Time
	RefreshExpiry time.Time
}

type authStateStore struct {
	mu              sync.RWMutex
	authCodes       map[string]authCodeEntry
	accessSessions  map[string]authSession
	refreshSessions map[string]authSession
}

func newAuthStateStore() *authStateStore {
	return &authStateStore{
		authCodes:       make(map[string]authCodeEntry),
		accessSessions:  make(map[string]authSession),
		refreshSessions: make(map[string]authSession),
	}
}

func (s *authStateStore) issueAuthCode(environment authEnvironment) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code, err := generateRandomToken(16)
	if err != nil {
		return "", err
	}
	s.authCodes[code] = authCodeEntry{
		Environment: environment,
		ExpiresAt:   time.Now().Add(authCodeTTL),
	}
	return code, nil
}

func (s *authStateStore) consumeAuthCode(code string) (authEnvironment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.authCodes[code]
	if !exists {
		return "", false
	}
	delete(s.authCodes, code)
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}
	return entry.Environment, true
}

func (s *authStateStore) issueSession(environment authEnvironment) (authSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	accessToken, err := generateRandomToken(24)
	if err != nil {
		return authSession{}, err
	}
	refreshToken, err := generateRandomToken(24)
	if err != nil {
		return authSession{}, err
	}

	session := authSession{
		Environment:   environment,
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpiry:  time.Now().Add(accessTokenTTL),
		RefreshExpiry: time.Now().Add(refreshTokenTTL),
	}

	s.accessSessions[accessToken] = session
	s.refreshSessions[refreshToken] = session

	return session, nil
}

func (s *authStateStore) getSessionByAccessToken(accessToken string) (authSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.accessSessions[accessToken]
	if !exists {
		return authSession{}, false
	}
	if time.Now().After(session.AccessExpiry) {
		return authSession{}, false
	}
	return session, true
}

func (s *authStateStore) refreshSession(refreshToken string) (authSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.refreshSessions[refreshToken]
	if !exists {
		return authSession{}, false, nil
	}
	if time.Now().After(session.RefreshExpiry) {
		delete(s.refreshSessions, refreshToken)
		delete(s.accessSessions, session.AccessToken)
		return authSession{}, false, nil
	}

	newAccessToken, err := generateRandomToken(24)
	if err != nil {
		return authSession{}, false, err
	}

	delete(s.accessSessions, session.AccessToken)
	session.AccessToken = newAccessToken
	session.AccessExpiry = time.Now().Add(accessTokenTTL)
	s.accessSessions[newAccessToken] = session
	s.refreshSessions[refreshToken] = session

	return session, true, nil
}

func (s *authStateStore) revokeSessionByAccessToken(accessToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.accessSessions[accessToken]
	if !exists {
		return
	}
	delete(s.accessSessions, session.AccessToken)
	delete(s.refreshSessions, session.RefreshToken)
}

type AuthController struct {
	config authMockConfig
	store  *authStateStore
}

func NewAuthController() (*AuthController, error) {
	path := resolveAuthMockConfigPath()
	config, err := loadAuthMockConfig(path)
	if err != nil {
		return nil, err
	}

	return &AuthController{
		config: config,
		store:  newAuthStateStore(),
	}, nil
}

func (r *AuthController) GetTestLoginURL(ctx http.Context) http.Response {
	return r.respondLoginURL(ctx, authEnvironmentTest)
}

func (r *AuthController) GetProdLoginURL(ctx http.Context) http.Response {
	return r.respondLoginURL(ctx, authEnvironmentProd)
}

func (r *AuthController) MockLogin(ctx http.Context) http.Response {
	environment := parseAuthEnvironment(ctx.Request().Query("env", string(authEnvironmentTest)))
	if !environment.valid() {
		return r.fail(ctx, stdhttp.StatusBadRequest, "invalid env")
	}

	code, err := r.store.issueAuthCode(environment)
	if err != nil {
		return r.fail(ctx, stdhttp.StatusInternalServerError, "failed to issue auth code")
	}

	redirectURL := fmt.Sprintf("lobsterai://auth/callback?code=%s", url.QueryEscape(code))
	return ctx.Response().Redirect(stdhttp.StatusFound, redirectURL)
}

func (r *AuthController) Exchange(ctx http.Context) http.Response {
	var payload authExchangeRequest
	if err := ctx.Request().Bind(&payload); err != nil {
		return r.fail(ctx, stdhttp.StatusBadRequest, "invalid request body")
	}

	authCode := strings.TrimSpace(payload.AuthCode)
	if authCode == "" {
		return r.fail(ctx, stdhttp.StatusBadRequest, "authCode is required")
	}

	environment, ok := r.store.consumeAuthCode(authCode)
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "invalid authCode")
	}

	session, err := r.store.issueSession(environment)
	if err != nil {
		return r.fail(ctx, stdhttp.StatusInternalServerError, "failed to issue token")
	}

	envConfig, ok := r.config.getEnvironmentConfig(environment)
	if !ok {
		return r.fail(ctx, stdhttp.StatusInternalServerError, "environment config not found")
	}

	return r.success(ctx, http.Json{
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
		"user":         envConfig.User,
		"quota":        envConfig.Quota,
	})
}

func (r *AuthController) Refresh(ctx http.Context) http.Response {
	var payload authRefreshRequest
	if err := ctx.Request().Bind(&payload); err != nil {
		return r.fail(ctx, stdhttp.StatusBadRequest, "invalid request body")
	}

	refreshToken := strings.TrimSpace(payload.RefreshToken)
	if refreshToken == "" {
		return r.fail(ctx, stdhttp.StatusBadRequest, "refreshToken is required")
	}

	session, ok, err := r.store.refreshSession(refreshToken)
	if err != nil {
		return r.fail(ctx, stdhttp.StatusInternalServerError, "failed to refresh token")
	}
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "invalid refreshToken")
	}

	return r.success(ctx, http.Json{
		"accessToken":  session.AccessToken,
		"refreshToken": session.RefreshToken,
	})
}

func (r *AuthController) Logout(ctx http.Context) http.Response {
	accessToken, ok := extractBearerToken(ctx.Request().Header("Authorization"))
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "missing Authorization header")
	}

	r.store.revokeSessionByAccessToken(accessToken)
	return r.success(ctx, http.Json{})
}

func (r *AuthController) GetProfile(ctx http.Context) http.Response {
	envConfig, ok := r.authorizedEnvironmentConfig(ctx)
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "unauthorized")
	}
	return r.success(ctx, envConfig.User)
}

func (r *AuthController) GetQuota(ctx http.Context) http.Response {
	envConfig, ok := r.authorizedEnvironmentConfig(ctx)
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "unauthorized")
	}
	return r.success(ctx, envConfig.Quota)
}

func (r *AuthController) GetProfileSummary(ctx http.Context) http.Response {
	envConfig, ok := r.authorizedEnvironmentConfig(ctx)
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "unauthorized")
	}
	return r.success(ctx, envConfig.ProfileSummary)
}

func (r *AuthController) GetModels(ctx http.Context) http.Response {
	envConfig, ok := r.authorizedEnvironmentConfig(ctx)
	if !ok {
		return r.fail(ctx, stdhttp.StatusUnauthorized, "unauthorized")
	}
	return r.success(ctx, envConfig.Models)
}

func (r *AuthController) respondLoginURL(ctx http.Context, environment authEnvironment) http.Response {
	baseURL := strings.TrimSpace(os.Getenv(authPublicBaseURLenv))
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://%s", ctx.Request().Host())
	}
	baseURL = strings.TrimRight(baseURL, "/")

	value := fmt.Sprintf("%s/mock-login?env=%s", baseURL, environment)
	return r.success(ctx, http.Json{
		"value": value,
	})
}

func (r *AuthController) authorizedEnvironmentConfig(ctx http.Context) (authMockEnvironmentConfig, bool) {
	accessToken, ok := extractBearerToken(ctx.Request().Header("Authorization"))
	if !ok {
		return authMockEnvironmentConfig{}, false
	}

	session, ok := r.store.getSessionByAccessToken(accessToken)
	if !ok {
		return authMockEnvironmentConfig{}, false
	}

	return r.config.getEnvironmentConfig(session.Environment)
}

func (r *AuthController) success(ctx http.Context, data any) http.Response {
	return ctx.Response().Success().Json(authResponseEnvelope{
		Code: authResponseCodeSuccess,
		Data: data,
	})
}

func (r *AuthController) fail(ctx http.Context, statusCode int, message string) http.Response {
	return ctx.Response().Status(statusCode).Json(authResponseEnvelope{
		Code: authResponseCodeFailure,
		Msg:  message,
	})
}

func resolveAuthMockConfigPath() string {
	if value := strings.TrimSpace(os.Getenv(authMockConfigPathEnv)); value != "" {
		return value
	}
	return supportpath.Base(defaultAuthMockConfigPath)
}

func loadAuthMockConfig(configPath string) (authMockConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return authMockConfig{}, fmt.Errorf("open auth mock config %q: %w", configPath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var config authMockConfig
	if err := decoder.Decode(&config); err != nil {
		return authMockConfig{}, fmt.Errorf("decode auth mock config %q: %w", configPath, err)
	}

	if err := validateAuthMockConfig(config); err != nil {
		return authMockConfig{}, fmt.Errorf("validate auth mock config %q: %w", configPath, err)
	}

	return config, nil
}

func validateAuthMockConfig(config authMockConfig) error {
	if err := validateAuthMockEnvironmentConfig(config.Test, authEnvironmentTest); err != nil {
		return err
	}
	if err := validateAuthMockEnvironmentConfig(config.Prod, authEnvironmentProd); err != nil {
		return err
	}
	return nil
}

func validateAuthMockEnvironmentConfig(environmentConfig authMockEnvironmentConfig, environment authEnvironment) error {
	if strings.TrimSpace(environmentConfig.User.Nickname) == "" {
		return fmt.Errorf("%s.user.nickname is required", environment)
	}
	if strings.TrimSpace(environmentConfig.Quota.PlanName) == "" {
		return fmt.Errorf("%s.quota.planName is required", environment)
	}
	if environmentConfig.ProfileSummary.Nickname == "" {
		return fmt.Errorf("%s.profileSummary.nickname is required", environment)
	}
	if len(environmentConfig.Models) == 0 {
		return fmt.Errorf("%s.models must not be empty", environment)
	}
	for index, model := range environmentConfig.Models {
		if strings.TrimSpace(model.ModelID) == "" {
			return fmt.Errorf("%s.models[%d].modelId is required", environment, index)
		}
		if strings.TrimSpace(model.ModelName) == "" {
			return fmt.Errorf("%s.models[%d].modelName is required", environment, index)
		}
	}
	return nil
}

func (c authMockConfig) getEnvironmentConfig(environment authEnvironment) (authMockEnvironmentConfig, bool) {
	switch environment {
	case authEnvironmentTest:
		return c.Test, true
	case authEnvironmentProd:
		return c.Prod, true
	default:
		return authMockEnvironmentConfig{}, false
	}
}

func parseAuthEnvironment(value string) authEnvironment {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(authEnvironmentProd):
		return authEnvironmentProd
	case string(authEnvironmentTest):
		return authEnvironmentTest
	default:
		return ""
	}
}

func (e authEnvironment) valid() bool {
	return e == authEnvironmentTest || e == authEnvironmentProd
}

func extractBearerToken(headerValue string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(headerValue))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return parts[1], true
}

func generateRandomToken(byteCount int) (string, error) {
	if byteCount <= 0 {
		return "", errors.New("byteCount must be greater than 0")
	}
	buffer := make([]byte, byteCount)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
