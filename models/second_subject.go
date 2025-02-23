package models

type SecondSubject struct {
    ID             int     `json:"id"`
    Subject        string  `json:"subject"`
    Score          float64 `json:"score"`
    FirstSubjectID int     `json:"first_subject_id"` 
}
