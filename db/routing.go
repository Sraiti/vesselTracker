package db

import "database/sql"

func GetVesselRoute(db *sql.DB, mmsi string) ([][]float64, error) {

	positions, err := db.Query(`SELECT longitude, latitude FROM vessel_positions ORDER BY timestamp ASC`)

	if err != nil {
		return nil, err
	}

	var route [][]float64

	for positions.Next() {
		var longitude, latitude float64
		positions.Scan(&longitude, &latitude)
		route = append(route, []float64{longitude, latitude})
	}

	return route, nil
}
