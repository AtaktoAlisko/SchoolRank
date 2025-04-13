package driver

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// ConnectDB устанавливает подключение к базе данных
func ConnectDB() *sql.DB {
	var err error
	var dbURL string

	// Получаем строку подключения из переменных окружения
	dbURL = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", 
		os.Getenv("DB_USER"), 
		os.Getenv("DB_PASSWORD"), 
		os.Getenv("DB_HOST"), 
		os.Getenv("DB_PORT"), 
		os.Getenv("DB_NAME"))

	// Открываем подключение к базе данных
	db, err = sql.Open("mysql", dbURL)
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}

	log.Println("Подключение к базе данных успешно!")
	return db
}
