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
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)
func (c Controller) Signup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		var error models.Error

		// Декодируем JSON-запрос
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			error.Message = "Invalid request body."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, передан email или телефон
		if user.Email == "" && user.Phone == "" {
			error.Message = "Email or phone is required."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, допустима ли роль
		validRoles := map[string]bool{
			"student":        true,
			"parent":         true,
			"teacher":        true,
			"vice_principal": true,
			"director":       true,
		}
		if !validRoles[user.Role] {
			error.Message = "Invalid role. Allowed roles: student, parent, teacher, vice_principal, director."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, это email или телефон
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

		// Проверяем наличие пароля
		if user.Password == "" {
			error.Message = "Password is required."
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// Проверяем, существует ли пользователь в базе
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

		// Хешируем пароль
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}
		user.Password = string(hash)

		// Генерируем 4-значный OTP
		otpCode, err := utils.GenerateOTP()
		if err != nil {
			log.Printf("Error generating OTP: %v", err)
			error.Message = "Failed to generate OTP."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Генерируем токен для подтверждения email
		verificationToken, err := utils.GenerateVerificationToken(user.Email)
		if err != nil {
			log.Printf("Error generating verification token: %v", err)
			error.Message = "Failed to generate verification token."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Записываем пользователя в БД с is_verified = false
		if isEmail {
			query = "INSERT INTO users (email, password, first_name, last_name, age, role, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, false, ?, ?)"
			_, err = db.Exec(query, user.Email, user.Password, user.FirstName, user.LastName, user.Age, user.Role, otpCode, verificationToken)
		} else {
			query = "INSERT INTO users (phone, password, first_name, last_name, age, role, verified, otp_code, verification_token) VALUES (?, ?, ?, ?, ?, ?, true, NULL, ?)"
			_, err = db.Exec(query, user.Phone, user.Password, user.FirstName, user.LastName, user.Age, user.Role, verificationToken)
		}

		if err != nil {
			log.Printf("Error inserting user: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Отправляем email с токеном и OTP
		if isEmail {
			utils.SendVerificationEmail(user.Email, verificationToken, otpCode)
		}

		user.Password = ""  // Не передаем пароль пользователю
		message := "User registered successfully."
		if isEmail {
			message += " Please verify your email with the OTP code."
		}

		utils.ResponseJSON(w, map[string]string{"message": message})
	}
}
func (c Controller) Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		var error models.Error

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
		var isVerified bool

		// Проверка email или телефона
		if user.Email != "" {
			query = "SELECT id, email, phone, password, first_name, last_name, age, role, is_verified FROM users WHERE email = ?"
			identifier = user.Email
		} else {
			query = "SELECT id, email, phone, password, first_name, last_name, age, role, is_verified FROM users WHERE phone = ?"
			identifier = user.Phone
		}

		row := db.QueryRow(query, identifier)
		err = row.Scan(&user.ID, &email, &phone, &hashedPassword, &user.FirstName, &user.LastName, &user.Age, &role, &isVerified)

		if err != nil {
			if err == sql.ErrNoRows {
				error.Message = "User not found."
				utils.RespondWithError(w, http.StatusNotFound, error)
				return
			}
			log.Printf("Error querying user: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		// Проверка подтверждения email
		if !isVerified {
			error.Message = "Please verify your email before logging in."
			utils.RespondWithError(w, http.StatusForbidden, error)
			return
		}

		// Проверка пароля
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
		if err != nil {
			error.Message = "Invalid password."
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// Генерация токенов
		accessToken, err := utils.GenerateToken(user)
		if err != nil {
			log.Printf("Error generating token: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		refreshToken, err := utils.GenerateRefreshToken(user)
		if err != nil {
			log.Printf("Error generating refresh token: %v", err)
			error.Message = "Server error."
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		utils.ResponseJSON(w, map[string]string{
			"token":         accessToken,
			"refresh_token": refreshToken,
		})
	}
}
func (c Controller) TokenVerifyMiddleware(next http.HandlerFunc) http.HandlerFunc{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorObject models.Error
		authHeader := r.Header.Get("Authorization")
		bearerToken := strings.Split(authHeader, " ")

		if len(bearerToken) == 2{
			authToken := bearerToken[1]

			token, error :=jwt.Parse(authToken, func(token*jwt.Token)(interface{},error){
				if _, ok :=token.Method.(*jwt.SigningMethodHMAC); !ok{
					return nil, fmt.Errorf("There was an error")
				}
				return []byte(os.Getenv("SECRET")),nil
			})
			if error != nil{
				errorObject.Message=error.Error()
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}
			if token.Valid{
				next.ServeHTTP(w,r)
			} else{
				errorObject.Message=error.Error()
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}
		} else{
			errorObject.Message="Invailed Token."
			utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
			return
		}
	})

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

        // Проверяем, существует ли email
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
func (c Controller) VerifyResetToken(w http.ResponseWriter, r *http.Request) {
    tokenStr := r.FormValue("token")
    if tokenStr == "" {
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
func (c *Controller) GetMe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверяем токен и получаем userID
		id, err := utils.VerifyToken(r)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: err.Error()})
			return
		}

		// Запрос к базе для получения данных пользователя
		var user models.User
		var email sql.NullString // Используем sql.NullString для обработки NULL
		var phone sql.NullString // Используем sql.NullString для обработки NULL

		err = db.QueryRow("SELECT id, first_name, last_name, email, phone FROM users WHERE id = ?", id).
			Scan(&user.ID, &user.FirstName, &user.LastName, &email, &phone)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, models.Error{Message: "User not found"})
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: err.Error()})
			}
			return
		}

		// Если email не NULL, присваиваем его
		if email.Valid {
			user.Email = email.String
		}

		// Если phone не NULL, присваиваем его
		if phone.Valid {
			user.Phone = phone.String
		}

		utils.ResponseJSON(w, user)
	}
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
        token, err := utils.ParseToken(jwtToken.RefreshToken)
        if err != nil {
            error.Message = "Invalid refresh token."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        if !token.Valid {
            error.Message = "Refresh token expired."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            error.Message = "Invalid claims."
            utils.RespondWithError(w, http.StatusUnauthorized, error)
            return
        }
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
        newAccessToken, err := utils.GenerateToken(user)
        if err != nil {
            error.Message = "Error generating new access token."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }
        jwtToken.Token = newAccessToken
        utils.ResponseJSON(w, jwtToken)
    }
}
func (c Controller) ForgotPassword(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email string `json:"email"`
        }
        var error models.Error

        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" {
            error.Message = "Invalid request body or missing email."
            utils.RespondWithError(w, http.StatusBadRequest, error)
            return
        }

        // Проверяем, существует ли email
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

        // Генерируем 6-значный код OTP
        otpCode := fmt.Sprintf("%06d", rand.Intn(1000000))

        // Генерируем уникальный токен
        token := utils.GenerateResetToken(requestData.Email)

        // Сохраняем OTP и токен в базе
        _, err = db.Exec("INSERT INTO password_resets (email, otp_code, reset_token) VALUES (?, ?, ?)", requestData.Email, otpCode, token)
        if err != nil {
            log.Printf("Error saving reset token: %v", err)
            error.Message = "Server error."
            utils.RespondWithError(w, http.StatusInternalServerError, error)
            return
        }

        // Отправляем email с кодом и ссылкой
        resetLink := fmt.Sprintf("http://localhost:8000/reset-password?token=%s", token)
        utils.SendEmail(requestData.Email, "Reset your password", fmt.Sprintf("Your OTP: %s\nReset link: %s", otpCode, resetLink))

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"message": "Reset email sent"})
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
func (c *Controller) VerifyEmail(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var requestData struct {
            Email    string `json:"email"`
            OTPCode  string `json:"otp_code"`
        }

        // Декодируем тело запроса
        err := json.NewDecoder(r.Body).Decode(&requestData)
        if err != nil || requestData.Email == "" || requestData.OTPCode == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Email or OTP code is missing"})
            return
        }

        // Проверка, что OTP совпадает с тем, что хранится в базе данных
        var storedOTP string
        err = db.QueryRow("SELECT otp_code FROM users WHERE email = ?", requestData.Email).Scan(&storedOTP)
        if err != nil {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid email or OTP"})
            return
        }

        if storedOTP != requestData.OTPCode {
            utils.RespondWithError(w, http.StatusUnauthorized, models.Error{Message: "Invalid OTP code"})
            return
        }

        // Обновляем статус верификации
        _, err = db.Exec("UPDATE users SET is_verified = true WHERE email = ?", requestData.Email)
        if err != nil {
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to verify email"})
            return
        }

        utils.ResponseJSON(w, "Email verified successfully")
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

func sendVerificationEmail(email, verificationLink string) {
	// Implement email sending logic using a service like SendGrid, Mailgun, or SMTP
	// Example: sending email with verification link
	fmt.Println("Verification email sent to", email)
	fmt.Println("Verification Link:", verificationLink)
}








