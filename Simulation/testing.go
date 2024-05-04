package Simulation

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Connect() *sql.DB {
	//serverName := "postgreSQL15"
	//username := "postgres"
	//password := "admin"
	//dbName := "HistoricalData"

	// Create the connection string
	connStr := "postgres://treding_user:LXgxMefi1XjBaaXid87qB0i5Uhoe2GN8@dpg-cn858hmd3nmc73db0v3g-a.oregon-postgres.render.com/treding"

	// Connect to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	//defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
	}

	log.Println("Successfully connected to the PostgreSQL database!")
	return db
}
