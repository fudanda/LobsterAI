package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/goravel/framework/contracts/http"
	supportpath "github.com/goravel/framework/support/path"
)

const (
	defaultConfigPath = "public/update.json"
	configPathEnv     = "LOBSTERAI_UPDATE_CONFIG_PATH"
)

type updateConfig struct {
	Test updateValue `json:"test"`
	Prod updateValue `json:"prod"`
}

type updateResponse struct {
	Code int                `json:"code"`
	Data updateResponseData `json:"data"`
}

type updateResponseData struct {
	Value updateValue `json:"value"`
}

type updateValue struct {
	Version    string           `json:"version"`
	Date       string           `json:"date"`
	ChangeLog  changeLog        `json:"changeLog"`
	MacIntel   platformDownload `json:"macIntel"`
	MacArm     platformDownload `json:"macArm"`
	WindowsX64 platformDownload `json:"windowsX64"`
}

type changeLog struct {
	Ch changeLogLang `json:"ch"`
	En changeLogLang `json:"en"`
}

type changeLogLang struct {
	Title   string   `json:"title"`
	Content []string `json:"content"`
}

type platformDownload struct {
	URL string `json:"url"`
}

type UpdateController struct {
	cfg updateConfig
}

func NewUpdateController() (*UpdateController, error) {
	path := resolveConfigPath()
	cfg, err := loadUpdateConfig(path)
	if err != nil {
		return nil, err
	}
	return &UpdateController{cfg: cfg}, nil
}

func (r *UpdateController) Healthz(ctx http.Context) http.Response {
	return ctx.Response().Success().Json(http.Json{
		"ok": true,
	})
}

func (r *UpdateController) GetTestUpdate(ctx http.Context) http.Response {
	return ctx.Response().Success().Json(updateResponse{
		Code: 0,
		Data: updateResponseData{
			Value: r.cfg.Test,
		},
	})
}

func (r *UpdateController) GetTestUpdateManual(ctx http.Context) http.Response {
	return ctx.Response().Success().Json(updateResponse{
		Code: 0,
		Data: updateResponseData{
			Value: r.cfg.Test,
		},
	})
}

func (r *UpdateController) GetProdUpdate(ctx http.Context) http.Response {
	return ctx.Response().Success().Json(updateResponse{
		Code: 0,
		Data: updateResponseData{
			Value: r.cfg.Prod,
		},
	})
}

func (r *UpdateController) GetProdUpdateManual(ctx http.Context) http.Response {
	return ctx.Response().Success().Json(updateResponse{
		Code: 0,
		Data: updateResponseData{
			Value: r.cfg.Prod,
		},
	})
}

func resolveConfigPath() string {
	if value := strings.TrimSpace(os.Getenv(configPathEnv)); value != "" {
		return value
	}
	return supportpath.Base(defaultConfigPath)
}

func loadUpdateConfig(configPath string) (updateConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return updateConfig{}, fmt.Errorf("open update config %q: %w", configPath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	var cfg updateConfig
	if err := decoder.Decode(&cfg); err != nil {
		return updateConfig{}, fmt.Errorf("decode update config %q: %w", configPath, err)
	}
	if err := validateUpdateConfig(cfg); err != nil {
		return updateConfig{}, fmt.Errorf("validate update config %q: %w", configPath, err)
	}

	return cfg, nil
}

func validateUpdateConfig(cfg updateConfig) error {
	if strings.TrimSpace(cfg.Test.Version) == "" {
		return errors.New("test.version is required")
	}
	if strings.TrimSpace(cfg.Prod.Version) == "" {
		return errors.New("prod.version is required")
	}
	return nil
}
