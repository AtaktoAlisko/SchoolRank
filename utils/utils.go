package utils

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"ranking-school/models"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

var secretKey = []byte(os.Getenv("SECRET"))

func RespondWithError(w http.ResponseWriter, status int, error models.Error) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(error); err != nil {
		log.Printf("Ошибка при отправке JSON ошибки: %v", err)
	}
}
func ResponseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Не удалось сформировать JSON", http.StatusInternalServerError)
	}
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
func GenerateToken(user models.User, expiration time.Duration) (string, error) {
	secret := os.Getenv("SECRET")
	if secret == "" {
		return "", errors.New("SECRET environment variable is not set")
	}

	if user.Email == "" && user.Phone == "" {
		return "", errors.New("user must have either email or phone")
	}

	// Create token with explicit expiration time
	expirationTime := time.Now().Add(expiration)

	claims := jwt.MapClaims{
		"iss":     "course",
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     expirationTime.Unix(),
		"iat":     time.Now().Unix(), // Issued at time
	}

	if user.Email != "" {
		claims["email"] = user.Email
	} else if user.Phone != "" {
		claims["phone"] = user.Phone
	}

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
func ParseToken(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("SECRET")
	if secret == "" {
		return nil, errors.New("SECRET environment variable is not set")
	}

	// Use ParseWithClaims to explicitly handle claims validation
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	// Handle validation errors
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New("token expired")
			}
		}
		return nil, err
	}

	return token, nil
}
func VerifyToken(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, errors.New("Authorization header missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return 0, errors.New("Invalid Authorization header format")
	}

	tokenString := parts[1]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("Invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("Invalid token claims")
	}

	// Extract userID
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("user_id not found in token")
	}

	return int(userIDFloat), nil // Return the userID (int), not the whole struct
}
func GenerateRefreshToken(user models.User, expiration time.Duration) (string, error) {
	secret := os.Getenv("SECRET")
	if secret == "" {
		return "", errors.New("SECRET environment variable is not set")
	}

	// Создаем claims для Refresh Token
	claims := jwt.MapClaims{
		"iss":     "course",
		"user_id": user.ID,                           // Добавляем user_id
		"exp":     time.Now().Add(expiration).Unix(), // Токен истекает через expiration времени
	}

	if user.Email != "" {
		claims["email"] = user.Email
	} else if user.Phone != "" {
		claims["phone"] = user.Phone
	}

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
func UploadFileToS3(file multipart.File, fileName string, fileType string) (string, error) {
	var accessKey, secretKey, region, bucketName string

	// Выбираем набор ключей и бакет в зависимости от типа файла
	switch fileType {
	case "avatar":
		accessKey = os.Getenv("AWS_ACCESS_KEY2_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS2_KEY")
		region = os.Getenv("AWS_REGION2")
		bucketName = "avatarschoolrank" // Бакет для аватаров
	case "schoolphoto":
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		region = os.Getenv("AWS_REGION")
		bucketName = "schoolrank-schoolphotos" // Бакет для школьных фото
	case "olympiaddoc":
		accessKey = os.Getenv("AWS_ACCESS_KEY3_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS3_KEY")
		region = os.Getenv("AWS_REGION3")
		bucketName = "olympiaddocument" // Бакет для документов олимпиад
	default:
		return "", fmt.Errorf("unknown file type: %s", fileType)
	}

	// Проверяем, что ключи и регион заданы
	if accessKey == "" || secretKey == "" || region == "" {
		return "", fmt.Errorf("AWS credentials or region not set in environment for %s", fileType)
	}

	// Создаем сессию с AWS
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Создаем клиент для S3
	svc := s3.New(sess)

	// Считываем файл в буфер
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	if err != nil {
		return "", fmt.Errorf("failed to read file buffer: %v", err)
	}

	// Задаем имя бакета
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader(buf.Bytes()),
	}

	// Загружаем файл в S3
	_, err = svc.PutObject(input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	// Формируем URL для доступа к файлу
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, fileName)
	return url, nil
}
func StrToInt(s string) (int, error) {
	s = strings.TrimSpace(s) // Убираем все пробельные символы (включая новую строку)
	return strconv.Atoi(s)
}
func DeleteFileFromS3(fileURL string) error {
	// Определяем, какой бакет использовать
	var accessKey, secretKey, region, bucketName string

	if strings.Contains(fileURL, "avatar") {
		// Для аватаров
		accessKey = os.Getenv("AWS_ACCESS_KEY2_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS2_KEY")
		region = os.Getenv("AWS_REGION2")
		bucketName = "avatarschoolrank" // Бакет для аватаров
	} else {
		// Для других файлов (школьных фото)
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		region = os.Getenv("AWS_REGION")
		bucketName = "your-school-photo-bucket" // Бакет для школьных фото
	}

	// Создаем сессию с AWS
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %v", err)
	}

	svc := s3.New(sess)
	// Извлекаем ключ из URL
	key := strings.TrimPrefix(fileURL, "https://"+bucketName+".s3."+region+".amazonaws.com/")

	// Удаляем объект из S3
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %v", err)
	}

	return nil
}
func GenerateRandomPassword(length int) string {
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println("Error generating random bytes:", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(randomBytes)[:length]
}
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
func GenerateAccessToken(user models.User, expiration time.Duration) (string, error) {
	secret := os.Getenv("SECRET")
	if secret == "" {
		return "", errors.New("SECRET environment variable is not set")
	}

	if user.Email == "" && user.Phone == "" {
		return "", errors.New("user must have either email or phone")
	}

	claims := jwt.MapClaims{
		"iss":     "course",
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(expiration).Unix(), // Используем expiration для времени жизни токена
	}

	if user.Email != "" {
		claims["email"] = user.Email
	} else if user.Phone != "" {
		claims["phone"] = user.Phone
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
func GetUserByID(db *sql.DB, userID int) (models.User, error) {
	var user models.User
	var email sql.NullString
	var phone sql.NullString

	// Try to find in users table first
	query := "SELECT id, email, phone, first_name, last_name, age, role FROM users WHERE id = ?"
	err := db.QueryRow(query, userID).Scan(&user.ID, &email, &phone, &user.FirstName, &user.LastName, &user.Age, &user.Role)

	// If not found in users table, check students table
	if err == sql.ErrNoRows {
		var studentGrade int
		studentQuery := "SELECT student_id, email, phone, first_name, last_name, grade, role FROM Student WHERE student_id = ?"
		err = db.QueryRow(studentQuery, userID).Scan(&user.ID, &email, &phone, &user.FirstName, &user.LastName, &studentGrade, &user.Role)
		if err != nil {
			return user, err
		}
	} else if err != nil {
		return user, err
	}

	// Set user properties
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}

	return user, nil
}
func IsTokenExpired(tokenString string) bool {
	token, err := ParseToken(tokenString)
	if err != nil {
		return true
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return true
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return true
	}

	return time.Now().Unix() > int64(exp)
}
func UploadFileToS3Compat(file multipart.File, fileName string, isAvatar bool) (string, error) {
	fileType := "schoolphoto"
	if isAvatar {
		fileType = "avatar"
	}
	return UploadFileToS3(file, fileName, fileType)
}
func GetUserIDFromToken(r *http.Request) (int, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return 0, errors.New("Authorization header missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return 0, errors.New("Invalid Authorization header format")
	}

	tokenString := parts[1]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Unexpected signing method")
		}
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("Invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("Invalid token claims")
	}

	// Extract userID
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("user_id not found in token")
	}

	return int(userIDFloat), nil
}
func NullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
