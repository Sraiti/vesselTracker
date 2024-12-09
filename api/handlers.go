package api

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
	"github.com/Sraiti/vesselTracker/utils"
)

type FetchParams struct {
	OriginPortUnLoCode      string
	DestinationPortUnLoCode string
	Destination             string
	Origin                  string
	DepartureDate           models.CustomTime
}

type VesselsMessagesSummary struct {
	EventTypes []string
	MMSIs      []float64
	LastEvent  models.CustomTime
	Count      int
}
type VesselMessageSummary struct {
	EventTypes string
	MMSIs      float64
	TimeStamp  models.CustomTime
}

func saveScheduleToDB(db *sql.DB, data []models.ReducedOceanProduct) {

	for _, product := range data {

		validTo := product.ProductValidToDate.Time
		validFrom := product.ProductValidFromDate.Time
		var oceanProductID int

		err := db.QueryRow(`
				INSERT INTO ocean_products (
					carrier_product_id, product_valid_to_date, product_valid_from_date,
					origin_city, origin_name, origin_country, origin_port_un_lo_code,
					origin_carrier_site_geo_id, origin_carrier_city_geo_id,
					destination_city, destination_name, destination_country,
					destination_port_un_lo_code, destination_carrier_site_geo_id,
					destination_carrier_city_geo_id, departure_vessel_carrier_code,
					departure_vessel_name, departure_vessel_imo_number,
					departure_vessel_mmsi, departure_date_time, arrival_date_time,
					transit_time
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 
						  $15, $16, $17, $18, $19, $20, $21, $22)
				ON CONFLICT ( origin_port_un_lo_code, destination_port_un_lo_code, 
							departure_vessel_imo_number, departure_date_time, arrival_date_time) 
				DO UPDATE SET
					carrier_product_id = $1,
					product_valid_to_date = $2,
					product_valid_from_date = $3,
					origin_city = $4,
					origin_name = $5, 
					origin_country = $6,
					origin_carrier_site_geo_id = $8,
					origin_carrier_city_geo_id = $9,
					destination_city = $10,
					destination_name = $11,
					destination_country = $12,
					destination_carrier_site_geo_id = $14,
					destination_carrier_city_geo_id = $15,
					departure_vessel_carrier_code = $16,
					departure_vessel_name = $17,
					departure_vessel_mmsi = $19,
					transit_time = $22
				RETURNING id`,
			product.CarrierProductID, validTo, validFrom, product.OriginCity,
			product.OriginName, product.OriginCountry, product.OriginPortUnLoCode,
			product.OriginCarrierSiteGeoID, product.OriginCarrierCityGeoID,
			product.DestinationCity, product.DestinationName,
			product.DestinationCountry, product.DestinationPortUnLoCode,
			product.DestinationCarrierSiteGeoID, product.DestinationCarrierCityGeoID,
			product.DepartureVesselCarrierCode, product.DepartureVesselName,
			product.DepartureVesselIMONumber, product.DepartureVesselMMSI,
			product.DepartureDateTime.Time, product.ArrivalDateTime.Time,
			product.TransitTime).Scan(&oceanProductID)

		if err != nil {
			log.Printf("Error inserting ocean product: %v", err)
			continue
		}

		for _, leg := range product.TransportLegs {
			// Insert ocean product first

			// Insert transport leg linked to ocean product
			_, err = db.Exec(`
				INSERT INTO transport_legs (
					ocean_product_id, departure_date_time, arrival_date_time,
					vessel_carrier_code, vessel_name, vessel_imo_number, vessel_mmsi,
					origin_city, origin_name, origin_country, origin_port_un_lo_code,
					origin_carrier_site_geo_id, origin_carrier_city_geo_id,
					destination_city, destination_name, destination_country,
					destination_port_un_lo_code, destination_carrier_site_geo_id,
					destination_carrier_city_geo_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
						  $14, $15, $16, $17, $18, $19)
				ON CONFLICT ( vessel_imo_number, departure_date_time, arrival_date_time)
				DO UPDATE SET	
					vessel_carrier_code = $4,
					vessel_name = $5,
					vessel_mmsi = $7,
					origin_city = $8,
					origin_name = $9,
					origin_country = $10,
					origin_carrier_site_geo_id = $12,
					origin_carrier_city_geo_id = $13,
					destination_city = $14,
					destination_name = $15,
					destination_country = $16,
					destination_carrier_site_geo_id = $18,
					destination_carrier_city_geo_id = $19`,
				oceanProductID, leg.DepartureDateTime.Time, leg.ArrivalDateTime.Time,
				leg.VesselCarrierCode, leg.VesselName, leg.VesselIMONumber,
				leg.VesselMMSI, leg.OriginCity, leg.OriginName, leg.OriginCountry,
				leg.OriginPortUnLoCode, leg.OriginCarrierSiteGeoID,
				leg.OriginCarrierCityGeoID, leg.DestinationCity, leg.DestinationName,
				leg.DestinationCountry, leg.DestinationPortUnLoCode,
				leg.DestinationCarrierSiteGeoID, leg.DestinationCarrierCityGeoID)

			if err != nil {
				log.Printf("Error inserting transport leg: %v", err)
			}
		}
	}

}

func AutoCompleteHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		text := r.URL.Query().Get("text")

		log.Println("AutoCompleteHandler:", text)

		locations, err := db.AutoComplete(database, text)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(locations)
	}
}

func FetchHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		totalStart := time.Now()

		var params FetchParams
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		locations, err := db.GetLocations(database, []string{params.OriginPortUnLoCode, params.DestinationPortUnLoCode})

		if err != nil {

			log.Println("Error getting locations")
			log.Println(err)
		} else if len(locations) != 2 || locations[0].MaerskID == "" || locations[1].MaerskID == "" {
			log.Println("Invalid number of locations")
			go func() {
				locationsWithoutMaerskID := []string{}

				for _, location := range locations {
					log.Println("location")
					log.Println(location.MaerskID)
					if location.MaerskID == "" {
						locationsWithoutMaerskID = append(locationsWithoutMaerskID, location.Unlocode)
					}
				}
				GetMaerskLocations(database, locationsWithoutMaerskID)
				if err != nil {
					log.Println("Error getting Maersk locations")
					log.Println(err)
				}
			}()
		}

		if len(locations[0].Location) == 0 || len(locations[1].Location) == 0 {
			log.Println("Missing coordinates in background")
			go func() {
				log.Println("Enriching missing coordinates in background")
				for _, loc := range locations {
					if len(loc.Location) == 0 {
						lat, lon, err := GetLocationCoordinates(loc)
						if err != nil {
							log.Printf("Error getting coordinates for %s: %v", loc.Unlocode, err)
							continue
						}

						if err := db.UpdateLocationCoordinates(database, loc.Unlocode, lat, lon); err != nil {
							log.Printf("Error updating coordinates for %s: %v", loc.Unlocode, err)
						}
					}
				}
			}()
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Fetch data from Maersk API
		maerskStart := time.Now()
		data, err := GetMaerskPointToPoint(params, locations)
		log.Printf("Maersk API fetch took: %v", time.Since(maerskStart))

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if data.OceanProducts == nil {
			http.Error(w, "No data found", http.StatusNotFound)
			return
		}

		// Extract and process data
		processingStart := time.Now()
		reducedProducts := extractReducedOceanProducts(database, data)
		log.Printf("Data processing took: %v", time.Since(processingStart))

		saveScheduleToDB(database, reducedProducts)

		// Prepare response
		response := struct {
			Schedules        []models.ReducedOceanProduct `json:"schedules"`
			VesselsMMSI      []string                     `json:"vesselsMMSI"`
			VesselsIMONumber []string                     `json:"vesselsIMONumber"`
		}{
			Schedules: reducedProducts,
			VesselsIMONumber: func() []string {
				imoSet := make(map[string]struct{})
				for _, product := range reducedProducts {
					if product.DepartureVesselIMONumber != "" {
						imoSet[product.DepartureVesselIMONumber] = struct{}{}
					}
					for _, leg := range product.TransportLegs {
						if leg.VesselIMONumber != "" {
							imoSet[leg.VesselIMONumber] = struct{}{}
						}
					}
				}
				imoList := make([]string, 0, len(imoSet))
				for imo := range imoSet {
					imoList = append(imoList, imo)
				}
				return imoList
			}(),
			VesselsMMSI: func() []string {
				mmsiSet := make(map[string]struct{})
				for _, product := range reducedProducts {
					if product.DepartureVesselMMSI != "" {
						mmsiSet[product.DepartureVesselMMSI] = struct{}{}
					}
					for _, leg := range product.TransportLegs {
						if leg.VesselMMSI != "" {
							mmsiSet[leg.VesselMMSI] = struct{}{}
						}
					}
				}
				mmsiList := make([]string, 0, len(mmsiSet))
				for mmsi := range mmsiSet {
					mmsiList = append(mmsiList, mmsi)
				}
				return mmsiList
			}(),
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		encodingStart := time.Now()
		json.NewEncoder(w).Encode(response)
		log.Printf("JSON encoding took: %v", time.Since(encodingStart))

		log.Printf("Total request time: %v", time.Since(totalStart))
	}
}

func GetVesselLastKnownPosition(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		mmsi := r.URL.Query().Get("mmsi")

		log.Println("Getting last known position for mmsi:", mmsi)

		var vessel db.Vessel

		vessel, err := db.GetVesselByMMSI(database, mmsi)

		log.Println("Vessel found:", vessel)
		log.Println("vessel last known position:", vessel.LastKnownPosition)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(vessel.LastKnownPosition)
	}
}
func updateVesselsInDB(database *sql.DB, vessels map[string]db.Vessel) {
	for _, vessel := range vessels {
		if err := db.UpsertVessel(database, vessel); err != nil {
			log.Printf("Error upserting vessel: %v", err)
		}
	}
}

type NominatimResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

func GetLocationCoordinates(location db.Location) (float64, float64, error) {
	// Build search query using location name and country
	searchQuery := url.QueryEscape(fmt.Sprintf("%s, %s", location.Name, location.CountryCode))
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", searchQuery)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, err
	}

	// Required by Nominatim's terms of use
	req.Header.Set("User-Agent", "VesselTracker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var results []NominatimResponse
	log.Println("GetLocationCoordinates")
	log.Println("Response:", resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, err
	}

	if len(results) == 0 {
		return 0, 0, fmt.Errorf("no coordinates found for location")
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return 0, 0, err
	}

	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return 0, 0, err
	}

	return lat, lon, nil
}
func extractReducedOceanProducts(database *sql.DB, data models.MaerskPointToPoint) []models.ReducedOceanProduct {
	collectionStart := time.Now()

	// Collect unique IMO numbers
	imoSet := utils.CollectUniqueVessels(data)

	log.Printf("IMO collection took: %v, Found %d unique IMOs", time.Since(collectionStart), len(imoSet))
	mmsiStart := time.Now()

	vf := &utils.VesselFetcher{
		DB:        database,
		MmsiCache: make(map[string]db.Vessel),
	}
	mmsiCache := vf.FetchVesselData(imoSet)

	log.Printf("Vessel data fetching took: %v", time.Since(mmsiStart))

	go updateVesselsInDB(database, mmsiCache)

	// Build and return products
	buildStart := time.Now()

	products := buildReducedProducts(data, mmsiCache)

	log.Printf("Product building took: %v", time.Since(buildStart))

	return products

}

func GetVesselRoute(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		mmsi := r.URL.Query().Get("mmsi")

		log.Println("Getting route for mmsi:", mmsi)

		route, err := db.GetVesselRoute(database, mmsi)

		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println("Route:", route)

		json.NewEncoder(w).Encode(route)
	}
}

func GetVesselRouteGeoJSON(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mmsi := r.URL.Query().Get("mmsi")

		positions, err := db.GetVesselRoute(database, mmsi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Create features slice to store all points
		features := make([]map[string]interface{}, 0, len(positions))

		// Process each position into a GeoJSON feature
		for _, pos := range positions {
			feature := map[string]interface{}{
				"type": "Feature",
				"geometry": map[string]interface{}{
					"type":        "Point",
					"coordinates": pos,
				},
				// "properties": map[string]interface{}{
				// 	"title":         fmt.Sprintf("Point %d", i+1),
				// 	"marker-color":  colors[2], // Cycle through colors
				// 	"marker-size":   "medium",
				// 	"marker-symbol": "circle",
				// 	"point-number":  i + 1,
				// },
			}
			features = append(features, feature)
		}

		// Create the final GeoJSON structure
		geojson := map[string]interface{}{
			"type":     "FeatureCollection",
			"features": features,
		}

		json.NewEncoder(w).Encode(geojson)
	}
}

func buildReducedProducts(data models.MaerskPointToPoint, mmsiCache map[string]db.Vessel) []models.ReducedOceanProduct {
	var reducedProducts []models.ReducedOceanProduct

	getVesselInfo := func(imo string) struct {
		MMSI     string
		Position []float64
	} {
		if vessel, exists := mmsiCache[imo]; exists {
			return struct {
				MMSI     string
				Position []float64
			}{
				MMSI:     vessel.MMSI,
				Position: vessel.LastKnownPosition,
			}
		}
		return struct {
			MMSI     string
			Position []float64
		}{
			MMSI:     "",
			Position: nil,
		}
	}

	for _, product := range data.OceanProducts {
		for _, schedule := range product.TransportSchedules {
			reduced := models.ReducedOceanProduct{

				// valid to date
				ProductValidToDate: models.CustomTime{Time: product.ProductValidToDate.Time},
				// valid from date
				ProductValidFromDate: models.CustomTime{Time: product.ProductValidFromDate.Time},

				CarrierProductID:  product.CarrierProductID,
				DepartureDateTime: models.CustomTime{Time: schedule.DepartureDateTime.Time},
				ArrivalDateTime:   models.CustomTime{Time: schedule.ArrivalDateTime.Time},
				// Origin
				OriginName:             (schedule.Facilities.CollectionOrigin.LocationName),
				OriginCity:             (schedule.Facilities.CollectionOrigin.CityName),
				OriginCountry:          (schedule.Facilities.CollectionOrigin.CountryCode),
				OriginPortUnLoCode:     schedule.Facilities.CollectionOrigin.UNLocationCode,
				OriginCarrierSiteGeoID: schedule.Facilities.CollectionOrigin.CarrierSiteGeoID,
				OriginCarrierCityGeoID: schedule.Facilities.CollectionOrigin.CarrierCityGeoID,
				// Destination
				DestinationCity:             schedule.Facilities.DeliveryDestination.CityName,
				DestinationName:             schedule.Facilities.DeliveryDestination.LocationName,
				DestinationCountry:          schedule.Facilities.DeliveryDestination.CountryCode,
				DestinationPortUnLoCode:     schedule.Facilities.DeliveryDestination.UNLocationCode,
				DestinationCarrierSiteGeoID: schedule.Facilities.DeliveryDestination.CarrierSiteGeoID,
				DestinationCarrierCityGeoID: schedule.Facilities.DeliveryDestination.CarrierCityGeoID,
				//Vessel
				DepartureVesselName:        schedule.FirstDepartureVessel.VesselName,
				DepartureVesselIMONumber:   schedule.FirstDepartureVessel.VesselIMONumber,
				DepartureVesselCarrierCode: schedule.FirstDepartureVessel.CarrierVesselCode,
				DepartureVesselMMSI:        getVesselInfo(schedule.FirstDepartureVessel.VesselIMONumber).MMSI,
				LastKnownPosition:          getVesselInfo(schedule.FirstDepartureVessel.VesselIMONumber).Position,
				//Transit time
				TransitTime: func() int32 {
					if t, err := strconv.Atoi(schedule.TransitTime); err == nil {
						return int32(t)
					}
					return 0
				}(),
				TransportLegs: func() []models.ReducedTransportLeg {
					legs := []models.ReducedTransportLeg{}
					for _, leg := range schedule.TransportLegs {
						legs = append(legs, models.ReducedTransportLeg{
							//time
							DepartureDateTime: models.CustomTime{Time: leg.DepartureDateTime.Time},
							ArrivalDateTime:   models.CustomTime{Time: leg.ArrivalDateTime.Time},
							//vessel
							VesselCarrierCode: leg.Transport.Vessel.CarrierVesselCode,
							VesselName:        leg.Transport.Vessel.VesselName,
							VesselIMONumber:   leg.Transport.Vessel.VesselIMONumber,
							VesselMMSI:        getVesselInfo(leg.Transport.Vessel.VesselIMONumber).MMSI,
							LastKnownPosition: getVesselInfo(leg.Transport.Vessel.VesselIMONumber).Position,
							//origin
							OriginCity:             leg.Facilities.StartLocation.CityName,
							OriginName:             leg.Facilities.StartLocation.LocationName,
							OriginCountry:          leg.Facilities.StartLocation.CountryCode,
							OriginPortUnLoCode:     leg.Facilities.StartLocation.UNLocationCode,
							OriginCarrierCityGeoID: leg.Facilities.StartLocation.CarrierCityGeoID,
							OriginCarrierSiteGeoID: leg.Facilities.StartLocation.CarrierSiteGeoID,
							//destination
							DestinationCity:             leg.Facilities.EndLocation.CityName,
							DestinationName:             leg.Facilities.EndLocation.LocationName,
							DestinationCountry:          leg.Facilities.EndLocation.CountryCode,
							DestinationPortUnLoCode:     leg.Facilities.EndLocation.UNLocationCode,
							DestinationCarrierCityGeoID: leg.Facilities.EndLocation.CarrierCityGeoID,
							DestinationCarrierSiteGeoID: leg.Facilities.EndLocation.CarrierSiteGeoID,
						})
					}
					return legs
				}(),
			}
			reducedProducts = append(reducedProducts, reduced)
		}
	}

	return reducedProducts
}

func GetTrackedVesselsHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vessels, err := db.GetTopVessels(database, 40)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Extract MMSIs for AIS tracking
		mmsis := make([]string, 0, len(vessels))
		for _, v := range vessels {
			if v.MMSI != "" {
				mmsis = append(mmsis, v.MMSI)
			}
		}

		response := struct {
			Vessels []db.Vessel `json:"vessels"`
			MMSIs   []string    `json:"mmsis"`
		}{
			Vessels: vessels,
			MMSIs:   mmsis,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
