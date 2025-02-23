package models

type School struct {
	SchoolID    int     `json:"school_id"`
	UNTID       int     `json:"unt_id"`
	ID          int     `json:"id"` 
	AvgUNTScore float64 `json:"avg_unt_score"`
}
