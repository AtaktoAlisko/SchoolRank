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

    // Создаем payload для токена
    claims := jwt.MapClaims{
        "iss": "course",
        "user_id": user.ID, // Добавляем user_id
    }

    // Добавляем email или phone в зависимости от того, что предоставил пользователь
    if user.Email != "" {
        claims["email"] = user.Email
    } else if user.Phone != "" {
        claims["phone"] = user.Phone
    }

    // Генерация токена
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
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
		return 0, errors.New("missing token")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		return 0, errors.New("invalid token format")
	}

	secret := os.Getenv("SECRET")
	if secret == "" {
		return 0, errors.New("SECRET environment variable is not set")
	}

	// Парсинг токена
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Убедитесь, что это правильный метод подписания
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}

	// Извлекаем ID пользователя из токена
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("user_id not found in token")
	}

	return int(userID), nil
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



