package models

type SecondType struct {
    ID                        int     `json:"id"`
    HistoryOfKazakhstanCreative *int    `json:"history_of_kazakhstan_creative,omitempty"`
    ReadingLiteracyCreative     *int    `json:"reading_literacy_creative,omitempty"`
    CreativeExam1              *int    `json:"creative_exam1,omitempty"`
    CreativeExam2              *int    `json:"creative_exam2,omitempty"`
    TotalScoreCreative         *int    `json:"total_score_creative,omitempty"` // изменено с TotalScore на TotalScoreCreative
    Type                       string  `json:"type"` // Новый параметр для типа экзамена
}
