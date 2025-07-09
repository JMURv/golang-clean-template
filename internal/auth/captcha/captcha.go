package captcha

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"go.uber.org/zap"
)

type Port interface {
	VerifyRecaptcha(token string, action Actions) (bool, error)
}

type Actions string

const (
	PassAuth Actions = "pass_auth"
)

const captchaScore = 0.1

type Core struct {
	secret string
}

func New(conf config.Config) *Core {
	return &Core{
		secret: conf.Auth.Captcha.Secret,
	}
}

func (c *Core) VerifyRecaptcha(token string, action Actions) (bool, error) {
	resp, err := http.PostForm(
		"https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {c.secret},
			"response": {token},
		},
	)
	if err != nil {
		zap.L().Error("failed to verify recaptcha", zap.Error(err))
		return false, err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			zap.L().Error("failed to close body", zap.Error(err))
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error("failed to read body", zap.Error(err))
		return false, err
	}

	var result dto.RecaptchaResponse
	if err = json.Unmarshal(body, &result); err != nil {
		zap.L().Error("failed to unmarshal body", zap.Error(err))
		return false, err
	}

	return result.Success && result.Score > captchaScore && result.Action == string(action), nil
}
