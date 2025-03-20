package models

type Student struct {
	ID               int    `json:"id"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Patronymic       string `json:"patronymic"`
	DateOfBirth      string `json:"date_of_birth"`
	IIN              string `json:"iin"`
	Grade            int    `json:"grade"`
	SchoolID         int    `json:"school_id"`
	ParentName       string `json:"parent_name,omitempty"`
	ParentPhoneNumber string `json:"parent_phone_number,omitempty"`
}
