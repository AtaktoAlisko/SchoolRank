package models


type Review struct {
	ID         int     `json:"id"`         
	SchoolID   int     `json:"school_id"`  
	UserID     int     `json:"user_id"`   
	Rating     float64 `json:"rating"`   
	Comment    string  `json:"comment"`   
	CreatedAt  string  `json:"created_at"` 
	SchoolName string  `json:"school_name"`
	Likes      int     `json:"likes"` 
}
type Like struct {
	ID        int    `json:"id"`         
	ReviewID  int    `json:"review_id"`  
	UserID    int    `json:"user_id"`    
	CreatedAt string `json:"created_at"` 
}
