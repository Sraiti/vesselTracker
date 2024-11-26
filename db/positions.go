package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Sraiti/vesselTracker/models"
	aisstream "github.com/aisstream/ais-message-models/golang/aisStream"
)

// id SERIAL PRIMARY KEY,
//
//	vessel_id INTEGER REFERENCES vessels(id),
//	mmsi TEXT REFERENCES vessels(mmsi),
//	latitude DECIMAL(10,8),
//	longitude DECIMAL(11,8),
//	timestamp TIMESTAMP NOT NULL,
//	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
func InsertPositionReport(db *sql.DB, mmsi string, positionReport aisstream.PositionReport, timestamp models.CustomTime) error {
	log.Printf("Inserting position report for mmsi %s", mmsi)

	if timestamp.IsZero() {
		log.Printf("Timestamp is zero")
		return fmt.Errorf("timestamp is zero")
	}

	vessel, err := GetVesselByMMSI(db, mmsi)
	if err != nil {
		log.Printf("Error getting vessel: %s", err)
		return err
	}

	log.Printf("Vessel found: %+v", vessel)

	log.Printf("Position report: %+v", positionReport)

	_, err = db.Exec(`
        INSERT INTO vessel_positions 
        VALUES (DEFAULT,$1, $2, $3, $4, $5)
    `, vessel.ID, vessel.MMSI, positionReport.Latitude, positionReport.Longitude, timestamp.Time.Format("2006-01-02 15:04:05"))

	updateVesselLastKnownPosition(db, vessel.ID, positionReport.Latitude, positionReport.Longitude)

	log.Println("Position report inserted", err)

	return err
}
