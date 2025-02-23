package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"ranking-school/models"
	"ranking-school/utils"
)

// Controller для школ
type SchoolController struct{}
// Создание школы
func (sc SchoolController) CreateSchool(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var school models.School

		// Декодируем JSON из запроса
		if err := json.NewDecoder(r.Body).Decode(&school); err != nil {
			log.Println("JSON Decode Error:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request body"})
			return
		}

		// Проверяем, существует ли `unt_id`
		var untExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM UNT_Score WHERE unt_score_id = ?)", school.UNTID).Scan(&untExists)
		if err != nil || !untExists {
			log.Println("UNT ID not found:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "UNT ID does not exist"})
			return
		}

		// Проверяем, существует ли `id`
		var idExists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM Student WHERE id = ?)", school.ID).Scan(&idExists)
		if err != nil || !idExists {
			log.Println("ID not found:", err)
			utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "ID does not exist"})
			return
		}

		// Вставляем школу в БД
		query := "INSERT INTO School (unt_id, id, avg_unt_score) VALUES (?, ?, ?)"
		result, err := db.Exec(query, school.UNTID, school.ID, school.AvgUNTScore)
		if err != nil {
			log.Println("SQL Insert Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create school"})
			return
		}

		// Получаем ID вставленной записи
		id, _ := result.LastInsertId()
		school.SchoolID = int(id)

		// Возвращаем созданную школу
		utils.ResponseJSON(w, school)
	}
}
// Получение списка всех школ
func (sc SchoolController) GetSchools(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT school_id, unt_id, id, avg_unt_score FROM School")
		if err != nil {
			log.Println("SQL Select Error:", err)
			utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to get schools"})
			return
		}
		defer rows.Close()

		var schools []models.School
		for rows.Next() {
			var school models.School
			if err := rows.Scan(&school.SchoolID, &school.UNTID, &school.ID, &school.AvgUNTScore); err != nil {
				log.Println("SQL Scan Error:", err)
				utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to parse schools"})
				return
			}
			schools = append(schools, school)
		}

		// Возвращаем список школ
		utils.ResponseJSON(w, schools)
	}
}
// Создание нового студента
func (sc SchoolController) CreateStudent(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var student models.Student
        if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        query := `INSERT INTO Student (first_name, last_name, patronymic, iin) VALUES(?, ?, ?, ?)`
        _, err := db.Exec(query, student.FirstName, student.LastName, student.Patronymic, student.IIN)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create student"})
            return
        }

        utils.ResponseJSON(w, "Student created successfully")
    }
}
// Выбор типа теста
func (sc SchoolController) CreateUNTType(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var untType models.UNTType
        if err := json.NewDecoder(r.Body).Decode(&untType); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        query := `INSERT INTO UNT_Type (first_type_id, second_type_id, history_of_kazakhstan, mathematical_literacy, reading_literacy) 
                  VALUES(?, ?, ?, ?, ?)`
        _, err := db.Exec(query, untType.FirstTypeID, untType.SecondTypeID, untType.HistoryKazakh, untType.MathLiteracy, untType.ReadingLiteracy)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to create UNT Type"})
            return
        }

        utils.ResponseJSON(w, "UNT Type created successfully")
    }
}
// Расчет итогового балла
func (sc SchoolController) CalculateScore(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var score models.UNTScore
        if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
            utils.RespondWithError(w, http.StatusBadRequest, models.Error{Message: "Invalid request"})
            return
        }

        // Логика расчета баллов
        totalScore := score.FirstSubjectScore + score.SecondSubjectScore + score.HistoryKazakhstan + score.MathLiteracy + score.ReadingLiteracy
        score.TotalScore = totalScore // сохраняем итоговый балл в поле Score

        query := `INSERT INTO UNT_Score (year, unt_type_id, student_id, first_subject_score, second_subject_score, history_of_kazakhstan, mathematical_literacy, reading_literacy, score) 
                  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`
        _, err := db.Exec(query, score.Year, score.UNTTypeID, score.StudentID, score.FirstSubjectScore, score.SecondSubjectScore, score.HistoryKazakhstan, score.MathLiteracy, score.ReadingLiteracy, score.TotalScore)
        if err != nil {
            log.Println("SQL Error:", err)
            utils.RespondWithError(w, http.StatusInternalServerError, models.Error{Message: "Failed to calculate and save score"})
            return
        }

        utils.ResponseJSON(w, "Score calculated and saved successfully")
    }
}

