package captcha

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/dto"
	"github.com/goccy/go-json"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type Port interface {
	VerifyRecaptcha(ctx context.Context, token string, action Actions) (bool, error)
}

type Actions string

const (
	PassAuth Actions = "pass_auth"
)

const captchaScore = 0.1

type Core struct {
	enabled bool
	secret  string
}

func New(conf config.Config) *Core {
	return &Core{
		enabled: conf.Auth.Captcha.Enabled,
		secret:  conf.Auth.Captcha.Secret,
	}
}

func (c *Core) VerifyRecaptcha(ctx context.Context, token string, action Actions) (bool, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "VerifyRecaptcha")
	defer span.Finish()

	// Use for testing purposes
	if !c.enabled {
		return true, nil
	}

	resp, err := http.PostForm(
		"https://www.google.com/recaptcha/api/siteverify",
		url.Values{
			"secret":   {c.secret},
			"response": {token},
		},
	)
	if err != nil {
		span.SetTag(config.ErrorSpanTag, true)
		zap.L().Error("failed to verify recaptcha", zap.Error(err))
		return false, err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			span.SetTag(config.ErrorSpanTag, true)
			zap.L().Error("failed to close body", zap.Error(err))
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		span.SetTag(config.ErrorSpanTag, true)
		zap.L().Error("failed to read body", zap.Error(err))
		return false, err
	}

	var result dto.RecaptchaResponse
	if err = json.Unmarshal(body, &result); err != nil {
		span.SetTag(config.ErrorSpanTag, true)
		zap.L().Error("failed to unmarshal body", zap.Error(err))
		return false, err
	}

	score := result.Success && result.Score > captchaScore
	if !score {
		zap.L().Debug("not enough score", zap.Float64("score", result.Score))
	}
	return score && result.Action == string(action), nil
}
