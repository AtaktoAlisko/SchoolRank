package models

import "time"

type CityOlympiad struct {
    ID               int       `json:"id"`
    StudentID        int       `json:"student_id"`
    CityOlympiadPlace int       `json:"city_olympiad_place"`
    Score            int       `json:"score"`
    CompetitionDate  time.Time `json:"competition_date"`
}
