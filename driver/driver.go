package driver

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// ConnectDB establishes a connection to the database
func ConnectDB() *sql.DB {
	var db *sql.DB
	var err error
	var dbURL string

	// Check for available database URLs in order of preference
	if url := os.Getenv("JAWSDB_URL"); url != "" {
		dbURL = url
		log.Println("Using JAWSDB_URL for database connection")
	} else if url := os.Getenv("DATABASE_URL"); url != "" {
		dbURL = url
		log.Println("Using DATABASE_URL for database connection")
	} else if url := os.Getenv("JAWSDB_PURPLE_URL"); url != "" {
		dbURL = url
		log.Println("Using JAWSDB_PURPLE_URL for database connection")
	} else {
		// Local development fallback
		dbURL = "root:Zhanibek321@tcp(127.0.0.1:3306)/my_database"
		log.Println("No database URL found, using local database configuration")
	}

	// Open the database connection
	db, err = sql.Open("mysql", dbURL)
	if err != nil {
		log.Fatal("Error opening database connection:", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection successful")
	return db
}