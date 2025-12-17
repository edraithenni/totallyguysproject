package utils

import (
    "crypto/rand"
    "fmt"
    "crypto/tls"
  //  "time"
    mail "gopkg.in/mail.v2"
    "golang.org/x/crypto/bcrypt"
    "github.com/joho/godotenv"
    "os"
   // "github.com/golang-jwt/jwt/v5"
)

var errorloading = godotenv.Load()
var apppassword = []byte(os.Getenv("APP_PASSWORD"))

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

func SendEmail(to, subject, body string) error {
    m := mail.NewMessage()
    m.SetHeader("From", "noreply@gmail.com")
    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)
    m.SetBody("text/html", body)

   d := mail.NewDialer(
        "smtp.gmail.com",
        587,
        "yana.prapirnaya@gmail.com",
        string(apppassword),
    )

    d.TLSConfig = &tls.Config{
        InsecureSkipVerify: true,
    }

    err := d.DialAndSend(m)
    if err != nil {
        fmt.Println("MAILTRAP SMTP ERROR:", err)
    }
    return err
}
