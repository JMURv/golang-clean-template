package jwt

import (
	"context"
	"time"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
)

type Port interface {
	GetAccessTime() time.Time
	GetRefreshTime() time.Time
	GenPair(ctx context.Context, uid uuid.UUID) (string, string, error)
	NewToken(ctx context.Context, uid uuid.UUID, d time.Duration) (string, error)
	ParseClaims(ctx context.Context, tokenStr string) (Claims, error)
}

type Core struct {
	secret []byte
	issuer string
}

type Claims struct {
	UID uuid.UUID `json:"uid"`
	jwt.RegisteredClaims
}

func New(conf config.Config) *Core {
	return &Core{secret: []byte(conf.Auth.JWT.Secret), issuer: conf.Auth.JWT.Issuer}
}

func (c *Core) GetAccessTime() time.Time {
	return time.Now().Add(config.AccessTokenDuration)
}

func (c *Core) GetRefreshTime() time.Time {
	return time.Now().Add(config.RefreshTokenDuration)
}

func (c *Core) GenPair(ctx context.Context, uid uuid.UUID) (string, string, error) {
	const op = "auth.GenPair.jwt"
	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	access, err := c.NewToken(ctx, uid, config.AccessTokenDuration)
	if err != nil {
		zap.L().Error(
			"Failed to generate token pair",
			zap.String("uid", uid.String()),
			zap.Error(err),
		)

		return "", "", err
	}

	refresh, err := c.NewToken(ctx, uid, config.RefreshTokenDuration)
	if err != nil {
		zap.L().Error(
			"Failed to generate token pair",
			zap.String("uid", uid.String()),
			zap.Error(err),
		)

		return "", "", err
	}

	return access, refresh, nil
}

func (c *Core) NewToken(ctx context.Context, uid uuid.UUID, d time.Duration) (string, error) {
	const op = "auth.NewToken.jwt"
	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	signed, err := jwt.NewWithClaims(
		jwt.SigningMethodHS256, &Claims{
			UID: uid,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(d)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    c.issuer,
			},
		},
	).SignedString(c.secret)
	if err != nil {
		zap.L().Error(
			ErrWhileCreatingToken.Error(),
			zap.Error(err),
		)

		return "", ErrWhileCreatingToken
	}

	return signed, nil
}

func (c *Core) ParseClaims(ctx context.Context, tokenStr string) (Claims, error) {
	const op = "auth.ParseClaims.jwt"
	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr, &claims, func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrUnexpectedSignMethod
			}

			return c.secret, nil
		},
	)
	if err != nil {
		zap.L().Error(
			"Failed to parse claims",
			zap.String("op", op),
			zap.Any("token", tokenStr),
			zap.Error(err),
		)

		return claims, err
	}

	if !token.Valid {
		zap.L().Debug(
			"Token is invalid",
			zap.String("op", op),
			zap.String("token", tokenStr),
		)

		return claims, ErrInvalidToken
	}

	return claims, nil
}
