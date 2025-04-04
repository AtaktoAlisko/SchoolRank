package models

import "database/sql"

type UNTScore struct {
    ID                    int             `json:"id"`                     
    Year                  int             `json:"year"`                   
    UNTTypeID             int             `json:"unt_type_id"`            
    StudentID             int             `json:"student_id"`             
    TotalScore            int             `json:"total_score"`            
    TotalScoreCreative    int             `json:"total_score_creative"`   
    CreativeExam1         int             `json:"creative_exam1"`         
    CreativeExam2         int             `json:"creative_exam2"`         
    FirstSubjectName      sql.NullString  `json:"first_subject_name"`     // Используем sql.NullString
    FirstSubjectScore     int             `json:"first_subject_score"`    
    SecondSubjectName     sql.NullString  `json:"second_subject_name"`    // Используем sql.NullString
    SecondSubjectScore    int             `json:"second_subject_score"`   
    HistoryKazakhstan     int             `json:"history_of_kazakhstan"`  
    MathematicalLiteracy  int             `json:"mathematical_literacy"`  
    ReadingLiteracy       int             `json:"reading_literacy"`       
    FirstName             string          `json:"first_name"`             
    LastName              string          `json:"last_name"`              
    IIN                   string          `json:"iin"`                    

    AverageRating         float64         `json:"average_rating"`         
    AverageRatingSecond   float64         `json:"average_rating_second"`  
}

