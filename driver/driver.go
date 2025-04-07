package driver

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// ConnectDB устанавливает подключение к базе данных
func ConnectDB() *sql.DB {
	var err error
	var dbURL string

	// Проверяем, есть ли переменная окружения JAWSDB_URL (для Heroku)
	if os.Getenv("JAWSDB_URL") != "" {
		// Если переменная окружения есть, используем её
		dbURL = os.Getenv("JAWSDB_URL")
	} else {
		// Если переменной окружения нет, подключаемся к локальной базе данных
		dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database"
	}

	// Открываем подключение к базе данных
	db, err = sql.Open("mysql", dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	// Проверяем, что подключение успешное
	if err := db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}
	return db
}
