package shared

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// IsJWTTokenExpired This method returns true if JWT token is expired, otherwise returns false
func IsJWTTokenExpired(accessToken string) bool {
	token, tokenErr := jwt.Parse(accessToken, nil)
	if tokenErr != nil {
		return true
	}
	tokenClaims := token.Claims.(jwt.MapClaims)
	exp := tokenClaims["exp"].(float64)
	return exp-float64(time.Now().Unix()) <= 0
}
