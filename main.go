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
	contactController := &controllers.ContactUsController{}
	SubjectOlympiadController := controllers.SubjectOlympiadController{}
	olympiadController := &controllers.OlympiadController{}
	eventController := controllers.EventController{}
	EventsParticipantController := controllers.EventsParticipantController{}

	router := mux.NewRouter()

	// =======================
	// Аутентификация и авторизация
	// =======================
	router.HandleFunc("/api/auth/signup", controller.Signup(db)).Methods("POST")
	router.HandleFunc("/api/auth/login", controller.Login(db)).Methods("POST")
	router.HandleFunc("/api/auth/logout", controller.Logout).Methods("POST")
	router.HandleFunc("/api/auth/password/forgot", controller.ForgotPassword(db)).Methods("POST")
	router.HandleFunc("/api/auth/password/reset", controller.ResetPassword(db)).Methods("POST")
	router.HandleFunc("/api/auth/code/resend", controller.ResendCode(db)).Methods("POST")
	router.HandleFunc("/api/auth/password/update", controller.TokenVerifyMiddleware(controller.UpdatePassword(db))).Methods("PUT")
	router.HandleFunc("/api/auth/email/verify", controller.VerifyEmail(db)).Methods("POST")

	// =======================
	// Профиль пользователя и аватар
	// =======================
	router.HandleFunc("/api/users/me", controller.GetMe(db)).Methods("GET")
	router.HandleFunc("/api/users", controller.GetAllUsers(db)).Methods("GET")
	router.HandleFunc("/api/users/{user_id}", controller.UpdateUser(db)).Methods("PUT")
	router.HandleFunc("/api/users/total", controller.GetTotalUsersWithRoleCount(db)).Methods("GET")
	router.HandleFunc("/api/users", controller.CreateUser(db)).Methods("POST")
	router.HandleFunc("/api/users/me", controller.EditProfile(db)).Methods("PUT")
	router.HandleFunc("/api/users/me/avatar", controller.UploadAvatar(db)).Methods("POST")
	router.HandleFunc("/api/users/me/avatar", controller.UpdateAvatar(db)).Methods("PUT")
	router.HandleFunc("/api/users/me/avatar", controller.DeleteAvatar(db)).Methods("DELETE")
	router.HandleFunc("/api/users/delete-account/{user_id}", controller.DeleteAccount(db)).Methods("DELETE")

	router.HandleFunc("/api/users/schooladmins", controller.GetAllSchoolAdmins(db)).Methods("GET")
	router.HandleFunc("/api/users/superadmins", controller.GetAllSuperAdmins(db)).Methods("GET")
	router.HandleFunc("/api/users/schooladmins/no-school", controller.GetSchoolAdminsWithoutSchools(db)).Methods("GET")

	// =======================
	// Административные операции
	// =======================
	router.HandleFunc("/api/admin/change-role", controller.ChangeUserRole(db)).Methods("POST")

	// =======================
	// Проверка токена
	// =======================
	router.HandleFunc("/api/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/api/auth/refresh", controller.RefreshTokenHandler(db))

	// =======================
	// Работа со школами
	// =======================

	//superadmin
	router.HandleFunc("/api/schools", schoolController.CreateSchool(db)).Methods("POST")
	router.HandleFunc("/api/schools/{id}", schoolController.UpdateSchool(db)).Methods("PUT")
	router.HandleFunc("/api/schools/{id}", schoolController.DeleteSchool(db)).Methods("DELETE")
	router.HandleFunc("/api/schools", schoolController.GetAllSchools(db)).Methods("GET")
	router.HandleFunc("/api/schools/student", schoolController.GetAllStudents(db)).Methods("GET")
	router.HandleFunc("/api/schools/total", schoolController.GetTotalSchools(db)).Methods("GET")
	router.HandleFunc("/api/schools/{id}", schoolController.GetSchoolByID(db)).Methods("GET")

	// =======================
	// Работа с отзывами (Reviews)
	// =======================
	router.HandleFunc("/api/reviews", reviewController.CreateReview(db)).Methods("POST")
	router.HandleFunc("/api/reviews", reviewController.GetAllReviews(db)).Methods("GET")
	router.HandleFunc("/api/reviews/{school_id}", reviewController.GetReviewBySchoolID(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/reviews", reviewController.GetReviewsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/reviews/average-rating", reviewController.GetAverageRating(db)).Methods("GET")

	// =======================
	// Работа с UNT Scores (оценками)
	// =======================
	router.HandleFunc("/api/unt/{school_id}", untScoreController.CreateUNT(db)).Methods("POST")
	router.HandleFunc("/api/unt", untScoreController.GetUNTExams(db)).Methods("GET")
	router.HandleFunc("/api/unt/school/{school_id}", untScoreController.GetUNTBySchoolID(db)).Methods("GET")
	router.HandleFunc("/api/unt/{id}", untScoreController.UpdateUNTExam(db)).Methods("PUT")
	router.HandleFunc("/api/unt/{id}", untScoreController.DeleteUNTExam(db)).Methods("DELETE")
	router.HandleFunc("/api/unt_scores/total-score-school", untScoreController.GetTotalScoreForSchool(db)).Methods("GET")
	router.HandleFunc("/api/average-rating/{school_id}", untScoreController.GetAverageRatingBySchool(db)).Methods("GET")
	router.HandleFunc("/api/school/combined-average-rating", untScoreController.GetCombinedAverageRating(db)).Methods("GET")

	// =======================
	// Работа с типами UNT (например, для классификации)
	// =======================

	router.HandleFunc("/api/unt-types", untTypeController.CreateUNTType(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/unt-types", typeController.GetUNTTypesBySchool(db)).Methods("GET")

	// =======================
	// CRUD студентов
	// =======================
	router.HandleFunc("/api/students", studentController.CreateStudent(db)).Methods("POST")
	router.HandleFunc("/api/students", studentController.GetAllStudents(db)).Methods("GET")
	router.HandleFunc("/api/students/{student_id}", studentController.UpdateStudent(db)).Methods("PUT")
	router.HandleFunc("/api/students/{id}", studentController.GetStudentByID(db)).Methods("GET")

	router.HandleFunc("/api/schools/{school_id}/total-students", studentController.GetTotalStudentsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/students/{student_id}", studentController.DeleteStudent(db)).Methods("DELETE")
	router.HandleFunc("/api/schools/{school_id}/students", studentController.GetStudentsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/students/grades", studentController.GetAvailableGradesBySchool(db)).Methods("GET")
	router.HandleFunc("/students/letters/{school_id}/{grade}", studentController.GetAvailableLettersByGrade(db)).Methods("GET")
	// Роут для получения данных о студенте
	router.HandleFunc("/api/schools/{school_id}/students/{student_id}", studentController.GetStudentData(db)).Methods("GET")

	router.HandleFunc("/api/student-filters", studentController.GetStudentFilters(db)).Methods("GET")
	router.HandleFunc("/api/grades", studentController.GetAvailableGrades(db)).Methods("GET")
	router.HandleFunc("/api/letters", studentController.GetAvailableLetters(db)).Methods("GET")
	router.HandleFunc("/api/schools/{schoolId}/grades/{grade}/letters/{letter}/students", studentController.GetFilteredStudents(db)).Methods("GET")

	// =======================
	// Работа с Events Participiant
	// =======================
	router.HandleFunc("/events/participants", EventsParticipantController.AddEventsParticipant(db)).Methods("POST")
	router.HandleFunc("/events/participants/{events_id}", EventsParticipantController.UpdateEventsParticipant(db)).Methods("PUT")
	router.HandleFunc("/events/participants/{events_id}", EventsParticipantController.GetSingleEventsParticipant(db)).Methods("GET")
	router.HandleFunc("/events/participants/{events_id}", EventsParticipantController.DeleteEventsParticipant(db)).Methods("DELETE")
	router.HandleFunc("/events/participants", EventsParticipantController.GetEventsParticipant(db)).Methods("GET")
	router.HandleFunc("/events/participants/school/{school_id}", EventsParticipantController.GetEventsParticipantBySchool(db)).Methods("GET")

	// =======================
	// Работа с Second Types
	// =======================

	router.HandleFunc("/api/second-types", typeController.GetSecondTypes(db)).Methods("GET")
	router.HandleFunc("/api/second-types", typeController.CreateSecondType(db)).Methods("POST")
	router.HandleFunc("/api/schools/{school_id}/second-types", typeController.GetSecondTypesBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/second-types/average-rating", typeController.GetAverageRatingSecondBySchool(db)).Methods("GET")
	router.HandleFunc("/api/{school_id}/combined-average-rating", untScoreController.GetCombinedAverageRating(db)).Methods("GET")

	// =======================
	// Crud для олимпиад
	// =======================
	router.HandleFunc("/api/olympiads", olympiadController.CreateOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/olympiads", olympiadController.GetOlympiad(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/{school_id}", olympiadController.GetOlympiadById(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/{olympiad_id}", olympiadController.DeleteOlympiad(db)).Methods("DELETE")
	router.HandleFunc("/api/olympiads/{olympiad_id}", olympiadController.UpdateOlympiad(db)).Methods("PUT")

	// =======================
	// Итоговый рейтинг по олимппиадам
	// =======================

	router.HandleFunc("/api/subject-olympiads/create/{school_id}", SubjectOlympiadController.CreateSubjectOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/subject-olympiadsAll", SubjectOlympiadController.GetAllSubjectOlympiads(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/{school_id}", SubjectOlympiadController.GetSubjectOlympiads(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/{id}", SubjectOlympiadController.EditOlympiadsCreated(db)).Methods("PUT")
	router.HandleFunc("/api/subject-olympiads/{id}", SubjectOlympiadController.DeleteSubjectOlympiad(db)).Methods("DELETE")

	router.HandleFunc("/api/subject-olympiadsAll/GetAll", SubjectOlympiadController.GetAllSubOlypmiad(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/GetAllNamePicture", SubjectOlympiadController.GetAllSubOlypmiadNamePicture(db)).Methods("GET")
	// =======================
	// Контактная информация
	// =======================
	router.HandleFunc("/api/events", eventController.AddEvent(db)).Methods("POST")
	router.HandleFunc("/api/events", eventController.GetEvents(db)).Methods("GET")
	router.HandleFunc("/api/events/{event_id}", eventController.UpdateEvent(db)).Methods("PUT")
	router.HandleFunc("/api/events/{event_id}", eventController.DeleteEvent(db)).Methods("DELETE")

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
