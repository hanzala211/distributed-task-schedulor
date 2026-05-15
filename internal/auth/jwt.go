package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthenticator struct {
	secret     string
	expiryTime time.Time
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTAuthenticator(secret string, expiryTime time.Time) *JWTAuthenticator {
	return &JWTAuthenticator{
		secret:     secret,
		expiryTime: expiryTime,
	}
}

func (a *JWTAuthenticator) GenerateToken(userID string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secret))
}

func (a *JWTAuthenticator) ValidateToken(token string) (string, error) {
	tokenParsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		_, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(a.secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := tokenParsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token")
	}
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.New("invalid token")
	}
	return userID, nil
}
