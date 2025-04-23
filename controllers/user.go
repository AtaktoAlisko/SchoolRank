package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"ranking-school/models"
	"ranking-school/utils"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)
func (c *Controller) Signup(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var user models.User
        var error models.Error

        // Декодируем запрос
        err := json.NewDecoder(r.Body).Decode(&user)
        if err != nil {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем формат даты рождения и вычисляем возраст
        if user.DateOfBirth != "" {
            // Проверка формата даты
            _, err := time.Parse("2006-01-02", user.DateOfBirth)
            if err != nil {
                error.Message = "Invalid date format. Please use YYYY-MM-DD format."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Вычисляем возраст, используя метод контроллера
            age, err := c.CalculateAge(&user)
            if err != nil {
                error.Message = "Error calculating age."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // Устанавливаем вычисленный возраст
            user.Age = age
            
            // Logging for debugging
            log.Printf("Date of birth: %s, Calculated age: %d", user.DateOfBirth, user.Age)
        } else {
            // If date of birth is not provided, explicitly set empty values
            user.DateOfBirth = ""
            user.Age = 0
        }

        // Проверка аутентификации суперадмина
        isCreatedBySuperAdmin := false
        
        // Извлекаем токен авторизации
        authHeader := r.Header.Get("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenString := strings.Split(authHeader, " ")[1]
            token, err := utils.ParseToken(tokenString)
            
            if err == nil && token.Valid {
                if claims, ok := token.Claims.(jwt.MapClaims); ok {
                    // Проверяем роль пользователя в токене
                    if role, exists := claims["role"].(string); exists && role == "superadmin" {
                        isCreatedBySuperAdmin = true
                        
                        // Дополнительная проверка ID суперадмина в базе
                        if userID, exists := claims["user_id"].(float64); exists {
                            var exists bool
                            err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ? AND role = 'superadmin')", int(userID)).Scan(&exists)
                            if err != nil || !exists {
                                isCreatedBySuperAdmin = false
                            }
                        }
                    }
                }
            }
        }

        // Если пользователь создается суперадмином и указан тип schooladmin
        if isCreatedBySuperAdmin && user.Role == "schooladmin" {
            
            // Проверяем наличие email, так как для админа школы обязателен email
            if user.Email == "" {
                error.Message = "Email is required for school administrator accounts."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Проверяем формат email
            if !strings.Contains(user.Email, "@") {
                error.Message = "Invalid email format."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Проверяем, существует ли уже email в базе
            var existingID int
            err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&existingID)
            if err == nil {
                error.Message = "Email already exists."
                utils.RespondWithError(w, http.StatusConflict, error)
                return
            } else if err != sql.ErrNoRows {
                log.Printf("Error checking existing user: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // Если пароль не указан, генерируем случайный пароль
            var plainPassword string
            if user.Password == "" {
                // Генерируем случайный пароль для школьного администратора (12 символов)
                randomBytes := make([]byte, 9) // 9 байтов дадут 12 символов в base64
                _, err := rand.Read(randomBytes)
                if err != nil {
                    log.Printf("Error generating random bytes: %v", err)
                    error.Message = "Failed to generate password."
                    utils.RespondWithError(w, http.StatusInternalServerError, error)
                    return
                }
                
                // Используем функцию для генерации пароля из случайных байтов
                plainPassword = generateRandomPassword(12)
            } else {
                plainPassword = user.Password
            }
            
            // Хешируем пароль для сохранения в базе
            hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
            if err != nil {
                log.Printf("Error hashing password: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            user.Password = string(hash)
            
            // Устанавливаем дефолтный аватар, если не указан
            if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
                user.AvatarURL = sql.NullString{String: "", Valid: false}
            }
            
            // Генерируем токен для верификации (для совместимости со схемой БД)
            verificationToken, err := utils.GenerateVerificationToken(user.Email)
            if err != nil {
                log.Printf("Error generating verification token: %v", err)
                error.Message = "Failed to generate verification token."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // ВАЖНО: Устанавливаем verified = true для schooladmin
            // Вставка данных в базу (учетная запись школьного администратора создается уже верифицированной)
            query := "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)"
            result, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
            if err != nil {
                log.Printf("Error inserting user: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // Получаем ID созданного пользователя
            userID, _ := result.LastInsertId()
            
            // Проверяем, что пользователь действительно создан и установлен verified = true
            var verified int
            err = db.QueryRow("SELECT verified FROM users WHERE id = ?", userID).Scan(&verified)
            if err != nil || verified != 1 {
                log.Printf("Failed to verify user creation or verification status: %v", err)
                // Если не удалось проверить или verified = false, исправляем это
                _, updateErr := db.Exec("UPDATE users SET verified = 1 WHERE id = ?", userID)
                if updateErr != nil {
                    log.Printf("Failed to update verification status: %v", updateErr)
                }
            }
            
            // Отправляем email администратору школы с данными для входа
            if plainPassword != user.Password {
                subject := "Ваши учетные данные для входа в систему"
                body := fmt.Sprintf(
                    "Уважаемый %s %s,\n\n"+
                    "Для вас был создан аккаунт школьного администратора.\n\n"+
                    "Логин: %s\n"+
                    "Пароль: %s\n\n"+
                    "Рекомендуем сменить пароль после первого входа в систему.\n\n"+
                    "С уважением,\n"+
                    "Администрация системы",
                    user.FirstName, user.LastName, user.Email, plainPassword,
                )
                
                utils.SendEmail(user.Email, subject, body)
            }
            
            // Формируем ответ суперадмину
            response := map[string]interface{}{
                "message": "School administrator account created successfully. Login credentials sent to email.",
                "email": user.Email,
            }
            
            utils.ResponseJSON(w, response)
            return
        }

        // Проверяем, если регистрируется суперадмин
        if user.Role == "superadmin" {
            // Проверяем наличие email для суперадмина
            if user.Email == "" {
                error.Message = "Email is required for superadmin accounts."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Проверяем формат email
            if !strings.Contains(user.Email, "@") {
                error.Message = "Invalid email format."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Проверяем, существует ли уже email в базе
            var existingID int
            err = db.QueryRow("SELECT id FROM users WHERE email = ?", user.Email).Scan(&existingID)
            if err == nil {
                error.Message = "Email already exists."
                utils.RespondWithError(w, http.StatusConflict, error)
                return
            } else if err != sql.ErrNoRows {
                log.Printf("Error checking existing user: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // Проверяем наличие пароля
            if user.Password == "" {
                error.Message = "Password is required for superadmin accounts."
                utils.RespondWithError(w, http.StatusBadRequest, error)
                return
            }
            
            // Хешируем пароль
            hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
            if err != nil {
                log.Printf("Error hashing password: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            user.Password = string(hash)
            
            // Устанавливаем дефолтный аватар, если не указан
            if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
                user.AvatarURL = sql.NullString{String: "", Valid: false}
            }
            
            // Генерируем токен для верификации (для совместимости со схемой БД)
            verificationToken, err := utils.GenerateVerificationToken(user.Email)
            if err != nil {
                log.Printf("Error generating verification token: %v", err)
                error.Message = "Failed to generate verification token."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // ВАЖНО: Устанавливаем verified = 1 для superadmin
            // MySQL/SQL использует 1 для true и 0 для false в булевых полях
            query := "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?)"
            result, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
            if err != nil {
                log.Printf("Error inserting superadmin user: %v", err)
                error.Message = "Server error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            // Получаем ID созданного пользователя
            userID, _ := result.LastInsertId()
            
            // Дополнительная проверка и принудительное обновление статуса verified если нужно
            var actualVerified int
            err = db.QueryRow("SELECT verified FROM users WHERE id = ?", userID).Scan(&actualVerified)
            if err == nil && actualVerified != 1 {
                // Если по какой-то причине verified не установлен в 1, исправляем это
                _, updateErr := db.Exec("UPDATE users SET verified = 1 WHERE id = ?", userID)
                if updateErr != nil {
                    log.Printf("Failed to update verification status for superadmin: %v", updateErr)
                } else {
                    log.Printf("Fixed verification status for superadmin: %s", user.Email)
                }
            }
            
            // Формируем ответ для создания суперадмина
            response := map[string]interface{}{
                "message": "Superadmin account created successfully. Account is already verified.",
                "email": user.Email,
            }
            
            utils.ResponseJSON(w, response)
            return
        }
        
        // Устанавливаем роль "user" по умолчанию
        if user.Role == "" {
            user.Role = "user"
        }

        // Устанавливаем дефолтный аватар, если не указан
        if !user.AvatarURL.Valid || user.AvatarURL.String == "" {
            user.AvatarURL = sql.NullString{String: "", Valid: false} // Если аватар не указан, записываем NULL в базе данных
        }

        // Проверяем, что email или телефон предоставлены
        if user.Email == "" && user.Phone == "" {
            error.Message = "Email or phone is required."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем, что формат email или телефона правильный
        var isEmail bool
        if user.Email != "" && strings.Contains(user.Email, "@") {
            isEmail = true
        } else if user.Phone != "" && utils.IsPhoneNumber(user.Phone) {
            isEmail = false
        } else {
            error.Message = "Invalid email or phone format."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем, что пароль не пустой
        if user.Password == "" {
            error.Message = "Password is required."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем, существует ли уже email или телефон в базе
        var existingID int
        var query string
        var identifier string

        if isEmail {
            query = "SELECT id FROM users WHERE email = ?"
            identifier = user.Email
        } else {
            query = "SELECT id FROM users WHERE phone = ?"
            identifier = user.Phone
        }

        err = db.QueryRow(query, identifier).Scan(&existingID)
        if err == nil {
            error.Message = "Email or phone already exists."
            utils.RespondWithError(w, http.StatusConflict, error)
            return
        } else if err != sql.ErrNoRows {
            log.Printf("Error checking existing user: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Хэшируем пароль
        hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
        if err != nil {
            log.Printf("Error hashing password: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        user.Password = string(hash)

        // Генерация OTP кода для верификации
        otpCode, err := utils.GenerateOTP()
        if err != nil {
            log.Printf("Error generating OTP: %v", err)
            error.Message = "Failed to generate OTP."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Генерация токена для верификации
        verificationToken, err := utils.GenerateVerificationToken(user.Email)
        if err != nil {
            log.Printf("Error generating verification token: %v", err)
            error.Message = "Failed to generate verification token."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Logging before insert
        log.Printf("Inserting user with date_of_birth: %s, age: %d", user.DateOfBirth, user.Age)

        // Вставка данных в базу
        if isEmail {
            query = "INSERT INTO users (email, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, false, ?, ?)"
            _, err = db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, otpCode, verificationToken)
        } else {
            query = "INSERT INTO users (phone, password, first_name, last_name, date_of_birth, age, role, avatar_url, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?, true, NULL, ?)"
            _, err = db.Exec(query, user.Phone, user.Password, user.FirstName, user.LastName, user.DateOfBirth, user.Age, user.Role, user.AvatarURL, verificationToken)
        }

        if err != nil {
            log.Printf("Error inserting user: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Verify the data was inserted correctly - grab the user ID and check the inserted values
        if isEmail {
            var insertedID int
            var insertedDOB string
            var insertedAge int
            errVerify := db.QueryRow("SELECT id, date_of_birth, age FROM users WHERE email = ?", user.Email).
                Scan(&insertedID, &insertedDOB, &insertedAge)
            if errVerify == nil {
                log.Printf("Verified user insertion: ID=%d, DOB=%s, Age=%d", insertedID, insertedDOB, insertedAge)
            } else {
                log.Printf("Could not verify user insertion: %v", errVerify)
            }
        }

        // Отправка email с OTP
        if isEmail {
            utils.SendVerificationEmail(user.Email, verificationToken, otpCode)
        }

        user.Password = "" // Убираем пароль из ответа

        // Формируем сообщение для пользователя в зависимости от типа регистрации
        var message string
        if isEmail {
            message = "User registered successfully. Please verify your email with the OTP code."
        } else {
            message = "User registered successfully."
        }

        // Создаем ответ без date_of_birth и age
        response := map[string]interface{}{
            "message": message,
        }

        // Добавляем OTP код только для пользователей с email, которым нужна верификация
        if isEmail {
            response["otp_code"] = otpCode
        }

        // Добавляем avatar_url в ответ, только если оно задано
        if user.AvatarURL.Valid && user.AvatarURL.String != "" {
            response["avatar_url"] = user.AvatarURL.String
        }

        utils.ResponseJSON(w, response)
    }
}
func (c *Controller) GetMe(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Проверяем токен и получаем userID
        id, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Запрос к базе для получения данных пользователя, включая date_of_birth и age
        var user models.User
        var email sql.NullString
        var role sql.NullString
        var avatarURL sql.NullString
        var dateOfBirth sql.NullString
        var age sql.NullInt64 // Use nullable int for age
        var passwordHash sql.NullString // Хэш пароля (но мы не будем его возвращать)

        err = db.QueryRow("SELECT id, first_name, last_name, email, role, avatar_url, date_of_birth, age, password FROM users WHERE id = ?", id).
            Scan(&user.ID, &user.FirstName, &user.LastName, &email, &role, &avatarURL, &dateOfBirth, &age, &passwordHash)

        if err != nil {
            if err == sql.ErrNoRows {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found"})
            } else {
                log.Printf("Error fetching user data: %v", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: err.Error()})
            }
            return
        }

        // Log the fetched values for debugging
        log.Printf("GetMe: Fetched user ID=%d with DOB=%v, Age=%v", id, dateOfBirth, age)

        // Если email не NULL, присваиваем его
        if email.Valid {
            user.Email = email.String
        }

        // Если роль не NULL, присваиваем роль
        if role.Valid {
            user.Role = role.String
        }

        // Если дата рождения или возраст равны NULL, используем корректные значения
        var dateOfBirthStr string
        var ageValue int
        
        if dateOfBirth.Valid {
            dateOfBirthStr = dateOfBirth.String
        } else {
            dateOfBirthStr = "" // Ensure empty string is returned rather than null
        }
        
        if age.Valid {
            ageValue = int(age.Int64)
        } else {
            ageValue = 0
            
            // Если есть дата рождения, но нет возраста, вычисляем его
            if dateOfBirth.Valid && dateOfBirth.String != "" {
                dob, err := time.Parse("2006-01-02", dateOfBirth.String)
                if err == nil {
                    now := time.Now()
                    ageValue = now.Year() - dob.Year()
                    
                    // Корректируем возраст, если день рождения еще не был в этом году
                    if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
                        ageValue--
                    }
                    
                    // Обновляем возраст в базе данных, если его пришлось вычислить
                    _, updateErr := db.Exec("UPDATE users SET age = ? WHERE id = ?", ageValue, id)
                    if updateErr != nil {
                        log.Printf("Failed to update age in database: %v", updateErr)
                    } else {
                        log.Printf("Updated missing age value to %d for user ID %d", ageValue, id)
                    }
                }
            }
        }

        // Создаем кастомную карту для ответа
        userMap := map[string]interface{}{
            "id":           user.ID,
            "email":        user.Email,
            "first_name":   user.FirstName,
            "last_name":    user.LastName,
            "role":         user.Role,
            "age":          ageValue,
            "date_of_birth": dateOfBirthStr, // Always include date_of_birth, even if empty
        }

        // Только если avatar_url существует, добавляем его
        if avatarURL.Valid {
            userMap["avatar_url"] = avatarURL.String
        } else {
            userMap["avatar_url"] = nil
        }

        // Возвращаем кастомный ответ
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(userMap)
    }
}
func (c *Controller) Login(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var user models.User
        var error models.Error

        // Decode request body
        err := json.NewDecoder(r.Body).Decode(&user)
        if err != nil {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        var query string
        var identifier string
        var hashedPassword string
        var email sql.NullString
        var phone sql.NullString
        var role string
        var verified bool

        // Check that email, phone, or login are provided
        if user.Email != "" {
            query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified FROM users WHERE email = ?"
            identifier = user.Email
        } else if user.Phone != "" {
            query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified FROM users WHERE phone = ?"
            identifier = user.Phone
        } else if user.Login != "" {  // If login is provided
            query = "SELECT id, email, phone, password, first_name, last_name, age, role, verified FROM users WHERE login = ?"
            identifier = user.Login
        } else {
            error.Message = "Email, phone, or login is required."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Try to find the user
        row := db.QueryRow(query, identifier)
        err = row.Scan(&user.ID, &email, &phone, &hashedPassword, &user.FirstName, &user.LastName, &user.Age, &role, &verified)

        // If user is not found, check students table
        if err == sql.ErrNoRows {
            error.Message = "User not found."
            utils.RespondWithError(w, http.StatusNotFound, error)
            return
        }

        // Check if the user is verified
      // Check if the user is verified (но только если это обычный пользователь)
if !verified && role == "user" {
    error.Message = "Email not verified. Please verify your email before logging in."
    utils.RespondWithError(w, http.StatusForbidden, error)
    return
}


        // Check password for non-students
        err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
        if err != nil {
            error.Message = "Invalid password."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Generate access token
        accessToken, err := utils.GenerateToken(user)
        if err != nil {
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Generate refresh token
        refreshToken, err := utils.GenerateRefreshToken(user)
        if err != nil {
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Send tokens in response
        utils.ResponseJSON(w, map[string]string{
            "token":         accessToken,
            "refresh_token": refreshToken,
        })
    }
}
func (c *Controller) GetAllUsers(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Запрос для получения всех пользователей из базы
        rows, err := db.Query("SELECT id, email, first_name, last_name, date_of_birth, role, password FROM users")
        if err != nil {
            log.Printf("Error fetching users: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Server error"})
            return
        }
        defer rows.Close()

        var users []map[string]interface{}

        // Проходим по всем строкам в результатах запроса
        for rows.Next() {
            var user models.User
            var password string
            var dateOfBirth sql.NullString // Используем sql.NullString для работы с NULL значениями

            // Извлекаем данные пользователя
            err := rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &dateOfBirth, &user.Role, &password)
            if err != nil {
                log.Printf("Error scanning user: %v", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error scanning user data"})
                return
            }

            // Преобразуем date_of_birth в строку, если оно не NULL
            var dateOfBirthStr string
            if dateOfBirth.Valid {
                dateOfBirthStr = dateOfBirth.String
            } else {
                dateOfBirthStr = "" // Если дата рождения NULL, то оставляем пустую строку
            }

            // Создаем карту для каждого пользователя, которую будем добавлять в ответ
            userMap := map[string]interface{}{
                "id":            user.ID,
                "email":         user.Email,
                "first_name":    user.FirstName,
                "last_name":     user.LastName,
                "date_of_birth": dateOfBirthStr,
                "role":          user.Role,
                "password":      password, // Возможно, вы хотите хранить хеш пароля или не включать его в ответ
            }

            users = append(users, userMap)
        }

        // Проверяем на ошибки после итерации
        if err = rows.Err(); err != nil {
            log.Printf("Error iterating over users: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error processing users"})
            return
        }

        // Отправляем список пользователей в формате JSON
        utils.ResponseJSON(w, users)
    }
}
func (c *Controller) UpdateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			FirstName   string `json:"first_name"`
			LastName    string `json:"last_name"`
			DateOfBirth string `json:"date_of_birth"`
			Email       string `json:"email"`
			Password    string `json:"password"` // можно не передавать
			Role        string `json:"role"`
		}

		// Проверка токена
		adminID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Проверка роли суперадмина
		var adminRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", adminID).Scan(&adminRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching role"})
			return
		}
		if adminRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can update users"})
			return
		}

		// Читаем тело запроса
		err = json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Валидация роли
		if requestData.Role != "user" && requestData.Role != "schooladmin" && requestData.Role != "superadmin" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid role"})
			return
		}

		// Получение ID из URL
		vars := mux.Vars(r)
		userIDStr := vars["id"]
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid user ID"})
			return
		}

		// Проверяем, существует ли пользователь
		var existingID int
		err = db.QueryRow("SELECT id FROM users WHERE id = ?", userID).Scan(&existingID)
		if err != nil || existingID == 0 {
			utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found"})
			return
		}

		// Если передан пароль, хешируем
		var query string
		var args []interface{}
		if requestData.Password != "" {
			hashedPassword, err := utils.HashPassword(requestData.Password)
			if err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Password hashing failed"})
				return
			}

			query = `UPDATE users SET first_name = ?, last_name = ?, date_of_birth = ?, email = ?, password = ?, role = ? WHERE id = ?`
			args = []interface{}{
				requestData.FirstName, requestData.LastName, requestData.DateOfBirth,
				requestData.Email, hashedPassword, requestData.Role, userID,
			}
		} else {
			query = `UPDATE users SET first_name = ?, last_name = ?, date_of_birth = ?, email = ?, role = ? WHERE id = ?`
			args = []interface{}{
				requestData.FirstName, requestData.LastName, requestData.DateOfBirth,
				requestData.Email, requestData.Role, userID,
			}
		}

		// Обновляем
		_, err = db.Exec(query, args...)
		if err != nil {
			log.Printf("SQL Exec error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update user"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "User updated successfully"})
	}
}
func (c *Controller) CreateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requestData struct {
			FirstName   string `json:"first_name"`
			LastName    string `json:"last_name"`
			DateOfBirth string `json:"date_of_birth"`
			Email       string `json:"email"`
			Password    string `json:"password"`
			Role        string `json:"role"`
		}

		// Проверка токена
		adminID, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
			return
		}

		// Проверка роли суперадмина
		var adminRole string
		err = db.QueryRow("SELECT role FROM users WHERE id = ?", adminID).Scan(&adminRole)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching role"})
			return
		}

		if adminRole != "superadmin" {
			utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "Only superadmin can create users"})
			return
		}

		// Читаем тело запроса
		err = json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Валидация роли
		if requestData.Role != "user" && requestData.Role != "schooladmin" && requestData.Role != "superadmin" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid role"})
			return
		}

		// Хешируем пароль
		hashedPassword, err := utils.HashPassword(requestData.Password)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Password hashing failed"})
			return
		}

		// Проверка на существование email
		var existingID int
		err = db.QueryRow("SELECT id FROM users WHERE email = ?", requestData.Email).Scan(&existingID)
		if err == nil && existingID > 0 {
			utils.RespondWithError(w, http.StatusConflict, models.Error{Message: "Email already exists"})
			return
		}

		// Вставляем нового пользователя в базу
		query := `INSERT INTO users (first_name, last_name, date_of_birth, email, password, role) 
		          VALUES (?, ?, ?, ?, ?, ?)`

		_, err = db.Exec(query, requestData.FirstName, requestData.LastName, requestData.DateOfBirth,
			requestData.Email, hashedPassword, requestData.Role)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create user"})
			return
		}

		utils.ResponseJSON(w, map[string]string{"message": "User created successfully"})
	}
}
func (c Controller) Logout(w http.ResponseWriter, r *http.Request) {
    // Get token from Authorization header
    authHeader := r.Header.Get("Authorization")
    bearerToken := strings.Split(authHeader, " ")

    if len(bearerToken) == 2 {
        tokenString := bearerToken[1]
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("there was an error")
            }
            return []byte(os.Getenv("SECRET")), nil
        })

        if err != nil || !token.Valid {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid or expired token"})
            return
        }

        // Continue with logging out, e.g., clearing session or token
        http.SetCookie(w, &http.Cookie{
            Name:     "token",
            Value:    "",
            Expires:  time.Unix(0, 0), // Set expiration time
            HttpOnly: true,
        })

        utils.ResponseJSON(w, map[string]string{"message": "Successfully logged out"})
        return
    } else {
        utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid token"})
        return
    }
}
func (c Controller) DeleteAccount(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var errorObject models.Error
        
        // Извлекаем user_id из URL
        vars := mux.Vars(r)
        userID := vars["user_id"]

        // Проверяем, что user_id валидный
        if userID == "" {
            errorObject.Message = "User ID is required"
            utils.RespondWithError(w, http.StatusBadRequest, errorObject)
            return
        }

        // Проверка, существует ли пользователь с таким ID в базе данных
        var existingUserID int
        err := db.QueryRow("SELECT id FROM users WHERE id = ?", userID).Scan(&existingUserID)
        if err != nil {
            if err == sql.ErrNoRows {
                errorObject.Message = "User not found"
                utils.RespondWithError(w, http.StatusNotFound, errorObject)
                return
            }
            errorObject.Message = "Error querying user"
            utils.RespondWithError(w, http.StatusInternalServerError, errorObject)
            return
        }

        // Удаление пользователя из базы данных
        _, err = db.Exec("DELETE FROM users WHERE id = ?", userID)
        if err != nil {
            errorObject.Message = "Failed to delete user"
            utils.RespondWithError(w, http.StatusInternalServerError, errorObject)
            return
        }

        // Ответ о успешном удалении
        utils.ResponseJSON(w, map[string]string{"message": "Account deleted successfully"})
    }
}
func (c Controller) EditProfile(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            FirstName string `json:"first_name"`
            LastName  string `json:"last_name"`
            Age       int    `json:"age"`
        }

        // Decode the body of the request
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body."})
            return
        }

        // Get the user ID from the token (which is validated in the middleware)
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Check if the user is updating their own profile
        var currentUserID int
        query := "SELECT id FROM users WHERE id = ?"
        err = db.QueryRow(query, userID).Scan(&currentUserID)
        if err != nil || currentUserID == 0 {
            utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found."})
            return
        }

        // Update the profile data in the database, but do not allow email to be changed
        updateQuery := `
            UPDATE users 
            SET first_name = ?, last_name = ?, age = ? 
            WHERE id = ?
        `
        _, err = db.Exec(updateQuery, requestData.FirstName, requestData.LastName, requestData.Age, userID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error updating profile."})
            return
        }

        // Respond with a success message
        utils.ResponseJSON(w, map[string]string{"message": "Profile updated successfully."})
    }
}
func (c Controller) UpdatePassword(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            CurrentPassword string `json:"current_password"`
            NewPassword     string `json:"new_password"`
            ConfirmPassword string `json:"confirm_password"`
        }

        // Decode the request body into the password change struct
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body."})
            return
        }

        // Verify the token to get the user ID
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Check if new password matches confirm password
        if requestData.NewPassword != requestData.ConfirmPassword {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "New password and confirm password do not match."})
            return
        }

        // Get the current password from the database
        var hashedPassword string
        query := "SELECT password FROM users WHERE id = ?"
        err = db.QueryRow(query, userID).Scan(&hashedPassword)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error retrieving user password."})
            return
        }

        // Compare current password with the stored hashed password
        err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(requestData.CurrentPassword))
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Incorrect current password."})
            return
        }

        // Hash the new password
        hashedNewPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.NewPassword), bcrypt.DefaultCost)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error hashing the new password."})
            return
        }

        // Update the password in the database
        _, err = db.Exec("UPDATE users SET password = ? WHERE id = ?", hashedNewPassword, userID)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error updating password."})
            return
        }

        // Send success response
        utils.ResponseJSON(w, map[string]string{"message": "Password updated successfully."})
    }
}
func (c Controller) TokenVerifyMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var errorObject models.Error
        authHeader := r.Header.Get("Authorization")
        bearerToken := strings.Split(authHeader, " ")

        if len(bearerToken) == 2 {
            authToken := bearerToken[1]

            token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("There was an error")
                }
                return []byte(os.Getenv("SECRET")), nil
            })

            if err != nil {
                errorObject.Message = err.Error()
                utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
                return
            }

            if token.Valid {
                next.ServeHTTP(w, r)
            } else {
                errorObject.Message = err.Error()
                utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
                return
            }
        } else {
            errorObject.Message = "Invalid Token."
            utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
            return
        }
    })
}
func (c Controller) RefreshToken(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var jwtToken models.JWT
        var error models.Error

        err := json.NewDecoder(r.Body).Decode(&jwtToken)
        if err != nil {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Разбираем refresh token
        token, err := utils.ParseToken(jwtToken.RefreshToken)
        if err != nil {
            error.Message = "Invalid refresh token."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Проверяем, действителен ли токен
        if !token.Valid {
            error.Message = "Refresh token expired."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Извлекаем claims из токена
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            error.Message = "Invalid claims."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Извлекаем user_id из claims
        userID, ok := claims["user_id"].(float64)
        if !ok {
            error.Message = "Invalid user_id in token."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        var user models.User
        query := "SELECT id, email, phone, first_name, last_name, age, status FROM users WHERE id = ?"
        err = db.QueryRow(query, int(userID)).Scan(&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.Age, &user.Role)
        if err != nil {
            error.Message = "User not found."
            utils.RespondWithError(w, http.StatusNotFound, error)
            return
        }

        // Генерация нового access токена
        newAccessToken, err := utils.GenerateToken(user)
        if err != nil {
            error.Message = "Error generating new access token."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Возвращаем новый токен
        jwtToken.Token = newAccessToken
        utils.ResponseJSON(w, jwtToken)
    }
}
func (c Controller) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
    tokenStr := r.FormValue("token")
    if tokenStr == "" {
        // Return the new access token in the response
        http.Error(w, "Token is required", http.StatusBadRequest)
        return
    }

    // Разбор токена
    parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return []byte(os.Getenv("SECRET")), nil
    })

    if err != nil || !parsedToken.Valid {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    claims, ok := parsedToken.Claims.(jwt.MapClaims)
    if !ok || claims["email"] == nil {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }

    // Если токен валиден, вернуть успешный ответ
    email := claims["email"].(string)

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Token valid", "email": email})
}
func (c *Controller) VerifyEmail(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email   string `json:"email"`
            OTPCode string `json:"otp_code"`
        }
        var error models.Error

        // Decode request body
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" {
            error.Message = "Email or OTP code is missing"
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Определяем переменные
        var storedOTP sql.NullString
        var userID int  // Инициализация переменной userID

        // Проверяем наличие OTP для сброса пароля в таблице password_resets
        err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ?", requestData.Email).Scan(&storedOTP)
        if err == nil && storedOTP.Valid {
            // Если OTP найден в таблице сброса пароля
            if storedOTP.String != requestData.OTPCode {
                error.Message = "Invalid OTP code"
                utils.RespondWithError(w, http.StatusUnauthorized, error)
                return
            }

            // Если OTP для сброса пароля действителен, удаляем старый
            _, err = db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
            if err != nil {
                error.Message = "Error processing password reset"
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }

            // Ответ для сброса пароля
            utils.ResponseJSON(w, map[string]string{
                "message": "Password reset code verified successfully",
            })
            return
        }

        // Если OTP не найден в password_resets, проверяем в таблице пользователей
        err = db.QueryRow("SELECT id, otp_code FROM users WHERE email = ?", requestData.Email).Scan(&userID, &storedOTP)
        if err != nil {
            if err == sql.ErrNoRows {
                error.Message = "User not found"
                utils.RespondWithError(w, http.StatusNotFound, error)
            } else {
                log.Printf("Database error: %v", err)
                error.Message = "Server error"
                utils.RespondWithError(w, http.StatusInternalServerError, error)
            }
            return
        }

        // Проверяем, совпадает ли OTP
        if !storedOTP.Valid || storedOTP.String != requestData.OTPCode {
            error.Message = "Invalid OTP code"
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Обновляем статус верификации и очищаем OTP
        _, err = db.Exec("UPDATE users SET verified = true, otp_code = NULL WHERE email = ?", requestData.Email)
        if err != nil {
            log.Printf("Error updating verification status: %v", err)
            error.Message = "Failed to verify email"
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Ответ для успешной верификации
        utils.ResponseJSON(w, map[string]string{
            "message": "Email verified successfully",
        })
    }
}
func sendVerificationEmail(email, verificationLink string) {
	fmt.Println("Verification email sent to", email)
	fmt.Println("Verification Link:", verificationLink)
}
func (c Controller) ResetPassword(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email    string `json:"email"`
            OTPCode  string `json:"otp_code"`
            Password string `json:"password"`
        }
        var error models.Error
        
        // Декодируем JSON-запрос
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }
        
        // Проверяем, существует ли email в базе данных
        var storedOTP string
        err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
        if err != nil {
            error.Message = "Invalid email or OTP expired."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        
        // Проверяем, совпадает ли введенный OTP
        if storedOTP != requestData.OTPCode {
            error.Message = "Invalid OTP code."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        
        // Получаем текущий хешированный пароль пользователя
        var currentHashedPassword string
        err = db.QueryRow("SELECT password FROM users WHERE email = ?", requestData.Email).Scan(&currentHashedPassword)
        if err != nil {
            error.Message = "Failed to retrieve current password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        
        // Проверка нового пароля с текущим хешированным
        err = bcrypt.CompareHashAndPassword([]byte(currentHashedPassword), []byte(requestData.Password))
        if err == nil { // если ошибки нет, значит пароли совпадают
            error.Message = "New password cannot be the same as the current password."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }
        
        // Хешируем новый пароль
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
        if err != nil {
            error.Message = "Failed to hash password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        
        // Обновляем пароль в базе данных
        _, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", string(hashedPassword), requestData.Email)
        if err != nil {
            error.Message = "Failed to update password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        
        // Обновляем статус верификации пользователя на true, чтобы он мог сразу войти
        _, err = db.Exec("UPDATE users SET is_verified = true WHERE email = ?", requestData.Email)
        if err != nil {
            error.Message = "Failed to verify email."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        
        // Удаляем OTP после успешного сброса пароля
        _, err = db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
        if err != nil {
            log.Printf("Error deleting reset token: %v", err)
        }
        
        // Ответ успешный
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "Password reset and email verified successfully"})
    }
}
func (c *Controller) ResendCode(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email string `json:"email"`
            Type  string `json:"type"` // "reset" or "verify"
        }
        var error models.Error

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || (requestData.Type != "reset" && requestData.Type != "verify") {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        log.Printf("Received resend code request for email: %s, type: %s", requestData.Email, requestData.Type)

        // Generate new OTP
        otpCode := fmt.Sprintf("%04d", rand.Intn(10000))
        log.Printf("Generated OTP: %s", otpCode)

        // If request type is "reset" - reset password
        if requestData.Type == "reset" {
            // Delete old OTP entries for password reset
            res, err := db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)
            if err != nil {
                error.Message = "Failed to clear old OTP."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            affected, _ := res.RowsAffected()
            log.Printf("Deleted %d old password reset entries", affected)

            // Insert new OTP into password_resets table with timestamp
            res, err = db.Exec("INSERT INTO password_resets (email, otp_code, created_at) VALUES (?, ?, ?)", 
                              requestData.Email, otpCode, time.Now())
            if err != nil {
                error.Message = "Failed to insert new OTP."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            affected, _ = res.RowsAffected()
            log.Printf("Inserted %d password reset entries", affected)

            // Send OTP for password reset
            utils.SendVerificationEmail(requestData.Email, "", otpCode)
        }

        // If request type is "verify" - verify email
        if requestData.Type == "verify" {
            // Check if user exists
            var exists bool
            err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", requestData.Email).Scan(&exists)
            if err != nil {
                error.Message = "Database error."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            
            if !exists {
                utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "No user found with this email"})
                return
            }

            // Update OTP in users table
            res, err := db.Exec("UPDATE users SET otp_code = ?, created_at = ? WHERE email = ?", 
                              otpCode, time.Now(), requestData.Email)
            if err != nil {
                error.Message = "Failed to update OTP."
                utils.RespondWithError(w, http.StatusInternalServerError, error)
                return
            }
            affected, _ := res.RowsAffected()
            log.Printf("Updated %d user records with new OTP", affected)

            // Send OTP for email verification
            utils.SendVerificationEmail(requestData.Email, "", otpCode)
        }

        // Form response with message and OTP
        response := map[string]interface{}{
            "message":  "OTP resent successfully",
            "otp_code": otpCode, // Send OTP in response
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
    }
}
func ChangeAdminPassword(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.ChangePasswordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var hashedPassword string
		err := db.QueryRow("SELECT Password FROM User WHERE Email = ?", req.Email).Scan(&hashedPassword)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Проверяем старый пароль
		if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.OldPassword)); err != nil {
			http.Error(w, "Incorrect password", http.StatusUnauthorized)
			return
		}

		// Хешируем новый пароль
		hashedNewPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)

		// Обновляем пароль и активируем аккаунт
		_, err = db.Exec("UPDATE User SET Password = ?, is_active = TRUE WHERE Email = ?", string(hashedNewPassword), req.Email)
		if err != nil {
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, "Password updated successfully")
	}
}
func (c Controller) ChangePassword(db *sql.DB, w http.ResponseWriter, r *http.Request) {
    tokenStr := r.FormValue("token")
    newPassword := r.FormValue("new_password")
    if tokenStr == "" || newPassword == "" {
        http.Error(w, "Token and new password are required", http.StatusBadRequest)
        return
    }

    // Разбор токена
    parsedToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return []byte(os.Getenv("SECRET")), nil
    })

    if err != nil || !parsedToken.Valid {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    claims, ok := parsedToken.Claims.(jwt.MapClaims)
    if !ok || claims["email"] == nil {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
    }

    email := claims["email"].(string)

    // Хеширование пароля перед сохранением в базе данных
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "Error hashing password", http.StatusInternalServerError)
        return
    }

    // Обновление пароля в базе данных
    query := "UPDATE users SET password = ? WHERE email = ?"
    _, err = db.Exec(query, hashedPassword, email)
    if err != nil {
        http.Error(w, "Error updating password", http.StatusInternalServerError)
        return
    }

    // Ответ пользователю
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
}
func (c Controller) ForgotPassword(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email string `json:"email"`
        }
        var error models.Error

        // Декодируем запрос
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" {
            error.Message = "Invalid request body or missing email."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем, существует ли email в базе данных
        var userID int
        err = db.QueryRow("SELECT id FROM users WHERE email = ?", requestData.Email).Scan(&userID)
        if err != nil {
            if err == sql.ErrNoRows {
                error.Message = "Email not found."
                utils.RespondWithError(w, http.StatusNotFound, error)
                return
            }
            log.Printf("Error checking email: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Генерируем случайный OTP код
        otpCode := fmt.Sprintf("%04d", rand.Intn(10000))

        // Генерируем уникальный токен для сброса пароля
        token := utils.GenerateResetToken(requestData.Email)

        // Сохраняем OTP и токен в базе данных
        _, err = db.Exec("INSERT INTO password_resets (email, otp_code, reset_token) VALUES (?, ?, ?)", requestData.Email, otpCode, token)
        if err != nil {
            log.Printf("Error saving reset token: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Отправляем email с кодом OTP и ссылкой для сброса пароля
        resetLink := fmt.Sprintf("http://localhost:8000/reset-password?token=%s", token)
        utils.SendEmail(requestData.Email, "Reset your password", fmt.Sprintf("Your OTP: %s\nReset link: %s", otpCode, resetLink))

        // Устанавливаем заголовок ответа, чтобы указать, что это JSON
        w.Header().Set("Content-Type", "application/json")

        // Возвращаем ответ с OTP для проверки на фронтенде
        response := map[string]interface{}{
            "message": "Reset email sent",
            "otp_code": otpCode,  // Отправляем OTP в ответе
        }

        // Отправляем JSON в ответ
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(response)
    }
}
func (c Controller) ConfirmResetPassword(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email    string `json:"email"`
            OTPCode  string `json:"otp_code"`
            Password string `json:"password"`
        }
        var error models.Error

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем код OTP
        var storedOTP string
        err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
        if err != nil || storedOTP != requestData.OTPCode {
            error.Message = "Invalid or expired OTP."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Хешируем новый пароль
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
        if err != nil {
            error.Message = "Failed to hash password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Обновляем пароль в БД
        _, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", hashedPassword, requestData.Email)
        if err != nil {
            error.Message = "Failed to update password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Удаляем OTP после успешного сброса
        db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
    }
}
func (c *Controller) RegisterUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
			return
		}

		// Generate unique verification token
		verificationToken := uuid.New().String()

		// Save user to DB with 'is_verified' as false
		query := `INSERT INTO users (email, password, first_name, last_name, is_verified, verification_token) 
		          VALUES(?, ?, ?, ?, false, ?)`
		_, err := db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, verificationToken)
		if err != nil {
			log.Println("SQL Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create user"})
			return
		}

		// Send verification email
		verificationLink := fmt.Sprintf("http://localhost:8000/verify-email?token=%s", verificationToken)
		go sendVerificationEmail(user.Email, verificationLink)

		utils.ResponseJSON(w, "User registered successfully. Please verify your email.")
	}
}
func GenerateRandomCode() (string, error) {
	code := make([]byte, 6) // генерируем 6-значный код
	_, err := rand.Read(code)
	if err != nil {
		log.Println("Error generating random code:", err)
		return "", err
	}
	return fmt.Sprintf("%x", code[:6]), nil
}
func (c Controller) ChangeUserRole(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            UserID int    `json:"user_id"`
            Role   string `json:"role"`
        }

        // Декодируем тело запроса для получения user_id и новой роли
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.UserID == 0 || requestData.Role == "" {
            log.Printf("Invalid request body: %v", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
            return
        }

        // Проверяем, что роль правильная (например, "user", "schooladmin", "superadmin")
        if requestData.Role != "user" && requestData.Role != "schooladmin" && requestData.Role != "superadmin" {
            log.Printf("Invalid role provided: %s", requestData.Role)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid role"})
            return
        }
        
        // Получаем ID пользователя из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            log.Printf("Unauthorized access: %v", err)
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Unauthorized"})
            return
        }

        // Получаем роль пользователя, который отправил запрос
        var userRole string
        err = db.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&userRole)
        if err != nil {
            log.Printf("Error fetching user role: %v", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Error fetching user role"})
            return
        }

        // Проверяем, что роль пользователя - superadmin
        if userRole != "superadmin" {
            utils.RespondWithError(w, http.StatusForbidden, models.Error{Message: "You do not have permission to change roles"})
            return
        }

        // Обновляем роль в базе данных
        _, err = db.Exec("UPDATE users SET role = ? WHERE id = ?", requestData.Role, requestData.UserID)
        if err != nil {
            log.Printf("Failed to update role for user %d: %v", requestData.UserID, err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update role"})
            return
        }

        // Отправляем успешный ответ
        utils.ResponseJSON(w, map[string]string{"message": "User role updated successfully"})
    }
}
func (c Controller) UploadAvatar(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Чтение файла аватара
        file, _, err := r.FormFile("avatar")
        if err != nil {
            log.Println("Error reading file:", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
            return
        }
        defer file.Close()

        // Генерация уникального имени файла для аватара
        uniqueFileName := fmt.Sprintf("avatar-%d-%d.jpg", userID, time.Now().Unix())

        // Загружаем файл в S3
        photoURL, err := utils.UploadFileToS3(file, uniqueFileName, true) // передаем true для использования второго набора ключей (для аватарки)
        if err != nil {
            log.Println("Error uploading file:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload avatar"})
            return
        }

        // Обновление URL аватара в базе данных
        query := "UPDATE users SET avatar_url = ? WHERE id = ?"
        _, err = db.Exec(query, photoURL, userID)
        if err != nil {
            log.Println("Error updating avatar URL:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update avatar URL"})
            return
        }

        // Ответ с подтверждением
        utils.ResponseJSON(w, map[string]string{"message": "Avatar uploaded successfully", "avatar_url": photoURL})
    }
}
func (c Controller) UpdateAvatar(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Получаем данные о старом аватаре
        var currentAvatarURL sql.NullString
        query := "SELECT avatar_url FROM users WHERE id = ?"
        err = db.QueryRow(query, userID).Scan(&currentAvatarURL)
        if err != nil {
            log.Println("Error fetching current avatar:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch current avatar"})
            return
        }

        // Удаление старого аватара с S3, если он существует
        if currentAvatarURL.Valid && currentAvatarURL.String != "" && currentAvatarURL.String != "https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg" {
            err := utils.DeleteFileFromS3(currentAvatarURL.String)
            if err != nil {
                log.Println("Error deleting old avatar from S3:", err)
                utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete old avatar"})
                return
            }
        }

        // Чтение нового аватара
        file, _, err := r.FormFile("avatar")
        if err != nil {
            log.Println("Error reading file:", err)
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Error reading file"})
            return
        }
        defer file.Close()

        // Генерация уникального имени файла для аватара
        uniqueFileName := fmt.Sprintf("avatar-%d-%d.jpg", userID, time.Now().Unix())

        // Загружаем новый файл в S3 с третьим аргументом `true` для аватарок
        newAvatarURL, err := utils.UploadFileToS3(file, uniqueFileName, true) // Передаем true, чтобы использовать второй набор ключей для аватарок
        if err != nil {
            log.Println("Error uploading new avatar:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to upload new avatar"})
            return
        }

        // Обновление URL аватара в базе данных
        query = "UPDATE users SET avatar_url = ? WHERE id = ?"
        _, err = db.Exec(query, newAvatarURL, userID)
        if err != nil {
            log.Println("Error updating avatar URL:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to update avatar URL"})
            return
        }

        // Ответ с подтверждением
        utils.ResponseJSON(w, map[string]string{"message": "Avatar updated successfully", "avatar_url": newAvatarURL})
    }
}
func (c Controller) DeleteAvatar(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Получаем userID из токена
        userID, err := utils.VerifyToken(r)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
            return
        }

        // Получаем текущий URL аватара из базы данных
        var currentAvatarURL string
        query := "SELECT avatar_url FROM users WHERE id = ?"
        err = db.QueryRow(query, userID).Scan(&currentAvatarURL)
        if err != nil {
            log.Println("Error fetching current avatar:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to fetch current avatar"})
            return
        }

        // Проверяем, что у пользователя есть аватар, который можно удалить
        if currentAvatarURL == "" || currentAvatarURL == "https://your-bucket-name.s3.amazonaws.com/default-avatar.jpg" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "No avatar to delete"})
            return
        }

        // Удаляем старое изображение с S3
        err = utils.DeleteFileFromS3(currentAvatarURL) // Здесь больше не передаем параметр bool
        if err != nil {
            log.Println("Error deleting avatar from S3:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to delete avatar"})
            return
        }

        // Сбрасываем URL аватара на NULL в базе данных
        query = "UPDATE users SET avatar_url = NULL WHERE id = ?"
        _, err = db.Exec(query, userID)
        if err != nil {
            log.Println("Error resetting avatar URL:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to reset avatar URL"})
            return
        }

        // Ответ с подтверждением
        utils.ResponseJSON(w, map[string]string{"message": "Avatar deleted successfully"})
    }
}
func (c Controller) ResetPasswordConfirm(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email    string `json:"email"`
            OTPCode  string `json:"otp_code"`
            Password string `json:"password"`
        }
        var error models.Error

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" || requestData.Password == "" {
            error.Message = "Invalid request body."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем код OTP
        var storedOTP string
        err = db.QueryRow("SELECT otp_code FROM password_resets WHERE email = ? ORDER BY created_at DESC LIMIT 1", requestData.Email).Scan(&storedOTP)
        if err != nil || storedOTP != requestData.OTPCode {
            error.Message = "Invalid or expired OTP."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }

        // Хешируем новый пароль
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestData.Password), bcrypt.DefaultCost)
        if err != nil {
            error.Message = "Failed to hash password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Обновляем пароль в БД
        _, err = db.Exec("UPDATE users SET password = ? WHERE email = ?", hashedPassword, requestData.Email)
        if err != nil {
            error.Message = "Failed to update password."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Удаляем OTP после успешного сброса
        db.Exec("DELETE FROM password_resets WHERE email = ?", requestData.Email)

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "Password reset successfully"})
    }
}
func (c *Controller) VerifyResetCode(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email   string `json:"email"`
            OTPCode string `json:"otp_code"`
        }
        var error models.Error
        
        // Decode request body
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" {
            error.Message = "Email or OTP code is missing"
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }
        
        log.Printf("Verifying reset code for email: %s with OTP: %s", requestData.Email, requestData.OTPCode)
        
        // Check that OTP matches what's stored in the password_resets table
        var storedOTP string
        var createdAt time.Time
        err = db.QueryRow("SELECT otp_code, created_at FROM password_resets WHERE email = ?", 
                        requestData.Email).Scan(&storedOTP, &createdAt)
        if err != nil {
            if err == sql.ErrNoRows {
                log.Printf("Reset code not found for email: %s", requestData.Email)
                error.Message = "Invalid or expired reset code"
                utils.RespondWithError(w, http.StatusNotFound, error)
            } else {
                log.Printf("Database error: %v", err)
                error.Message = "Server error"
                utils.RespondWithError(w, http.StatusInternalServerError, error)
            }
            return
        }
        
        // Check if OTP has expired (15 minutes)
        if time.Now().Sub(createdAt).Minutes() > 15 {
            error.Message = "OTP has expired. Please request a new one."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        
        // Compare OTP codes
        if storedOTP != requestData.OTPCode {
            log.Printf("Reset code mismatch. Stored: %s, Received: %s", storedOTP, requestData.OTPCode)
            error.Message = "Invalid reset code"
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        
        utils.ResponseJSON(w, map[string]string{
            "message": "Reset code verified successfully",
        })
    }
}
func generateRandomPassword(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+"
    rand.Seed(time.Now().UnixNano())
    
    password := make([]byte, length)
    for i := range password {
        password[i] = charset[rand.Intn(len(charset))]
    }
    
    return string(password)
}
func (c *Controller) CalculateAge(user *models.User) (int, error) {
    dob, err := time.Parse("2006-01-02", user.DateOfBirth)
    if err != nil {
        return 0, err
    }
    
    now := time.Now()
    age := now.Year() - dob.Year()
    
    // Adjust age if birthday hasn't occurred yet this year
    if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
        age--
    }
    
    return age, nil
}

