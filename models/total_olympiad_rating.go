package models

import "time"


type TotalOlympiadRating struct {
    SchoolID        int       `json:"school_id"`       
    TotalRating     float64   `json:"total_rating"`    
    CreatedAt       time.Time `json:"created_at"`     
    UpdatedAt       time.Time `json:"updated_at"`      
}
