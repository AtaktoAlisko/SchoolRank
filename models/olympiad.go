package models

type Olympiad struct {
	OlympiadID   int    `json:"olympiad_id"`
	SchoolID     int    `json:"school_id"`
	StudentID    int    `json:"student_id"`
	Level        string `json:"level"` // "city", "region", "republic"
	Place        int    `json:"place"` // 1, 2, 3
	Year         int    `json:"year"`
}

type OlympiadRating struct {
	SchoolID     string  `json:"school_id"`
	OlympiadPoints float64 `json:"olympiad_points"`
	MaxPoints      int     `json:"max_points"`
}
