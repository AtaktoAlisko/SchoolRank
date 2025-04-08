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

    // Получаем строку подключения из переменной окружения DATABASE_URL
    dbURL = os.Getenv("DATABASE_URL")

    // Если переменная окружения не установлена, используем локальную базу данных
    if dbURL == "" {
        log.Println("DATABASE_URL не установлена, используем локальную базу данных.")
        dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database" // Локальная база данных
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

    log.Println("Подключение к базе данных успешно!")
    return db
}
