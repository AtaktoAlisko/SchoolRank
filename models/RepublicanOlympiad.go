package models

import "time"

type RepublicanOlympiad struct {
    ID                    int       `json:"id"`
    StudentID             int       `json:"student_id"`
    RepublicanOlympiadPlace int     `json:"republican_olympiad_place"`
    Score                 int       `json:"score"`
    CompetitionDate       time.Time `json:"competition_date"`
}
