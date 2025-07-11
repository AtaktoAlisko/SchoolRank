package models

//	type SubjectOlympiad struct {
//		ID               int     `json:"subject_olympiad_id"`
//		SubjectName      string  `json:"subject_name"`
//		StartDate        string  `json:"start_date"`
//		Description      string  `json:"description"`
//		SchoolID         int     `json:"school_id"`
//		EndDate          string  `json:"end_date"`
//		Level            string  `json:"level"`
//		Limit            int     `json:"limit_participants"`
//		CreatorID        int     `json:"creator_id"`
//		CreatorFirstName string  `json:"creator_first_name"`
//		CreatorLastName  string  `json:"creator_last_name"`
//		SchoolName       string  `json:"school_name"`
//		Expired          bool    `json:"expired"`
//		PhotoURL         *string `json:"photo_url"`
//		Participants     int     `json:"participants"`
//		Location         string  `json:"location"`
//	}
type SubjectOlympiad struct {
	ID          int    `json:"subject_olympiad_id"`
	SubjectName string `json:"subject_name"`
	StartDate   string `json:"start_date"`
	Description string `json:"description"`
	SchoolID    int    `json:"school_id"`
	EndDate     string `json:"end_date"`
	Level       string `json:"level"`
	Grade       int    `json:"grade"`

	Limit               int     `json:"limit_participants"`
	CreatorID           int     `json:"creator_id"`
	CreatorFirstName    string  `json:"creator_first_name"`
	CreatorLastName     string  `json:"creator_last_name"`
	SchoolName          string  `json:"school_name"`
	Expired             bool    `json:"expired"`
	PhotoURL            *string `json:"photo_url"`
	Participants        int     `json:"participants"`
	Location            string  `json:"location"`
	CurrentParticipants int     `json:"current_participants"` // ← вот это добавь

}
