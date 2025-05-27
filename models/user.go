package models

import (
	"database/sql"
)

type User struct {
	ID          int            `json:"id"`
	Email       string         `json:"email,omitempty"`
	Phone       string         `json:"phone,omitempty"`
	Password    string         `json:"password,omitempty"`
	FirstName   string         `json:"first_name,omitempty"`
	LastName    string         `json:"last_name,omitempty"`
	DateOfBirth string         `json:"date_of_birth,omitempty"`
	Age         int            `json:"age,omitempty"`
	Role        string         `json:"role,omitempty"`
	IsVerified  bool           `json:"is_verified,omitempty"`
	SchoolID    int            `json:"school_id,omitempty"`
	AvatarURL   sql.NullString `json:"avatar_url,omitempty"`
	Login       string         `json:"login"`
}
