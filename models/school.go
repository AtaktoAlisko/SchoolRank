package models

import "database/sql"


type School struct {
	SchoolID         int            `json:"school_id"`          
	UserID           int            `json:"user_id"`           
	SchoolName       string         `json:"school_name"`       
	SchoolAddress    sql.NullString `json:"school_address"`     
	City             string         `json:"city"`              
	AboutSchool      sql.NullString `json:"about_school"`       
	PhotoURL         sql.NullString `json:"photo_url"`         
	SchoolEmail      sql.NullString `json:"school_email"`     
	SchoolPhone      sql.NullString `json:"school_phone"`     
	SchoolAdminLogin sql.NullString `json:"school_admin_login"` 
	CreatedAt        sql.NullString `json:"created_at"`         
	UpdatedAt        sql.NullString `json:"updated_at"`         
	Specializations  []string       `json:"specializations"`
}
