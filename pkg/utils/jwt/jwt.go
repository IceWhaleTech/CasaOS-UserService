package jwt

import (
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	Username string `json:"username"`
	PassWord string `json:"password"`
	Id       int    `json:"id"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

func GenerateToken(username, password string, id int, issuer string, t time.Duration) (string, error) {
	clims := Claims{
		username,
		password,
		id,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    issuer,
		},
	}

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, clims)
	token, err := tokenClaims.SignedString(jwtSecret)
	return token, err

}

func ParseToken(token string, valid bool) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if tokenClaims != nil {
		if clims, ok := tokenClaims.Claims.(*Claims); ok {
			if valid && tokenClaims.Valid {
				return clims, nil
			} else if !valid {
				return clims, nil
			}
		}
	}
	return nil, err
}
