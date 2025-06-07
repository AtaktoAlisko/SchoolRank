// package models

// type MyHistoryEvent struct {
// 	Name      string `json:"name"`
// 	Category  string `json:"category"`
// 	StartDate string `json:"start_date"`
// 	EndDate   string `json:"end_date"`
// 	Status    string `json:"status"`
// }

// type MyHistoryOlympiad struct {
// 	Subject   string `json:"subject"`
// 	Level     string `json:"level"`
// 	StartDate string `json:"start_date"`
// 	EndDate   string `json:"end_date"`
// 	Status    string `json:"status"`
// 	Score     int    `json:"score,omitempty"`
// 	Place     int    `json:"place,omitempty"`
// }

package models

type MyHistoryOlympiad struct {
	Subject    string `json:"subject"`
	Level      string `json:"level"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Status     string `json:"status"`
	Score      int    `json:"score,omitempty"`
	Place      int    `json:"place,omitempty"`
	SchoolName string `json:"school_name"`
}

type MyHistoryEvent struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
	Status     string `json:"status"`
	SchoolName string `json:"school_name"`
}
