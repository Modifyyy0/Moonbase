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

	// Keep databases created before conversation types were introduced usable.
	var hasConversationType int
	err = DB.QueryRow(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		  AND table_name = 'conversations'
		  AND column_name = 'conversation_type'
	`).Scan(&hasConversationType)
	if err != nil {
		return err
	}
	if hasConversationType == 0 {
		if _, err = DB.Exec(`
			ALTER TABLE conversations
			ADD COLUMN conversation_type ENUM('direct', 'group') NOT NULL DEFAULT 'group'
		`); err != nil {
			return err
		}
	}

	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS message_receipts (
			message_id INT NOT NULL,
			user_id INT NOT NULL,
			delivered_at TIMESTAMP NULL,
			read_at TIMESTAMP NULL,
			PRIMARY KEY (message_id, user_id),
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
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
