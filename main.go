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
	typeController := controllers.TypeController{}
	untTypeController := controllers.UNTTypeController{}
	studentController := controllers.StudentController{}

	router := mux.NewRouter()

	// Маршруты для аутентификации и управления паролями
	router.HandleFunc("/signup", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/login", controller.Login(db)).Methods("POST")
	router.HandleFunc("/getMe", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/reset-password", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/forgot-password", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/verify-email", controller.VerifyEmail(db)).Methods("POST")
	router.HandleFunc("/verify-email", controller.VerifyEmail(db)).Methods("GET")


	// Маршруты для работы со школами
	router.HandleFunc("/schools", schoolController.GetSchools(db)).Methods("GET")
	router.HandleFunc("/schools/create", schoolController.CreateSchool(db)).Methods("POST")

	// Маршруты для UNT Score
	router.HandleFunc("/unt_scores", untScoreController.GetUNTScores(db)).Methods("GET")
	router.HandleFunc("/unt_scores/create", untScoreController.CreateUNTScore(db)).Methods("POST")

	// Маршруты для предметов
	router.HandleFunc("/subjects/first", subjectController.GetFirstSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/first", subjectController.CreateFirstSubject(db)).Methods("POST")
	router.HandleFunc("/subjects/second", subjectController.GetSecondSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/second", subjectController.CreateSecondSubject(db)).Methods("POST")

	// Маршруты для типов предметов
	router.HandleFunc("/subjects/firstType", typeController.CreateFirstType(db)).Methods("POST")
	router.HandleFunc("/subjects/firstType", typeController.GetFirstTypes(db)).Methods("GET")
	router.HandleFunc("/subjects/secondType", typeController.CreateSecondType(db)).Methods("POST")
	router.HandleFunc("/subjects/secondType", typeController.GetSecondTypes(db)).Methods("GET")

	// Маршруты для UNT Types
	router.HandleFunc("/untTypes", untTypeController.CreateUNTType(db)).Methods("POST")
	router.HandleFunc("/untTypes", untTypeController.GetUNTTypes(db)).Methods("GET")

	// Маршруты для студентов
	router.HandleFunc("/students", studentController.GetStudents(db)).Methods("GET")
	router.HandleFunc("/students/create", studentController.CreateStudents(db)).Methods("POST")

	// Расчёт оценки
	router.HandleFunc("/score/calculate", schoolController.CalculateScore(db)).Methods("GET")

	// Оборачиваем роутер в CORS middleware
	handler := corsMiddleware(router)

	log.Println("Server started on port 8000")
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", handler))
}

// CORS Middleware Function
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем все домены
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Обработка preflight запроса
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
