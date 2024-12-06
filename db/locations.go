package db

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

func AutoComplete(db *sql.DB, text string) ([]Location, error) {
	// Query matches start of UNLOCODE, country_code, or name (case-insensitive)
	query := `
		SELECT id, unlocode, name, country_code, 
			   is_airport, is_port, is_train_station, created_at,
			   CASE 
                WHEN location IS NOT NULL 
                THEN ARRAY[ST_Y(location::geometry), ST_X(location::geometry)]
                 ELSE ARRAY[]::float8[]
            END as location 
		FROM locations 
		WHERE 
			unlocode ILIKE $1 OR 
			country_code ILIKE $1 OR 
			name ILIKE $1
		LIMIT 10
	`

	// Add % after the search term for prefix matching
	searchPattern := text + "%"

	rows, err := db.Query(query, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []Location
	for rows.Next() {
		var loc Location
		err := rows.Scan(
			&loc.ID,
			&loc.Unlocode,
			&loc.Name,
			&loc.CountryCode,
			&loc.IsAirport,
			&loc.IsPort,
			&loc.IsTrainStation,
			&loc.CreatedAt,
			pq.Array(&loc.Location),
		)
		if err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}

	return locations, nil
}

// Location struct to match the database schema
type Location struct {
	ID             int       `json:"id"`
	Unlocode       string    `json:"unlocode"`
	Name           string    `json:"name"`
	CountryCode    string    `json:"country_code"`
	Location       []float64 `json:"location"`
	IsAirport      bool      `json:"is_airport"`
	IsPort         bool      `json:"is_port"`
	IsTrainStation bool      `json:"is_train_station"`
	CreatedAt      time.Time `json:"created_at"`
}
