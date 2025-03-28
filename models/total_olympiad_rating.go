package models

import "time"

// TotalOlympiadRating represents the total olympiad rating for a school.
type TotalOlympiadRating struct {
    SchoolID        int       `json:"school_id"`       // ID of the school
    TotalRating     float64   `json:"total_rating"`    // Total olympiad rating
    CreatedAt       time.Time `json:"created_at"`      // Timestamp for when the record was created
    UpdatedAt       time.Time `json:"updated_at"`      // Timestamp for when the record was last updated
}
