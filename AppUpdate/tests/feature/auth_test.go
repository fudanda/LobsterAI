package feature

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"goravel/tests"
)

type AuthTestSuite struct {
	suite.Suite
	tests.TestCase
}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestLoginURLRoutes() {
	tests := []struct {
		name             string
		path             string
		expectedEnvQuery string
	}{
		{
			name:             "test login url",
			path:             "/openapi/get/luna/hardware/lobsterai/test/login-url",
			expectedEnvQuery: "env=test",
		},
		{
			name:             "prod login url",
			path:             "/openapi/get/luna/hardware/lobsterai/prod/login-url",
			expectedEnvQuery: "env=prod",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp, err := s.Http(s.T()).Get(tt.path)
			s.Require().NoError(err)
			resp.AssertOk()

			var body struct {
				Code int `json:"code"`
				Data struct {
					Value string `json:"value"`
				} `json:"data"`
			}
			s.Require().NoError(resp.Bind(&body))
			s.Equal(0, body.Code)
			s.Contains(body.Data.Value, "/mock-login?")
			s.Contains(body.Data.Value, tt.expectedEnvQuery)
		})
	}
}

func (s *AuthTestSuite) TestAuthFlow() {
	for _, environment := range []string{authEnvironmentTestValue, authEnvironmentProdValue} {
		s.Run(environment, func() {
			code := s.issueAuthCode(environment)
			session := s.exchangeAuthCode(code)

			s.verifyProtectedEndpoints(session.AccessToken, environment)

			refreshedAccessToken := s.refreshAccessToken(session.RefreshToken)
			s.NotEqual(session.AccessToken, refreshedAccessToken)

			s.logout(refreshedAccessToken)

			afterLogoutProfile, err := s.Http(s.T()).
				WithToken(refreshedAccessToken).
				Get("/api/user/profile")
			s.Require().NoError(err)
			afterLogoutProfile.AssertUnauthorized()
		})
	}
}

func (s *AuthTestSuite) TestInvalidAuthCodeAndRefreshToken() {
	invalidExchange, err := s.Http(s.T()).
		Post("/api/auth/exchange", strings.NewReader(`{"authCode":"invalid-auth-code"}`))
	s.Require().NoError(err)
	invalidExchange.AssertUnauthorized()

	invalidRefresh, err := s.Http(s.T()).
		Post("/api/auth/refresh", strings.NewReader(`{"refreshToken":"invalid-refresh-token"}`))
	s.Require().NoError(err)
	invalidRefresh.AssertUnauthorized()
}

func (s *AuthTestSuite) TestProtectedEndpointsRequireToken() {
	endpoints := []string{
		"/api/user/profile",
		"/api/user/quota",
		"/api/user/profile-summary",
		"/api/models/available",
	}

	for _, endpoint := range endpoints {
		resp, err := s.Http(s.T()).Get(endpoint)
		s.Require().NoError(err)
		resp.AssertUnauthorized()
	}
}

const (
	authEnvironmentTestValue = "test"
	authEnvironmentProdValue = "prod"
)

type authSessionTokens struct {
	AccessToken  string
	RefreshToken string
}

func (s *AuthTestSuite) issueAuthCode(environment string) string {
	mockLoginResp, err := s.Http(s.T()).Get(fmt.Sprintf("/mock-login?env=%s", environment))
	s.Require().NoError(err)
	mockLoginResp.AssertFound()

	location := mockLoginResp.Headers().Get("Location")
	s.NotEmpty(location)

	parsedLocation, err := url.Parse(location)
	s.Require().NoError(err)
	s.Equal("lobsterai", parsedLocation.Scheme)
	s.Equal("auth", parsedLocation.Host)
	s.Equal("/callback", parsedLocation.Path)

	code := parsedLocation.Query().Get("code")
	s.NotEmpty(code)

	return code
}

func (s *AuthTestSuite) exchangeAuthCode(code string) authSessionTokens {
	exchangeResp, err := s.Http(s.T()).
		Post("/api/auth/exchange", strings.NewReader(fmt.Sprintf(`{"authCode":"%s"}`, code)))
	s.Require().NoError(err)
	exchangeResp.AssertOk()

	var body struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			User         struct {
				Nickname string `json:"nickname"`
			} `json:"user"`
			Quota struct {
				PlanName string `json:"planName"`
			} `json:"quota"`
		} `json:"data"`
	}
	s.Require().NoError(exchangeResp.Bind(&body))
	s.Equal(0, body.Code)
	s.NotEmpty(body.Data.AccessToken)
	s.NotEmpty(body.Data.RefreshToken)
	s.NotEmpty(body.Data.User.Nickname)
	s.NotEmpty(body.Data.Quota.PlanName)

	return authSessionTokens{
		AccessToken:  body.Data.AccessToken,
		RefreshToken: body.Data.RefreshToken,
	}
}

func (s *AuthTestSuite) refreshAccessToken(refreshToken string) string {
	refreshResp, err := s.Http(s.T()).
		Post("/api/auth/refresh", strings.NewReader(fmt.Sprintf(`{"refreshToken":"%s"}`, refreshToken)))
	s.Require().NoError(err)
	refreshResp.AssertOk()

	var body struct {
		Code int `json:"code"`
		Data struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
		} `json:"data"`
	}
	s.Require().NoError(refreshResp.Bind(&body))
	s.Equal(0, body.Code)
	s.NotEmpty(body.Data.AccessToken)
	s.Equal(refreshToken, body.Data.RefreshToken)

	return body.Data.AccessToken
}

func (s *AuthTestSuite) logout(accessToken string) {
	logoutResp, err := s.Http(s.T()).
		WithToken(accessToken).
		Post("/api/auth/logout", nil)
	s.Require().NoError(err)
	logoutResp.AssertOk()
}

func (s *AuthTestSuite) verifyProtectedEndpoints(accessToken string, environment string) {
	envExpectedNickname := map[string]string{
		authEnvironmentTestValue: "Mock Test User",
		authEnvironmentProdValue: "Mock Prod User",
	}
	envExpectedModelID := map[string]string{
		authEnvironmentTestValue: "mock-test-model",
		authEnvironmentProdValue: "mock-prod-model",
	}

	profileResp, err := s.Http(s.T()).
		WithToken(accessToken).
		Get("/api/user/profile")
	s.Require().NoError(err)
	profileResp.AssertOk()
	var profileBody struct {
		Code int `json:"code"`
		Data struct {
			Nickname string `json:"nickname"`
		} `json:"data"`
	}
	s.Require().NoError(profileResp.Bind(&profileBody))
	s.Equal(0, profileBody.Code)
	s.Equal(envExpectedNickname[environment], profileBody.Data.Nickname)

	quotaResp, err := s.Http(s.T()).
		WithToken(accessToken).
		Get("/api/user/quota")
	s.Require().NoError(err)
	quotaResp.AssertOk()
	var quotaBody struct {
		Code int `json:"code"`
		Data struct {
			CreditsLimit     int `json:"creditsLimit"`
			CreditsRemaining int `json:"creditsRemaining"`
		} `json:"data"`
	}
	s.Require().NoError(quotaResp.Bind(&quotaBody))
	s.Equal(0, quotaBody.Code)
	s.Greater(quotaBody.Data.CreditsLimit, 0)
	s.GreaterOrEqual(quotaBody.Data.CreditsRemaining, 0)

	profileSummaryResp, err := s.Http(s.T()).
		WithToken(accessToken).
		Get("/api/user/profile-summary")
	s.Require().NoError(err)
	profileSummaryResp.AssertOk()
	var profileSummaryBody struct {
		Code int `json:"code"`
		Data struct {
			Nickname    string `json:"nickname"`
			CreditItems []struct {
				Type string `json:"type"`
			} `json:"creditItems"`
		} `json:"data"`
	}
	s.Require().NoError(profileSummaryResp.Bind(&profileSummaryBody))
	s.Equal(0, profileSummaryBody.Code)
	s.Equal(envExpectedNickname[environment], profileSummaryBody.Data.Nickname)
	s.NotEmpty(profileSummaryBody.Data.CreditItems)

	modelsResp, err := s.Http(s.T()).
		WithToken(accessToken).
		Get("/api/models/available")
	s.Require().NoError(err)
	modelsResp.AssertOk()
	var modelsBody struct {
		Code int `json:"code"`
		Data []struct {
			ModelID string `json:"modelId"`
		} `json:"data"`
	}
	s.Require().NoError(modelsResp.Bind(&modelsBody))
	s.Equal(0, modelsBody.Code)
	s.NotEmpty(modelsBody.Data)
	s.Equal(envExpectedModelID[environment], modelsBody.Data[0].ModelID)
}
