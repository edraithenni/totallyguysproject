package utils

import (
    "fmt"
    "time"
    "os"
    "github.com/golang-jwt/jwt/v5"
    "github.com/joho/godotenv"
)

var errorloadingenv = godotenv.Load()
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// GenerateJWT (userId+email) idk if its safe
func GenerateJWT(userID uint, email, role string) (string, error) {

	payload := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
        "role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Expires in 24 hours
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString(jwtSecret)
}

//verify jwt token
func ParseJWT(tokenStr string) (jwt.MapClaims, error) {

    token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return jwtSecret, nil
    })

    if err != nil || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("invalid claims")
    }
    //expired check
    if exp, ok := claims["exp"].(float64); ok {
        if time.Now().Unix() > int64(exp) {
            return nil, fmt.Errorf("token expired")
        }
    }

    return claims, nil
}
