package models

type SecondType struct {
	ID                          int    `json:"id"`
	HistoryOfKazakhstanCreative *int   `json:"history_of_kazakhstan_creative,omitempty"`
	ReadingLiteracyCreative     *int   `json:"reading_literacy_creative,omitempty"`
	CreativeExam1               *int   `json:"creative_exam1,omitempty"`
	CreativeExam2               *int   `json:"creative_exam2,omitempty"`
	TotalScoreCreative          *int   `json:"total_score_creative,omitempty"`
	Type                        string `json:"type"`
	StudentID                   int    `json:"student_id"` // Теперь обычный int
	StudentTypeID               int    `json:"student_type_id"`
	DocumentURL                 string `json:"document_url,omitempty"`
}
