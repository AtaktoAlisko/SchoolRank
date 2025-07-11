package models

type UNTType struct {
    UNTTypeID                   int     `json:"unt_type_id"`
    Type                        string  `json:"type"`
    FirstTypeID                 *int    `json:"first_type_id,omitempty"`
    FirstSubjectID              *int    `json:"first_subject_id,omitempty"`
    FirstSubjectName            *string `json:"first_subject_name,omitempty"`
    FirstSubjectScore           *int    `json:"first_subject_score,omitempty"`
    SecondSubjectID             *int    `json:"second_subject_id,omitempty"`
    SecondSubjectName           *string `json:"second_subject_name,omitempty"`
    SecondSubjectScore          *int    `json:"second_subject_score,omitempty"`
    HistoryKazakhstan           *int    `json:"history_of_kazakhstan,omitempty"`
    MathematicalLiteracy        *int    `json:"mathematical_literacy,omitempty"`
    ReadingLiteracy             *int    `json:"reading_literacy,omitempty"`
    SecondTypeID                *int    `json:"second_type_id,omitempty"`
    SecondTypeHistoryKazakhstan *int    `json:"second_type_history_kazakhstan,omitempty"`
    SecondTypeReadingLiteracy   *int    `json:"second_type_reading_literacy,omitempty"`
    CreativeExam1               *int    `json:"creative_exam1,omitempty"`
    CreativeExam2               *int    `json:"creative_exam2,omitempty"`
    TotalScore                  *int    `json:"total_score,omitempty"`
    TotalScoreCreative          *int    `json:"total_score_creative,omitempty"`
}
