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

	// Проверяем, есть ли переменная окружения JAWSDB_URL или DATABASE_URL
	if os.Getenv("JAWSDB_URL") != "" {
		// Если переменная окружения есть, используем её
		dbURL = os.Getenv("JAWSDB_URL")
	} else if os.Getenv("DATABASE_URL") != "" {
		// Если переменной окружения нет, используем DATABASE_URL
		dbURL = os.Getenv("DATABASE_URL")
	} else {
		// Если переменных окружения нет, подключаемся к локальной базе данных
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
	
	// Возвращаем объект подключения
	return db
}
