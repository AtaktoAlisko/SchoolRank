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
        dbURL = os.Getenv("JAWSDB_URL") // Используем строку подключения для Heroku
    } else {
        dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database" // Локальная база данных
    }

    db, err = sql.Open("mysql", dbURL)
    if err != nil {
        log.Fatal("Ошибка подключения к базе данных:", err)
    }
    if err := db.Ping(); err != nil {
        log.Fatal("Не удалось подключиться к базе данных:", err)
    }
    return db
}
