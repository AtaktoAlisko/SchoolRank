package driver

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
)
var db *sql.DB
func ConnectDB() *sql.DB {
	var err error
	db, err = sql.Open("mysql", "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database")
	if err != nil {
		log.Fatal("Ошибка подключения к базе данных:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}
	return db
}
