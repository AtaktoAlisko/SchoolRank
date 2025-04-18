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
	db := driver.ConnectDB()
	defer db.Close()
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
	typeController := controllers.TypeController{}
	untTypeController := controllers.UNTTypeController{}
	studentController := controllers.StudentController{}
	reviewController := controllers.ReviewController{}
	cityOlympiadController := controllers.CityOlympiadController{}
	regionalOlympiadController := controllers.RegionalOlympiadController{}
	republicanOlympiadController := controllers.RepublicanOlympiadController{}
	TotalOlympiadRatingController := controllers.TotalOlympiadRatingController{}
	contactController := &controllers.ContactUsController{}


	router := mux.NewRouter()


	// =======================
    // Аутентификация и авторизация
    // =======================
    router.HandleFunc("/api/auth/signup", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/api/auth/login", controller.Login(db)).Methods("POST")
	router.HandleFunc("/api/auth/me", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/api/auth/logout", controller.Logout).Methods("POST")
	router.HandleFunc("/api/auth/password/forgot", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/api/auth/password/reset", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/api/auth/code/resend", controller.ResendCode(db)).Methods("POST")
	router.HandleFunc("/api/auth/password/update", controller.TokenVerifyMiddleware(controller.UpdatePassword(db))).Methods("PUT")
	router.HandleFunc("/api/auth/email/verify", controller.VerifyEmail(db)).Methods("POST")


	// =======================
    // Профиль пользователя и аватар
    // =======================
	router.HandleFunc("/api/users/me", controller.TokenVerifyMiddleware(controller.EditProfile(db))).Methods("PUT")
	router.HandleFunc("/api/users/me/avatar", controller.UploadAvatar(db)).Methods("POST")
	router.HandleFunc("/api/users/me/avatar", controller.UpdateAvatar(db)).Methods("PUT")
	router.HandleFunc("/api/users/me/avatar", controller.DeleteAvatar(db)).Methods("DELETE")
	router.HandleFunc("/api/users/delete-account/{user_id}", controller.DeleteAccount(db)).Methods("DELETE")


	// =======================
    // Административные операции
    // =======================
	router.HandleFunc("/api/admin/change-role", controller.ChangeUserRole(db)).Methods("POST")


	// =======================
    // Проверка токена
    // =======================
	router.HandleFunc("/api/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")


	// =======================
    // Работа со школами
    // =======================
	router.HandleFunc("/api/schools", schoolController.GetSchools(db)).Methods("GET")
	router.HandleFunc("/api/schools", schoolController.CreateSchool(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/director", schoolController.GetSchoolForDirector(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}", schoolController.DeleteSchool(db)).Methods("DELETE")


	// =======================
    // Работа с отзывами (Reviews)
    // =======================
	router.HandleFunc("/api/reviews", reviewController.CreateReview(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/reviews", reviewController.GetReviewsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/reviews/average-rating", reviewController.GetAverageRating(db)).Methods("GET")


	// =======================
    // Работа с UNT Scores (оценками)
    // =======================
	router.HandleFunc("/api/unt_scores/total-score-school", untScoreController.GetTotalScoreForSchool(db)).Methods("GET")
    router.HandleFunc("/api/average-rating/{school_id}", untScoreController.GetAverageRatingBySchool(db)).Methods("GET")
	router.HandleFunc("/api/school/combined-average-rating", untScoreController.GetCombinedAverageRating(db)).Methods("GET")


	// =======================
    // Работа с типами UNT (например, для классификации)
    // =======================
	router.HandleFunc("/api/unt-types", untTypeController.CreateUNTType(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/unt-types", typeController.GetUNTTypesBySchool(db)).Methods("GET")


	// =======================
    // Работа со студентами
    // =======================
    router.HandleFunc("/api/students", studentController.CreateStudent(db)).Methods("POST")
    router.HandleFunc("/api/students", studentController.GetStudents(db)).Methods("GET")
	router.HandleFunc("/api/students/{student_id}", studentController.UpdateStudent(db)).Methods("PUT") //Нужно испроавить id гылп
	router.HandleFunc("/api/students/{student_id}", studentController.DeleteStudent(db)).Methods("DELETE") //Тоже нужно исправить
	router.HandleFunc("/api/schools/{school_id}/students", studentController.GetStudentsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/students/{grade}", studentController.GetStudentsBySchoolAndGrade(db)).Methods("GET")
	router.HandleFunc("/api/students/grade/{grade}/letter/{letter}", studentController.GetStudentsByGradeAndLetter(db)).Methods("GET")
	// Роут для получения данных о студенте
    router.HandleFunc("/api/schools/{school_id}/students/{student_id}", studentController.GetStudentData(db)).Methods("GET")



	// =======================
    // Работа с First Types
    // =======================
	router.HandleFunc("/api/first-types", typeController.CreateFirstType(db)).Methods("POST")
	router.HandleFunc("/api/first_types", typeController.GetFirstTypes(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/first-types", typeController.GetFirstTypesBySchool(db)).Methods("GET")
    router.HandleFunc("/api/schools/{school_id}/first-types/average-rating", typeController.GetAverageRatingBySchool(db)).Methods("GET")
	

	// =======================
    // Работа с Second Types
    // =======================
	router.HandleFunc("/api/second-types", typeController.GetSecondTypes(db)).Methods("GET")
	router.HandleFunc("/api/second-types", typeController.CreateSecondType(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/second-types", typeController.GetSecondTypesBySchool(db)).Methods("GET")
    router.HandleFunc("/api/schools/{school_id}/second-types/average-rating", typeController.GetAverageRatingSecondBySchool(db)).Methods("GET")
    router.HandleFunc("/api/{school_id}/combined-average-rating", untScoreController.GetCombinedAverageRating(db)).Methods("GET")


	// =======================
    // Олимпиады: городские, областные, республиканские
    // =======================
	router.HandleFunc("/api/city-olympiads", cityOlympiadController.CreateCityOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/city_olympiad", cityOlympiadController.GetCityOlympiad(db)).Methods("GET")
	router.HandleFunc("/api/city-olympiads/average-rating", cityOlympiadController.GetAverageCityOlympiadScore(db)).Methods("GET")
    router.HandleFunc("/api/city-olympiads/{olympiad_id}", cityOlympiadController.DeleteCityOlympiad(db)).Methods("DELETE") //Нужно исправить по id

	router.HandleFunc("/api/regional-olympiads", regionalOlympiadController.CreateRegionalOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/regional_olympiad", regionalOlympiadController.GetRegionalOlympiad(db)).Methods("GET")
	router.HandleFunc("/api/regional-olympiads/average-rating", regionalOlympiadController.GetAverageRegionalOlympiadScore(db)).Methods("GET")
	router.HandleFunc("//api/regional-olympiads/{olympiad_id}", regionalOlympiadController.DeleteRegionalOlympiad(db)).Methods("DELETE") //Нужно исправить по id

	router.HandleFunc("/api/republican-olympiads", republicanOlympiadController.CreateRepublicanOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/republican_olympiad", republicanOlympiadController.GetRepublicanOlympiad(db)).Methods("GET")
	router.HandleFunc("/api/republican-olympiads/average-rating", republicanOlympiadController.GetAverageRepublicanOlympiadScore(db)).Methods("GET")
	router.HandleFunc("/api/republican-olympiads/{olympiad_id}", republicanOlympiadController.DeleteRepublicanOlympiad(db)).Methods("DELETE") //Нужно исправить по id


    // =======================
	// Итоговый рейтинг по олимппиадам
	// =======================
	router.HandleFunc("/api/olympiads/total-rating", TotalOlympiadRatingController.GetTotalOlympiadRating(db)).Methods("GET")
	

    // =======================
    // Контактная информация
    // =======================
	router.HandleFunc("/api/contact", contactController.CreateContactRequest(db)).Methods("POST")

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