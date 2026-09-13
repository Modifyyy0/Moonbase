package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var DB *sql.DB

func Query(query string, args ...any) (*sql.Rows, error) {
	return DB.Query(query, args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return DB.QueryRow(query, args...)
}

func Exec(query string, args ...any) (sql.Result, error) {
	return DB.Exec(query, args...)
}

func Connect() error {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Env not loading")
		return err
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?%s",
		user,
		password,
		host,
		port,
		dbName,
		"parseTime=true",
	)

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("MySql not connected")
		return err
	}

	// Actually verify that we can reach MySQL.
	if err := DB.Ping(); err != nil {
		return err
	}

	fmt.Println("Connected")
	return nil
}

func Close() {
	if DB == nil {
		return
	}

	if err := DB.Close(); err != nil {
		fmt.Println("Error closing database:", err)
	}
}
