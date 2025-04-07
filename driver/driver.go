package driver

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func ConnectDB() *sql.DB {
	var err error
	var dbURL string

	// Если переменная окружения JAWSDB_URL существует (на Heroku)
	if os.Getenv("JAWSDB_URL") != "" {
		dbURL = os.Getenv("JAWSDB_URL") // Строка подключения для Heroku
	} else {
		// Если приложение работает локально, используем локальное подключение
		// Переменные DB_HOST, DB_USER, DB_PASSWORD и DB_NAME должны быть определены в .env файле для локальной разработки
		dbURL = os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASSWORD") + "@tcp(" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + ")/" + os.Getenv("DB_NAME")
	}

	// Подключаемся к базе данных
	db, err = sql.Open("mysql", dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Проверка на успешное подключение
	if err := db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}
	return db
}
