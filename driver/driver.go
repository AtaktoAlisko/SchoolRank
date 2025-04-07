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

    // Проверяем переменные окружения
    if os.Getenv("JAWSDB_URL") != "" {
        dbURL = os.Getenv("JAWSDB_URL")
    } else if os.Getenv("DATABASE_URL") != "" {
        dbURL = os.Getenv("DATABASE_URL")
    } else {
        // Локальная база данных (если переменные окружения нет)
        dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database"
    }

    // Открытие подключения
    db, err = sql.Open("mysql", dbURL)
    if err != nil {
        log.Fatal("Ошибка подключения к базе данных:", err)
    }

    // Проверяем подключение
    if err := db.Ping(); err != nil {
        log.Fatal("Не удалось подключиться к базе данных:", err)
    }

    return db
}
