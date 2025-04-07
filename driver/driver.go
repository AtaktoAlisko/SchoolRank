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

    // Используем строку подключения для Heroku (если переменная окружения JAWSDB_URL установлена)
    if os.Getenv("JAWSDB_URL") != "" {
        dbURL = os.Getenv("mysql://qp5og7m2hwek0759:l0exy0xxnha9d4gb@ofcmikjy9x4lroa2.cbetxkdyhwsb.us-east-1.rds.amazonaws.com:3306/e1t15009mrgutxos")
    } else {
        dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database" // Локальная база данных для разработки
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
