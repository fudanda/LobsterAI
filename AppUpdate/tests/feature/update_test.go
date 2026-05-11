package feature

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"goravel/tests"
)

type UpdateTestSuite struct {
	suite.Suite
	tests.TestCase
}

func TestUpdateTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateTestSuite))
}

func (s *UpdateTestSuite) TestHealthz() {
	resp, err := s.Http(s.T()).Get("/healthz")
	s.Require().NoError(err)
	resp.AssertOk()
	resp.AssertJson(map[string]any{
		"ok": true,
	})
}

func (s *UpdateTestSuite) TestUpdateRoutesReturnEnvironmentConfig() {
	tests := []struct {
		name    string
		path    string
		version string
		title   string
	}{
		{
			name:    "test auto update",
			path:    "/openapi/get/luna/hardware/lobsterai/test/update",
			version: "2026.5.8",
			title:   "测试版本更新",
		},
		{
			name:    "test manual update",
			path:    "/openapi/get/luna/hardware/lobsterai/test/update-manual",
			version: "2026.5.8",
			title:   "测试版本更新",
		},
		{
			name:    "prod auto update",
			path:    "/openapi/get/luna/hardware/lobsterai/prod/update",
			version: "2026.5.8",
			title:   "版本更新",
		},
		{
			name:    "prod manual update",
			path:    "/openapi/get/luna/hardware/lobsterai/prod/update-manual",
			version: "2026.5.8",
			title:   "版本更新",
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
					Value struct {
						Version   string `json:"version"`
						ChangeLog struct {
							Ch struct {
								Title string `json:"title"`
							} `json:"ch"`
						} `json:"changeLog"`
					} `json:"value"`
				} `json:"data"`
			}
			s.Require().NoError(resp.Bind(&body))
			s.Equal(0, body.Code)
			s.Equal(tt.version, body.Data.Value.Version)
			s.Equal(tt.title, body.Data.Value.ChangeLog.Ch.Title)
		})
	}
}

func (s *UpdateTestSuite) TestUnknownEnvironmentReturnsNotFound() {
	resp, err := s.Http(s.T()).Get("/openapi/get/luna/hardware/lobsterai/dev/update")
	s.Require().NoError(err)
	resp.AssertNotFound()
}
