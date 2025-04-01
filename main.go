package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"ranking-school/controllers"
	"ranking-school/driver"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	// Загрузка переменных из .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	// Получаем переменные из окружения
	awsAccessKeyID := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")

	// Создаем сессию с AWS
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, ""),
	})
	if err != nil {
		log.Fatal("Не удалось создать сессию:", err)
	}

	// Создаем клиент для S3
	svc := s3.New(sess)

	// Пример получения списка бакетов
	result, err := svc.ListBuckets(nil)
	if err != nil {
		log.Fatal("Не удалось получить список бакетов:", err)
	}

	// Выводим список бакетов
	fmt.Println("Бакеты:")
	for _, b := range result.Buckets {
		fmt.Printf("* %s создан в %s\n", *b.Name, b.CreationDate)
	}

	// Подключение к базе данных
	db = driver.ConnectDB()
	defer db.Close()

	
	controller := controllers.Controller{}
	schoolController := controllers.SchoolController{}
	untScoreController := controllers.UNTScoreController{}
	subjectController := controllers.SubjectController{}
	typeController := controllers.TypeController{}
	untTypeController := controllers.UNTTypeController{}
	studentController := controllers.StudentController{}
	reviewController := controllers.ReviewController{}
	cityOlympiadController := controllers.CityOlympiadController{}
	regionalOlympiadController := controllers.RegionalOlympiadController{}
	republicanOlympiadController := controllers.RepublicanOlympiadController{}
	TotalOlympiadRatingController := controllers.TotalOlympiadRatingController{}


	
	router := mux.NewRouter()

	// *** Аутентификация и пароли ***
    router.HandleFunc("/api/user/sign_up", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/api/user/sign_in", controller.Login(db)).Methods("POST")
	router.HandleFunc("/api/user/getMe", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/api/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/api/user/reset-password", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/api/user/forgot-password", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/api/user/verify-email", controller.VerifyEmail(db)).Methods("POST")
	router.HandleFunc("/api/user/logout", controller.Logout).Methods("POST")
	router.HandleFunc("/api/user/delete-account", controller.DeleteAccount(db)).Methods("DELETE")
	router.HandleFunc("/api/user/edit-profile", controller.TokenVerifyMiddleware(controller.EditProfile(db))).Methods("PUT")
	router.HandleFunc("/api/user/update-password", controller.TokenVerifyMiddleware(controller.UpdatePassword(db))).Methods("PUT")
	router.HandleFunc("/api/admin/change-role", controller.ChangeUserRole(db)).Methods("POST")
	router.HandleFunc("/api/user/upload-avatar", controller.UploadAvatar(db)).Methods("POST")
	router.HandleFunc("/api/user/update-avatar", controller.UpdateAvatar(db)).Methods("PUT")
	router.HandleFunc("/api/user/delete-avatar", controller.DeleteAvatar(db)).Methods("DELETE")
	

	// *** Школы ***
	router.HandleFunc("/schools", schoolController.GetSchools(db)).Methods("GET")
    router.HandleFunc("/schools/create", schoolController.CreateSchool(db)).Methods("POST")
	router.HandleFunc("/schools/GetSchoolForDirector", schoolController.GetSchoolForDirector(db)).Methods("GET")
	// Добавьте роут для удаления школы
    router.HandleFunc("/schools/delete", schoolController.DeleteSchool(db)).Methods("DELETE")
	
	// *** UNT Score ***
	// router.HandleFunc("/unt_scores", untScoreController.GetUNTScores(db)).Methods("GET")
	router.HandleFunc("/unt_scores/create", untScoreController.CreateUNTScore(db)).Methods("POST")
	// router.HandleFunc("/unt_scores/average", untScoreController.GetAverageScoreForSchool(db)).Methods("GET")
	// router.HandleFunc("/unt_scores/GetTotalScoreForSchool", untScoreController.GetTotalScoreForSchool(db)).Methods("GET")

	router.HandleFunc("/reviews/create", reviewController.CreateReview(db)).Methods("POST")
	router.HandleFunc("/reviews/school/{school_id}", reviewController.GetReviewsBySchool(db)).Methods("GET")
	router.HandleFunc("/reviews/average-rating/{school_id}", reviewController.GetAverageRating(db)).Methods("GET")
	
	// *** Subjects ***
	router.HandleFunc("/subjects/first", subjectController.GetFirstSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/first/create", subjectController.CreateFirstSubject(db)).Methods("POST")
	router.HandleFunc("/subjects/second", subjectController.GetSecondSubjects(db)).Methods("GET")
	router.HandleFunc("/subjects/second/create", subjectController.CreateSecondSubject(db)).Methods("POST")

	router.HandleFunc("/unt_types/create", untTypeController.CreateUNTType(db)).Methods("POST")
	router.HandleFunc("/unt_types", untTypeController.GetUNTTypes(db)).Methods("GET")
	
	// Добавьте маршруты перед запуском сервера
    router.HandleFunc("/students/create", studentController.CreateStudent(db)).Methods("POST")
    router.HandleFunc("/students", studentController.GetStudents(db)).Methods("GET")
	router.HandleFunc("/students/update", studentController.UpdateStudent(db)).Methods("PUT")
	router.HandleFunc("/students/delete", studentController.DeleteStudent(db)).Methods("DELETE")

	// *** First Type ***
	router.HandleFunc("/first_types", typeController.GetFirstTypes(db)).Methods("GET")
	router.HandleFunc("/first_types/create", typeController.CreateFirstType(db)).Methods("POST") 
	router.HandleFunc("/unt_scores/average", typeController.GetAverageUNTScore(db)).Methods("GET")

	// *** Second Type ***
	router.HandleFunc("/second_types", typeController.GetSecondTypes(db)).Methods("GET")
	router.HandleFunc("/second_types/create", typeController.CreateSecondType(db)).Methods("POST")

	// Роуты для городской олимпиады
	router.HandleFunc("/city_olympiad/create", cityOlympiadController.CreateCityOlympiad(db)).Methods("POST")
	router.HandleFunc("/city_olympiad", cityOlympiadController.GetCityOlympiad(db)).Methods("GET")
	router.HandleFunc("/city_olympiad/GetAverageCityOlympiadScore", cityOlympiadController.GetAverageCityOlympiadScore(db)).Methods("GET")
    router.HandleFunc("/city_olympiad/delete", cityOlympiadController.DeleteCityOlympiad(db)).Methods("DELETE")

	// Роуты для областной олимпиады
	router.HandleFunc("/regional_olympiad/create", regionalOlympiadController.CreateRegionalOlympiad(db)).Methods("POST")
	router.HandleFunc("/regional_olympiad", regionalOlympiadController.GetRegionalOlympiad(db)).Methods("GET")
	router.HandleFunc("/regional_olympiad/GetAverageRegionalOlympiadScore", regionalOlympiadController.GetAverageRegionalOlympiadScore(db)).Methods("GET")
	router.HandleFunc("/regional_olympiad/delete", regionalOlympiadController.DeleteRegionalOlympiad(db)).Methods("DELETE")

    // Роуты для республиканской олимпиады
	router.HandleFunc("/republican_olympiad/create", republicanOlympiadController.CreateRepublicanOlympiad(db)).Methods("POST")
	router.HandleFunc("/republican_olympiad", republicanOlympiadController.GetRepublicanOlympiad(db)).Methods("GET")
	router.HandleFunc("/republican_olympiad/delete", republicanOlympiadController.DeleteRepublicanOlympiad(db)).Methods("DELETE")
	router.HandleFunc("/regional_olympiad/GetAverageRepublicanOlympiadScore", republicanOlympiadController.GetAverageRepublicanOlympiadScore(db)).Methods("GET")
	router.HandleFunc("/olympiad/total-rating", TotalOlympiadRatingController.GetTotalOlympiadRating(db)).Methods("GET")





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
