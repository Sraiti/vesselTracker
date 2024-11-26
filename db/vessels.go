package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

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

func GetVesselByMMSI(db *sql.DB, mmsi string) (Vessel, error) {

	var lat float64
	var lon float64

	var vessel Vessel
	err := db.QueryRow(`SELECT id, imo_number, mmsi, name, is_tracked, carrier_code, appearance_count, last_seen, created_at, ST_Y(last_known_position::geometry) as latitude, ST_X(last_known_position::geometry) as longitude FROM vessels WHERE mmsi = $1`, mmsi).Scan(
		&vessel.ID,
		&vessel.IMONumber,
		&vessel.MMSI,
		&vessel.Name,
		&vessel.IsTracked,
		&vessel.CarrierCode,
		&vessel.AppearanceCount,
		&vessel.LastSeen,
		&vessel.CreatedAt,
		&lat,
		&lon,
	)

	vessel.LastKnownPosition = []float64{lat, lon}
	if err != nil {
		if err == sql.ErrNoRows {
			return Vessel{}, fmt.Errorf("vessel not found")
		}
		return Vessel{}, fmt.Errorf("error getting vessel: %w", err)
	}
	return vessel, nil

}

func GetVesselByIMO(db *sql.DB, imo string) (Vessel, error) {

	var lat float64
	var lon float64

	var vessel Vessel
	err := db.QueryRow(`SELECT 
	id, imo_number, mmsi, name, is_tracked, carrier_code, appearance_count,
	 last_seen, created_at,
        CASE 
                WHEN last_known_position IS NOT NULL 
                THEN ARRAY[ST_Y(last_known_position::geometry), ST_X(last_known_position::geometry)]
                ELSE NULL 
            END as last_known_position
	   FROM vessels WHERE imo_number = $1`, imo).Scan(
		&vessel.ID,
		&vessel.IMONumber,
		&vessel.MMSI,
		&vessel.Name,
		&vessel.IsTracked,
		&vessel.CarrierCode,
		&vessel.AppearanceCount,
		&vessel.LastSeen,
		&vessel.CreatedAt,
		&lat,
		&lon,
	)

	vessel.LastKnownPosition = []float64{lat, lon}
	if err != nil {
		if err == sql.ErrNoRows {
			return Vessel{}, fmt.Errorf("vessel not found")
		}
		return Vessel{}, fmt.Errorf("error getting vessel: %w", err)
	}
	return vessel, nil

}
func GetVesselsByIMOs(db *sql.DB, imos []string) (map[string]Vessel, error) {
	// Use a single query with IN clause
	query := `
        SELECT imo_number, mmsi, name, carrier_code,         
		CASE 
                WHEN last_known_position IS NOT NULL 
                THEN ARRAY[ST_Y(last_known_position::geometry), ST_X(last_known_position::geometry)]
                 ELSE ARRAY[]::float8[]
            END as last_known_position 
        FROM vessels 
        WHERE imo_number = ANY($1)
    `
	rows, err := db.Query(query, pq.Array(imos))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vessels := make(map[string]Vessel)

	for rows.Next() {
		var vessel Vessel

		err := rows.Scan(
			&vessel.IMONumber,
			&vessel.MMSI,
			&vessel.Name,
			&vessel.CarrierCode,
			pq.Array(&vessel.LastKnownPosition),
		)
		if err != nil {
			return nil, err
		}
		vessels[vessel.IMONumber] = vessel
	}
	return vessels, nil
}

func updateVesselLastKnownPosition(db *sql.DB, vesselID int, latitude float64, longitude float64) error {
	_, err := db.Exec(`UPDATE vessels SET last_known_position = ST_Point($2, $3, 4326) WHERE id = $1`, vesselID, latitude, longitude)
	return err
}

func GetVesselLastKnownPosition(db *sql.DB, imo string) ([]float64, error) {
	var lat float64
	var lon float64

	err := db.QueryRow(`SELECT  ST_Y(last_known_position::geometry) as latitude,
            ST_X(last_known_position::geometry) as longitude FROM vessels WHERE imo_number = $1`, imo).Scan(&lat, &lon)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vessel position not found")
	}

	return []float64{lat, lon}, err
}
