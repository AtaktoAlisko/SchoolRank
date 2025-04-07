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

	// Получаем строку подключения из переменной окружения JAWSDB_URL
	dbURL = os.Getenv("JAWSDB_URL")
	if dbURL == "" {
		// Если на Heroku не настроена строка подключения, используй локальную базу данных
		// Это нужно только для локальной разработки.
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
