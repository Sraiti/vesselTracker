package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Vessel struct {
	ID              int64
	IMONumber       string
	MMSI            string
	Name            string
	CarrierCode     string
	AppearanceCount int
	LastSeen        time.Time
	CreatedAt       time.Time
}

func UpsertVessel(db *sql.DB, vessel Vessel) error {
	_, err := db.Exec(`
        INSERT INTO vessels (imo_number, mmsi, name, carrier_code, appearance_count)
        VALUES ($1, $2, $3, $4, 1)
        ON CONFLICT (imo_number) 
        DO UPDATE SET 
            mmsi = EXCLUDED.mmsi,
            name = EXCLUDED.name,
            carrier_code = EXCLUDED.carrier_code,
            appearance_count = vessels.appearance_count + 1,
            last_seen = CURRENT_TIMESTAMP
    `, vessel.IMONumber, vessel.MMSI, vessel.Name, vessel.CarrierCode)
	return err
}

func UpdateTrackedVessels(db *sql.DB, mmsis []string) (int64, error) {

	// Then update the specified vessels to tracked
	// Create a string with the correct number of placeholders
	placeholders := make([]string, len(mmsis))
	args := make([]interface{}, len(mmsis))
	for i, mmsi := range mmsis {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = mmsi
	}

	query := fmt.Sprintf(`
        UPDATE vessels 
        SET is_tracked = true 
        WHERE mmsi IN (%s)`,
		strings.Join(placeholders, ","))

	result, err := db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to update tracked vessels: %v", err)
	}

	return result.RowsAffected()
}

// GetTopVessels returns the most frequently appearing vessels
func GetTopVessels(db *sql.DB, limit int) ([]Vessel, error) {
	rows, err := db.Query(`
        SELECT id, imo_number, mmsi, name, carrier_code, appearance_count, last_seen, created_at
        FROM vessels
        ORDER BY appearance_count DESC, last_seen DESC
        LIMIT $1
    `, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vessels []Vessel
	for rows.Next() {
		var v Vessel
		err := rows.Scan(&v.ID, &v.IMONumber, &v.MMSI, &v.Name, &v.CarrierCode,
			&v.AppearanceCount, &v.LastSeen, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		vessels = append(vessels, v)
	}
	return vessels, nil
}
