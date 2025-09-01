package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(connStr string) error {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	if err = DB.Ping(); err != nil {
		return err
	}
	// Apply migrations
	if err := RunMigrations(); err != nil {
		return err
	}
	log.Println("Database connected successfully")
	return nil
}
