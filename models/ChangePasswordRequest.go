package models

type ChangePasswordRequest struct {
	Email       string `json:"email"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

