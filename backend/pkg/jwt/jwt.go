package jwt

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func Configure(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return errors.New("JWT_SECRET must be set and non-empty")
	}
	jwtSecret = []byte(secret)
	return nil
}

func getSecret() ([]byte, error) {
	if len(jwtSecret) == 0 {
		return nil, errors.New("JWT secret has not been configured")
	}
	return jwtSecret, nil
}

func GenerateToken(userID int64, expiresIn time.Duration) (string, error) {
	secret, err := getSecret()
	if err != nil {
		return "", err
	}
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseToken(tokenString string) (*Claims, error) {
	secret, err := getSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, err
	}
	return claims, nil
}
