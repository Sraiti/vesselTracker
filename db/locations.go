package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Sraiti/vesselTracker/models"
	"github.com/lib/pq"
)

func UpsertLocations(db *sql.DB, locations []models.MaerskLocation) error {
	log.Println("Upserting locations")

	// Skip if no locations to update
	if len(locations) == 0 {
		return nil
	}

	// Prepare the bulk update query
	query := `
		UPDATE locations 
		SET maersk_id = tmp.maersk_id 
		FROM (
			SELECT unnest($1::text[]) as unlocode,
				   unnest($2::text[]) as maersk_id
		) tmp 
		WHERE locations.unlocode = tmp.unlocode`

	// Prepare the parameter arrays
	unlocodes := make([]string, len(locations))
	maerskIDs := make([]string, len(locations))

	for i, loc := range locations {
		unlocodes[i] = loc.UNLocationCode
		maerskIDs[i] = loc.CarrierGeoID
	}

	// Execute the bulk update
	_, err := db.Exec(query, pq.Array(unlocodes), pq.Array(maerskIDs))
	if err != nil {
		log.Printf("Error performing bulk upsert: %v", err)
		return err
	}

	return nil
}

func GetLocations(db *sql.DB, unLoCodes []string) ([]Location, error) {
	log.Println("Getting locations")

	query := `SELECT id, unlocode, name, country_code, is_airport, is_port, is_train_station, created_at, maersk_id 
			FROM locations 
			WHERE unlocode = ANY ($1)`

	rows, err := db.Query(query, pq.Array(unLoCodes))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []Location

	for rows.Next() {
		var loc Location
		var maerskID sql.NullString // Use NullString to handle NULL values

		err := rows.Scan(&loc.ID,
			&loc.Unlocode,
			&loc.Name,
			&loc.CountryCode,
			&loc.IsAirport,
			&loc.IsPort,
			&loc.IsTrainStation,
			&loc.CreatedAt,
			&maerskID)
		if err != nil {
			return nil, err
		}

		if maerskID.Valid {
			loc.MaerskID = maerskID.String
		}

		locations = append(locations, loc)
	}

	return locations, nil
}

func UpdateLocationCoordinates(db *sql.DB, unlocode string, latitude, longitude float64) error {
	_, err := db.Exec(`
        UPDATE locations 
        SET location = ST_SetSRID(ST_MakePoint($3, $2), 4326)
        WHERE unlocode = $1 AND location IS NULL
    `, unlocode, latitude, longitude)

	if err != nil {
		return fmt.Errorf("error updating location coordinates: %w", err)
	}
	return nil
}

func AutoComplete(db *sql.DB, text string) ([]Location, error) {
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
			name ILIKE $1 AND is_port = true
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
	MaerskID       string    `json:"maersk_id"`
}
