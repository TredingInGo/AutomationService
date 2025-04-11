package database

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
	// rtHEdzaiWrCuyx3ZpqcVknTPXnSGn3of

	//connStr := "postgres://treding_user:LXgxMefi1XjBaaXid87qB0i5Uhoe2GN8@dpg-cn858hmd3nmc73db0v3g-a.oregon-postgres.render.com/treding"
	connStrNew := "postgresql://trading_rcdz_user:rtHEdzaiWrCuyx3ZpqcVknTPXnSGn3of@dpg-cvoml4emcj7s73882lb0-a.oregon-postgres.render.com/Trading"
	db, err := sql.Open("postgres", connStrNew)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
	}

	log.Println("Successfully connected to the Trading PostgreSQL database!")
	return db
}
