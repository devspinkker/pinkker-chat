package jwt

import (
	"PINKKER-CHAT/config"
	"fmt"

	"github.com/golang-jwt/jwt"
)

func parseToken(tokenString string) (*jwt.Token, error) {
	TOKENPASSWORD := config.TOKENPASSWORD()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(TOKENPASSWORD), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("Invalid token")
	}

	return token, nil
}

func ExtractDataFromToken(tokenString string) (string, string, bool, error) {
	token, err := parseToken(tokenString)
	if err != nil {
		return "", "", false, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", false, fmt.Errorf("Invalid claims")
	}
	nameUser, ok := claims["nameuser"].(string)
	if !ok {
		return "", "", false, fmt.Errorf("Invalid nameUser")
	}
	_id, ok := claims["_id"].(string)
	if !ok {
		return "", "", false, fmt.Errorf("Invalid _id")
	}
	partner, ok := claims["partner"].(bool)
	if !ok {
		return "", "", false, fmt.Errorf("Invalid verified")
	}
	return nameUser, _id, partner, nil
}
