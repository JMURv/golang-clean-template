package auth

import (
	"errors"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"time"
)

const AccessTokenDuration = time.Hour * 72

var Au *Auth
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidToken = errors.New("invalid token")

type AuthService interface {
	NewToken(uid uuid.UUID) (string, error)
	VerifyToken(tokenStr string) (map[string]any, error)
	HashPassword(pswd string) (string, error)
	ComparePasswords(hashed, pswd []byte) error
}

type Auth struct {
	secret []byte
}

func New(secret string) {
	Au = &Auth{secret: []byte(secret)}
}

func (a *Auth) HashPassword(pswd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pswd), 7)
	return string(bytes), err
}

func (a *Auth) ComparePasswords(hashed, pswd []byte) error {
	if err := bcrypt.CompareHashAndPassword(hashed, pswd); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func (a *Auth) NewToken(uid uuid.UUID) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256, jwt.MapClaims{
			"uid": uid,
			"exp": time.Now().Add(AccessTokenDuration).Unix(),
		},
	)

	return token.SignedString(a.secret)
}

func (a *Auth) VerifyToken(tokenStr string) (map[string]any, error) {
	token, err := jwt.Parse(
		tokenStr, func(token *jwt.Token) (any, error) {
			return a.secret, nil
		},
	)

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	} else {
		return nil, ErrInvalidToken
	}
}
