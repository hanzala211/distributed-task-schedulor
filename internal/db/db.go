package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func New(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("database connected successfully")
	return db
}