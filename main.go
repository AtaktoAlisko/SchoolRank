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
		log.Fatal("Ошибка загрузки .env файла")
	}

	secret := os.Getenv("SECRET")
	if secret == "" {
		log.Fatal("Ошибка: переменная SECRET не установлена")
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

	// *** Аутентификация и пароли ***
	router.HandleFunc("/signup", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/login", controller.Login(db)).Methods("POST")
	router.HandleFunc("/getMe", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/reset-password", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/forgot-password", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/verify-email", controller.VerifyEmail(db)).Methods("POST")
	router.HandleFunc("/logout", controller.Logout).Methods("POST")
	router.HandleFunc("/delete-account", controller.DeleteAccount(db)).Methods("DELETE")
	router.HandleFunc("/edit-profile", controller.TokenVerifyMiddleware(controller.EditProfile(db))).Methods("PUT")
	router.HandleFunc("/update-password", controller.TokenVerifyMiddleware(controller.UpdatePassword(db))).Methods("PUT")
	router.HandleFunc("/admin/change-role", controller.ChangeUserRole(db)).Methods("POST")

	// *** Школы ***
	router.HandleFunc("/schools", schoolController.GetSchools(db)).Methods("GET")
    router.HandleFunc("/schools/create", schoolController.CreateSchool(db)).Methods("POST")
	router.HandleFunc("/schools/GetSchoolForDirector", schoolController.GetSchoolForDirector(db)).Methods("GET")
	
	// *** UNT Score ***
	router.HandleFunc("/unt_scores", untScoreController.GetUNTScores(db)).Methods("GET")
	router.HandleFunc("/unt_scores/create", untScoreController.CreateUNTScore(db)).Methods("POST")
	router.HandleFunc("/unt_scores/{student_id}", untScoreController.GetUNTScoreByStudent(db)).Methods("GET")

	// *** Subjects ***
	router.HandleFunc("/subjects/first", subjectController.GetFirstSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/first/create", subjectController.CreateFirstSubject(db)).Methods("POST")
	router.HandleFunc("/subjects/second", subjectController.GetSecondSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/second/create", subjectController.CreateSecondSubject(db)).Methods("POST")

	router.HandleFunc("/unt_types/create", untTypeController.CreateUNTType(db)).Methods("POST")
	router.HandleFunc("/unt_types", untTypeController.GetUNTTypes(db)).Methods("GET")
	
	router.HandleFunc("/students", studentController.GetStudents(db)).Methods("GET")
	router.HandleFunc("/students/create", studentController.CreateStudent(db)).Methods("POST")
	router.HandleFunc("/students/{student_id}/unt_results", studentController.GetUNTResults(db)).Methods("GET")

	// In your routes setup file
	router.HandleFunc("/students/untResults", studentController.CreateUNTResults(db)).Methods("POST")
	// e.g. GET /unt_scores

  
	// *** First Type ***
	router.HandleFunc("/first_types", typeController.GetFirstTypes(db)).Methods("GET")
	router.HandleFunc("/first_types/create", typeController.CreateFirstType(db)).Methods("POST") 
	// *** Second Type ***
	router.HandleFunc("/second_types", typeController.GetSecondTypes(db)).Methods("GET")
	router.HandleFunc("/second_types/create", typeController.CreateSecondType(db)).Methods("POST")

	router.HandleFunc("/students/untResults/create", studentController.CreateUNTResults(db)).Methods("POST")

    // Get results by student_id
    router.HandleFunc("/students/untResults", studentController.GetUNTResults(db)).Methods("GET")


	// Включаем CORS
	handler := corsMiddleware(router)

	// Запуск сервера
	log.Println("Сервер запущен на порту 8000")
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", handler))
}


func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
