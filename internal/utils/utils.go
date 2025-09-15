package utils

import (
    "crypto/rand"
  //  "fmt"
  //  "time"

    "golang.org/x/crypto/bcrypt"
   // "github.com/golang-jwt/jwt/v5"
)

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

func GenerateVerificationCode(length int) string {
    const digits = "0123456789"
    b := make([]byte, length)
    rand.Read(b)
    for i := range b {
        b[i] = digits[int(b[i])%len(digits)]
    }
    return string(b)
}
