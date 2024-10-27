package db

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {
	connStr := "user=" + os.Getenv("POSTGRES_USER") +
		" password=" + os.Getenv("POSTGRES_PASSWORD") +
		" dbname=" + os.Getenv("POSTGRES_DB") +
		" sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Create table with PostGIS support
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS vessel_locations (
		id SERIAL PRIMARY KEY,
		name TEXT,
		location GEOGRAPHY(POINT, 4326)
	)`)
	if err != nil {
		return nil, err
	}

	return db, nil
}
