package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"ranking-school/models"
	"ranking-school/utils" 
)

type ContactUsController struct{}

func (c *ContactUsController) CreateContactRequest(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var contactRequest struct {
			FullName string `json:"full_name"`
			Email    string `json:"email"`
			Message  string `json:"message"`
		}

	
		err := json.NewDecoder(r.Body).Decode(&contactRequest)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		
		if contactRequest.FullName == "" || contactRequest.Email == "" || contactRequest.Message == "" {
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Full name, email, and message are required"})
			return
		}

		
		query := `INSERT INTO contact_us (full_name, email, message) VALUES (?, ?, ?)`
		_, err = db.Exec(query, contactRequest.FullName, contactRequest.Email, contactRequest.Message)
		if err != nil {
			log.Println("Error inserting contact request:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to save contact request"})
			return
		}


		err = sendEmailToAdmin(contactRequest)
		if err != nil {
			log.Println("Error sending email:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to send email"})
			return
		}

		
		utils.ResponseJSON(w, map[string]string{"message": "Your request has been received. We will get back to you soon!"})
	}
}

func sendEmailToAdmin(contactRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}) error {
	
	from := "mralibekmurat27@gmail.com"        
	password := "bdyi mtae fqub cfcr"          
	to := []string{"mralibekmurat27@gmail.com"} 
	smtpHost := "smtp.gmail.com"                
	smtpPort := "587"                       

	subject := "New Contact Us Request"
	body := fmt.Sprintf("You have received a new contact request:\n\nName: %s\nEmail: %s\nMessage: %s",
		contactRequest.FullName, contactRequest.Email, contactRequest.Message)

	
	message := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	
	auth := smtp.PlainAuth("", from, password, smtpHost)
	return smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
}
