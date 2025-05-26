package controllers

import (
	"database/sql"
	"errors"
	"math"
)

// 1. Event Participants Percentage
func GetEventParticipantRank(db *sql.DB, schoolID int) (float64, string, error) {
	var schoolName string
	var participantCount int

	err := db.QueryRow(`
		SELECT s.school_name, COUNT(r.event_registration_id)
		FROM Schools s
		LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
		WHERE s.school_id = ? AND r.status IN ('registered', 'accepted', 'completed')
		GROUP BY s.school_name
	`, schoolID).Scan(&schoolName, &participantCount)

	if err != nil {
		return 0, "", err
	}

	var maxParticipants int
	err = db.QueryRow(`
		SELECT COUNT(r.event_registration_id)
		FROM Schools s
		LEFT JOIN EventRegistrations r ON s.school_id = r.school_id
		WHERE r.status IN ('registered', 'accepted', 'completed')
	`).Scan(&maxParticipants)

	if err != nil || maxParticipants == 0 {
		return 0, schoolName, errors.New("could not calculate max participants")
	}

	percentage := (float64(participantCount) / float64(maxParticipants)) * 100
	percentage = math.Round(percentage*100) / 100

	return percentage, schoolName, nil
}

// 2. Event Score
func GetEventScoreRank(db *sql.DB, schoolID int) (float64, error) {
	var eventCount int
	err := db.QueryRow(`
		SELECT COUNT(id) FROM Events WHERE school_id = ?
	`, schoolID).Scan(&eventCount)
	if err != nil {
		return 0, err
	}

	var maxEventCount int
	err = db.QueryRow(`
		SELECT MAX(event_count) FROM (
			SELECT COUNT(id) as event_count FROM Events GROUP BY school_id
		) as counts
	`).Scan(&maxEventCount)
	if err != nil || maxEventCount == 0 {
		return 0, err
	}

	score := (float64(eventCount) / float64(maxEventCount)) * 10
	return math.Round(score*100) / 100, nil
}

// 3. UNT Rank (25% of normalized score)
func GetUNTRank(db *sql.DB, schoolID int) (float64, error) {
	query := `
		SELECT exam_type, AVG(total_score), COUNT(*)
		FROM UNT_Exams
		WHERE school_id = ? AND exam_type IN ('regular', 'creative')
		GROUP BY exam_type`

	rows, err := db.Query(query, schoolID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var regularAvg, creativeAvg float64
	var regularCount, creativeCount int

	for rows.Next() {
		var examType string
		var avg float64
		var count int
		err := rows.Scan(&examType, &avg, &count)
		if err != nil {
			continue
		}
		if examType == "regular" {
			regularAvg = avg
			regularCount = count
		} else if examType == "creative" {
			creativeAvg = avg
			creativeCount = count
		}
	}
	total := regularCount + creativeCount
	if total == 0 {
		return 0, nil
	}

	normRegular := (regularAvg / 140.0) * 100
	normCreative := (creativeAvg / 120.0) * 100

	combined := (normRegular*float64(regularCount) + normCreative*float64(creativeCount)) / float64(total)
	untRank := (25.0 / 100.0) * combined
	return math.Round(untRank*100) / 100, nil
}

// 4. Review Rank (rating * 2)
func GetReviewRank(db *sql.DB, schoolID int) (float64, error) {
	var avgRating float64
	err := db.QueryRow(`SELECT AVG(rating) FROM Reviews WHERE school_id = ?`, schoolID).Scan(&avgRating)
	if err != nil {
		return 0, err
	}
	return math.Round(avgRating*2*100) / 100, nil
}

// 5. Olympiad Rank (sum of weighted levels * 25)
func GetOlympiadRank(db *sql.DB, schoolID int) (float64, string, error) {
	getLevelRating := func(level string, weight float64) float64 {
		var count int
		db.QueryRow(`SELECT COUNT(*) FROM Olympiad_Results WHERE school_id = ? AND level = ?`, schoolID, level).Scan(&count)
		return float64(count) * weight
	}

	city := getLevelRating("city", 0.2)
	region := getLevelRating("region", 0.3)
	republic := getLevelRating("republican", 0.5)
	total := city + region + republic
	olympRank := total * 25

	var schoolName string
	db.QueryRow(`SELECT school_name FROM Schools WHERE school_id = ?`, schoolID).Scan(&schoolName)

	return math.Round(olympRank*100) / 100, schoolName, nil
}
