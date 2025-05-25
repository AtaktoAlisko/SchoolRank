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
	OlympiadRegistrationController := controllers.OlympiadRegistrationController{}
	SubjectOlympiadController := controllers.SubjectOlympiadController{}
	eventController := controllers.EventController{}
	EventsParticipantController := controllers.EventsParticipantController{}
	EventsRegistrationController := controllers.EventsRegistrationController{}

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
	router.HandleFunc("/api/users/{user_id}", controller.UpdateUser(db)).Methods("PATCH")
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
	router.HandleFunc("/api/schools/{id}", schoolController.UpdateSchool(db)).Methods("PATCH")
	router.HandleFunc("/api/schools/{id}", schoolController.DeleteSchool(db)).Methods("DELETE")
	router.HandleFunc("/api/schools", schoolController.GetAllSchools(db)).Methods("GET")
	router.HandleFunc("/api/schools/student", schoolController.GetAllStudents(db)).Methods("GET")
	router.HandleFunc("/api/schools/total", schoolController.GetTotalSchools(db)).Methods("GET")
	router.HandleFunc("/api/schools/{id}", schoolController.GetSchoolByID(db)).Methods("GET")

	// =======================
	// Работа с отзывами (Reviews)
	// =======================
	router.HandleFunc("/api/reviews", reviewController.CreateReview(db)).Methods("POST")
	router.HandleFunc("/api/reviews/{id}", reviewController.DeleteReview(db)).Methods("DELETE")
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
	router.HandleFunc("/api/untscore/{student_id}", untScoreController.GetUNTScoreByStudentID(db)).Methods("GET")

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
	router.HandleFunc("/api/schools/count/{school_id}/students", studentController.GetStudentsCountBySchool(db)).Methods("GET")
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
	router.HandleFunc("/api/schools/{school_id}/second-types/average-rating", typeController.GetAverageRatingSecondBySchool(db)).Methods("GET")
	router.HandleFunc("/api/{school_id}/combined-average-rating", untScoreController.GetCombinedAverageRating(db)).Methods("GET")

	// =======================
	// Event registration routes
	// =======================
	router.HandleFunc("/events/register", EventsRegistrationController.RegisterForEvent(db)).Methods("POST")
	router.HandleFunc("/api/event-registrations/{id}", EventsRegistrationController.UpdateEventRegistrationStatus(db)).Methods("PATCH")
	router.HandleFunc("/api/event-registrations", EventsRegistrationController.GetEventRegistrations(db)).Methods("GET")
	router.HandleFunc("/api/event-registrations/{id}", EventsRegistrationController.DeleteEventRegistrationByID(db)).Methods("DELETE")
	router.HandleFunc("/api/my-registrations/{id}", EventsRegistrationController.DeleteMyEventRegistration(db)).Methods("DELETE")
	router.HandleFunc("/api/event-registrations/{id}/approve-or-cancel", EventsRegistrationController.ApproveOrCancelEventRegistration(db)).Methods("PATCH")
	router.HandleFunc("/api/school-ranking", EventsRegistrationController.GetSchoolRanking(db)).Methods("GET")

	// router.HandleFunc("/event-registrations/{event_registration_id}/status", eventController.UpdateEventRegistrationStatus(db)).Methods("PUT")
	// router.HandleFunc("/events/{event_id}/registrations", eventController.GetEventRegistrations(db)).Methods("GET")
	// router.HandleFunc("/students/{student_id}/event-registrations", eventController.GetStudentEventRegistrations(db)).Methods("GET")
	// router.HandleFunc("/my-event-registrations", eventController.GetStudentEventRegistrations(db)).Methods("GET") // для студентов

	// =======================
	// Добавление олимпиад
	// =======================

	router.HandleFunc("/api/subject-olympiads/create/{school_id}", SubjectOlympiadController.CreateSubjectOlympiad(db)).Methods("POST")
	router.HandleFunc("/api/subject-olympiads", SubjectOlympiadController.GetAllSubjectOlympiads(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/by-school/{school_id}", SubjectOlympiadController.GetAllSubjectOlympiadsSchool(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiadsAll/{subject_olympiad_id}", SubjectOlympiadController.GetOlympiadsBySubjectID(db)).Methods("GET")
	router.HandleFunc("/api/api/subject-olympiadsAll", SubjectOlympiadController.GetSubjectOlympiadsByNamePhoto(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/by-subject", SubjectOlympiadController.GetOlympiadsBySubjectName(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/{olympiad_id}", SubjectOlympiadController.GetSubjectOlympiad(db)).Methods("GET")
	router.HandleFunc("/api/subject-olympiads/{id}", SubjectOlympiadController.EditOlympiadsCreated(db)).Methods("PUT")
	router.HandleFunc("/api/subject-olympiads/{id}", SubjectOlympiadController.DeleteSubjectOlympiad(db)).Methods("DELETE")

	// =======================
	// Контактная Events
	// =======================
	router.HandleFunc("/api/events", eventController.AddEvent(db)).Methods("POST")
	router.HandleFunc("/api/events", eventController.GetEvents(db)).Methods("GET")
	router.HandleFunc("/api/events/school/{school_id}", eventController.GetEventsBySchoolAndType(db)).Methods("GET")
	router.HandleFunc("/api/events/{event_id}", eventController.UpdateEvent(db)).Methods("PUT")
	router.HandleFunc("/api/events/{event_id}", eventController.DeleteEvent(db)).Methods("DELETE")
	router.HandleFunc("/api/events/school/{school_id}", eventController.GetEventsBySchoolID(db)).Methods("GET")
	router.HandleFunc("/api/events/category/{category}", eventController.GetEventsByCategory(db)).Methods("GET")
	router.HandleFunc("/api/event/{id}", eventController.GetEventByID(db)).Methods("GET")

	// =======================
	// Контактная информация
	// =======================
	router.HandleFunc("/api/contact", contactController.CreateContactRequest(db)).Methods("POST")

	router.HandleFunc("/api/olympiads/register", OlympiadRegistrationController.RegisterStudent(db)).Methods("POST")
	router.HandleFunc("/api/olympiads/registrations", OlympiadRegistrationController.GetOlympiadRegistrations(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/register/{id}", OlympiadRegistrationController.UpdateRegistrationStatus(db)).Methods("PATCH")
	router.HandleFunc("/api/olympiads/registrations/{id}/place", OlympiadRegistrationController.AssignPlaceToRegistration(db)).Methods("POST")
	router.HandleFunc("/api/olympiads/register/{id}", OlympiadRegistrationController.DeleteRegistration(db)).Methods("DELETE")
	router.HandleFunc("/api/olympiads/total-rating/{school_id}", OlympiadRegistrationController.GetTotalOlympiadRating(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/participants-count", OlympiadRegistrationController.GetOverallOlympiadParticipationCount(db)).Methods("GET")
	router.HandleFunc("/api/olympiads/registrations/by-month", OlympiadRegistrationController.GetRegistrationsByMonth(db)).Methods("GET")

	// =======================
	// Dashboard (superadmin)
	// =======================
	router.HandleFunc("/api/users/count", controller.CountUsers(db)).Methods("GET")
	router.HandleFunc("/api/school-count", schoolController.GetSchoolCount(db)).Methods("GET")
	router.HandleFunc("/api/events/count", eventController.CountEvents(db)).Methods("GET")
	router.HandleFunc("/api/get-top-3-students-by-unt", untScoreController.GetTop3UNTStudents(db)).Methods("GET")
	router.HandleFunc("/api/count-users-by-role", controller.CountUsersByRole(db)).Methods("GET")
	router.HandleFunc("/api/countAllOlympiads", EventsParticipantController.CountOlympiadParticipants(db)).Methods("GET")

	// =======================
	// Dashboard (schooladmin)
	// =======================
	router.HandleFunc("/api/schools/count/{school_id}/students", studentController.GetStudentsCountBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/event-stats", EventsParticipantController.CountOlympiadParticipantsBySchool(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/top-3-unt-students", untScoreController.GetTop3UNTStudentsBySchoolID(db)).Methods("GET")
	router.HandleFunc("/api/schools/{school_id}/reviews/average-rating", reviewController.GetAverageRating(db)).Methods("GET")

	// Включаем CORS
	handler := corsMiddleware(router)

	// Запуск сервера
	log.Println("Сервер запущен на порту 8000")
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
