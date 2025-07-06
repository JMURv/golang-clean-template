package auth

import (
	"context"
	"time"

	"github.com/JMURv/golang-clean-template/internal/auth/captcha"
	"github.com/JMURv/golang-clean-template/internal/auth/jwt"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Core interface {
	Hash(pswd string) (string, error)
	ComparePasswords(hashed, pswd []byte) error
	jwt.Port
	captcha.Port
}

type Auth struct {
	jwt     jwt.Port
	captcha captcha.Port
}

func New(conf config.Config) *Auth {
	return &Auth{
		jwt:     jwt.New(conf),
		captcha: captcha.New(conf),
	}
}

func (a *Auth) Hash(val string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(val), bcrypt.MinCost)
	if err != nil {
		zap.L().Error(
			"Failed to generate hash",
			zap.String("val", val),
			zap.Error(err),
		)

		return "", err
	}

	return string(bytes), nil
}

func (a *Auth) ComparePasswords(hashed, pswd []byte) error {
	if err := bcrypt.CompareHashAndPassword(hashed, pswd); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (a *Auth) GetAccessTime() time.Time {
	return a.jwt.GetAccessTime()
}

func (a *Auth) GetRefreshTime() time.Time {
	return a.jwt.GetRefreshTime()
}

func (a *Auth) GenPair(ctx context.Context, uid uuid.UUID) (string, string, error) {
	return a.jwt.GenPair(ctx, uid)
}

func (a *Auth) NewToken(ctx context.Context, uid uuid.UUID, d time.Duration) (string, error) {
	return a.jwt.NewToken(ctx, uid, d)
}

func (a *Auth) ParseClaims(ctx context.Context, tokenStr string) (jwt.Claims, error) {
	return a.jwt.ParseClaims(ctx, tokenStr)
}

func (a *Auth) VerifyRecaptcha(token string, action captcha.Actions) (bool, error) {
	return a.captcha.VerifyRecaptcha(token, action)
}
