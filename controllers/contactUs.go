package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"ranking-school/models"
	"ranking-school/utils" // Импортируйте utils для обработки ошибок и ответа
)

type ContactUsController struct{}

func (c *ContactUsController) CreateContactRequest(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var contactRequest struct {
            Name    string `json:"name"`
            Email   string `json:"email"`
            Message string `json:"message"`
        }

        // Декодируем тело запроса в структуру
        err := json.NewDecoder(r.Body).Decode(&contactRequest)
        if err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
            return
        }

        // Проверяем, что все необходимые данные присутствуют
        if contactRequest.Name == "" || contactRequest.Email == "" || contactRequest.Message == "" {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Name, email, and message are required"})
            return
        }

        // Сохраняем запрос в базе данных
        query := `INSERT INTO contact_us (name, email, message) VALUES (?, ?, ?)`
        _, err = db.Exec(query, contactRequest.Name, contactRequest.Email, contactRequest.Message)
        if err != nil {
            log.Println("Error inserting contact request:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to save contact request"})
            return
        }

        // Отправляем email админу
        err = sendEmailToAdmin(contactRequest)
        if err != nil {
            log.Println("Error sending email:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to send email"})
            return
        }

        // Ответ с подтверждением
        utils.ResponseJSON(w, map[string]string{"message": "Your request has been received. We will get back to you soon!"})
    }
}
func sendEmailToAdmin(contactRequest struct{
    Name    string `json:"name"`
    Email   string `json:"email"`
    Message string `json:"message"`
}) error {
    // Настройки для отправки email
    from := "mralibekmurat27@gmail.com" // Ваша почта
    password := "bdyi mtae fqub cfcr"     // Пароль приложения, созданный в Google
    to := []string{"mralibekmurat27@gmail.com"} // Почта, на которую вы хотите получать письма
    smtpHost := "smtp.gmail.com" // Для Gmail
    smtpPort := "587" // Порт для Gmail SMTP

    subject := "New Contact Us Request"
    body := fmt.Sprintf("You have received a new contact request:\n\nName: %s\nEmail: %s\nMessage: %s",
        contactRequest.Name, contactRequest.Email, contactRequest.Message)

    // Собираем сообщение
    message := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

    // Отправляем email
    auth := smtp.PlainAuth("", from, password, smtpHost)
    return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
}


