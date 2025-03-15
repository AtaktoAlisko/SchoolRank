package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"ranking-school/models"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)
var secretKey = []byte(os.Getenv("SECRET")) 
func RespondWithError(w http.ResponseWriter, status int, error models.Error) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(error)
}
func ResponseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
func ComparePasswords(hashedPassword string, password []byte) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), password)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}
func IsPhoneNumber(input string) bool {
	phoneRegex := regexp.MustCompile(`^\d{7,15}$`)
	return phoneRegex.MatchString(strings.TrimSpace(input))
}
func GenerateToken(user models.User) (string, error) {
    secret := os.Getenv("SECRET")
    if secret == "" {
        return "", errors.New("SECRET environment variable is not set")
    }

    // Ensure the user has either email or phone
    if user.Email == "" && user.Phone == "" {
        return "", errors.New("user must have either email or phone")
    }

    // Create the claims for the token
    claims := jwt.MapClaims{
        "iss": "course",    // Issuer of the token
        "user_id": user.ID, // Add user ID claim
        "role": user.Role,  // Add role claim (make sure this exists in your user model)
    }

    // Add email or phone depending on what the user provided
    if user.Email != "" {
        claims["email"] = user.Email
    } else if user.Phone != "" {
        claims["phone"] = user.Phone
    }

    // Generate the token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    // Sign the token with the secret
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        return "", err
    }

    return tokenString, nil
}
func GenerateVerificationToken(email string) (string, error) {
	secret := os.Getenv("SECRET")
	if secret == "" {
		return "", fmt.Errorf("SECRET environment variable is not set")
	}

	claims := jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), // Токен истекает через 24 часа
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
func ParseToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
}
func VerifyToken(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, errors.New("Authorization header missing")
	}

	tokenString := strings.Split(authHeader, " ")[1]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int(claims["user_id"].(float64))
		return userID, nil
	}
	return 0, errors.New("Invalid token")
}
func GenerateRefreshToken(user models.User) (string, error) {
    secret := os.Getenv("SECRET")
    if secret == "" {
        return "", errors.New("SECRET environment variable is not set")
    }

    // Create refresh token claims
    claims := jwt.MapClaims{
        "iss": "course",
        "user_id": user.ID, // Adding user_id
        "exp": time.Now().Add(30 * 24 * time.Hour).Unix(), // Refresh token validity (30 days)
    }

    // Adding email or phone based on provided information
    if user.Email != "" {
        claims["email"] = user.Email
    } else if user.Phone != "" {
        claims["phone"] = user.Phone
    }

    // Generate refresh token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        return "", err
    }

    return tokenString, nil
}
func SendEmail(to, subject, body string) {
    from := "mralibekmurat27@gmail.com"
    password := "bdyi mtae fqub cfcr"

    smtpHost := "smtp.gmail.com"
    smtpPort := "587"

    auth := smtp.PlainAuth("", from, password, smtpHost)

    msg := []byte("To: " + to + "\r\n" +
        "Subject: " + subject + "\r\n" +
        "\r\n" + body + "\r\n")

    err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
    if err != nil {
        log.Printf("Error sending email: %v", err)
    }
}
func GenerateResetToken(email string) string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
func GenerateOTP() (string, error) {
	num, err := rand.Int(rand.Reader, big.NewInt(10000)) // 4-значный код
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", num.Int64()), nil
}
func SendVerificationEmail(to, token, otp string) {
	from := "mralibekmurat27@gmail.com"
	password := "bdyi mtae fqub cfcr"

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Создаем ссылку с токеном
	verificationLink := fmt.Sprintf("http://localhost:8000/verify-email?token=%s", token)

	// Сообщение с ссылкой и OTP
	message := fmt.Sprintf(
		"Click here to verify your email: %s\n\nYour OTP code is: %s", 
		verificationLink, otp)

	// Формируем и отправляем письмо
	msg := []byte("To: " + to + "\r\n" +
		"Subject: Verify Your Email and OTP\r\n" +
		"\r\n" + message + "\r\n")

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("Error sending email: %v", err)
	}
}
func SendVerificationOTP(to, otp string) {
	 from := "mralibekmurat27@gmail.com"
	 password := "bdyi mtae fqub cfcr"

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	auth := smtp.PlainAuth("", from, password, smtpHost)

	message := fmt.Sprintf("Your email verification code is: %s", otp)

	msg := []byte("To: " + to + "\r\n" +
		"Subject: Email Verification Code\r\n" +
		"\r\n" + message + "\r\n")

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		log.Printf("Error sending email: %v", err)
	}
}
func NullableValue(value interface{}) interface{} {
    if value == nil {
        return nil
    }
    return value
}



