package models

type JWT struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}
