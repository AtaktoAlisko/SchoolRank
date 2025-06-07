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
	Subject     string `json:"subject"`
	Level       string `json:"level"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Status      string `json:"status"`
	Score       int    `json:"score,omitempty"`
	Place       int    `json:"place,omitempty"`
	SchoolName  string `json:"school_name"`  // из school_name
	DocumentURL string `json:"document_url"` // из document
}

type MyHistoryEvent struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Status      string `json:"status"`
	SchoolName  string `json:"school_name"`
	Subject     string `json:"subject"` // из olympiad_name
	Level       string `json:"level"`
	Score       int    `json:"score,omitempty"`
	Place       int    `json:"place,omitempty"`
	DocumentURL string `json:"document_url"` // из document_url
}
