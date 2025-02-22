package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"ranking-school/controllers"
	"ranking-school/driver"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Fatal("SECRET variable is not set")
	}
	db = driver.ConnectDB()
	defer db.Close()
	controller := controllers.Controller{}
	schoolController := controllers.SchoolController{}
	untScoreController := controllers.UNTScoreController{}
	subjectController := controllers.SubjectController{}
	router := mux.NewRouter()

	router.HandleFunc("/signup", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/login", controller.Login(db)).Methods("POST")
	router.HandleFunc("/getMe", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/reset-password", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/forgot-password", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/verify-email", controller.VerifyEmail(db)).Methods("POST")
	
	router.HandleFunc("/schools", schoolController.GetSchools(db)).Methods("GET")
	router.HandleFunc("/schools/create", schoolController.CreateSchool(db)).Methods("POST")

	router.HandleFunc("/unt_scores", untScoreController.GetUNTScores(db)).Methods("GET")
	router.HandleFunc("/unt_scores/create", untScoreController.CreateUNTScore(db)).Methods("POST")
	
	router.HandleFunc("/subjects/first", subjectController.GetFirstSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/first", subjectController.CreateFirstSubject(db)).Methods("POST")
    router.HandleFunc("/subjects/second", subjectController.GetSecondSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/second", subjectController.CreateSecondSubject(db)).Methods("POST")

	log.Println("Server started on port 8000")
	log.Fatal(http.ListenAndServe(":8000", router))
}
