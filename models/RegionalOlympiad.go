package models

import "time"

type RegionalOlympiad struct {
    ID               int       `json:"id"`
    StudentID        int       `json:"student_id"`
    RegionalOlympiadPlace int   `json:"regional_olympiad_place"`
    Score            int       `json:"score"`
    CompetitionDate  time.Time `json:"competition_date"`
}
