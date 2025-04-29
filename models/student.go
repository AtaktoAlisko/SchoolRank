package models

type Student struct {
	ID          int    `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Patronymic  string `json:"patronymic"`
	DateOfBirth string `json:"date_of_birth"`
	IIN         string `json:"iin"`
	Grade       int    `json:"grade"`
	SchoolID    int    `json:"school_id"`
	Letter      string `json:"letter"`
	Gender      string `json:"gender"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Login       string `json:"login"`
	Password    string `json:"password"`
	Role        string `json:"role"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}
