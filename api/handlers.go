package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type FetchParams struct {
	OriginPortUnLoCode      string
	DestinationPortUnLoCode string
	Destination             string
	Origin                  string
	DepartureDate           time.Time
}

func FetchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params FetchParams
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Fetch data from Maersk API
		data, err := fetchMaerskData(params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(data)

		w.WriteHeader(http.StatusOK)

		// Return the fetched data from Maersk
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		log.Println("Content-Type set to application/json")

		json.NewEncoder(w).Encode(data)

		// // Insert data into the database
		// _, err = db.Exec(`INSERT INTO vessel_locations (name, location) VALUES ($1, ST_GeogFromText($2))`, data.Name, data.Location)
		// if err != nil {

		// 	log.Println(err)

		// 	http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	return
		// }

		// fmt.Fprintln(w, "Data inserted successfully")
	}
}

// func extractReducedOceanProducts(data models.MaerskPointToPoint) []models.ReducedOceanProduct {
// 	var reducedProducts []models.ReducedOceanProduct

// 	for _, product := range data.OceanProducts {
// 		for _, schedule := range product.TransportSchedules {
// 			reduced := models.ReducedOceanProduct{
// 				CarrierProductID: product.CarrierProductID,
// 				// DepartureDateTime:  schedule.DepartureDateTime.Time,
// 				// ArrivalDateTime:    schedule.ArrivalDateTime.Time,
// 				OriginCity:         string(schedule.Facilities.CollectionOrigin.CityName),
// 				OriginCountry:      string(schedule.Facilities.CollectionOrigin.CountryCode),
// 				DestinationCity:    string(schedule.Facilities.DeliveryDestination.CityName),
// 				DestinationCountry: string(schedule.Facilities.DeliveryDestination.CountryCode),
// 				VesselName:         schedule.FirstDepartureVessel.VesselName,
// 				VesselIMONumber:    schedule.FirstDepartureVessel.VesselIMONumber,
// 			}
// 			reducedProducts = append(reducedProducts, reduced)
// 		}
// 	}

// 	return reducedProducts
// }

// func VesselsHandler(db *sql.DB) http.HandlerFunc {
//     return func(w http.ResponseWriter, r *http.Request) {
//         rows, err := db.Query(`SELECT name, ST_AsText(location) FROM vessel_locations`)
//         if err != nil {
//             http.Error(w, err.Error(), http.StatusInternalServerError)
//             return
//         }
//         defer rows.Close()

//         var vessels []models.Vessel
//         for rows.Next() {
//             var vessel models.Vessel
//             if err := rows.Scan(&vessel.Name, &vessel.Location); err != nil {
//                 http.Error(w, err.Error(), http.StatusInternalServerError)
//                 return
//             }
//             vessels = append(vessels, vessel)
//         }

//         w.Header().Set("Content-Type", "application/json")
//         json.NewEncoder(w).Encode(vessels)
//     }
// }
