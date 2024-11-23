package api

import (
	"database/sql"
	"encoding/json"

	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
)

type FetchParams struct {
	OriginPortUnLoCode      string
	DestinationPortUnLoCode string
	Destination             string
	Origin                  string
	DepartureDate           models.CustomTime
}

func FetchHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		totalStart := time.Now()

		var params FetchParams
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			log.Println(err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Fetch data from Maersk API
		maerskStart := time.Now()
		data, err := fetchMaerskData(params)
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
		reducedProducts := extractReducedOceanProducts(db, data)
		log.Printf("Data processing took: %v", time.Since(processingStart))

		// Prepare response
		response := struct {
			RawData          models.MaerskPointToPoint    `json:"rawData"`
			ReducedProducts  []models.ReducedOceanProduct `json:"reducedProducts"`
			VesselsMMSI      []string                     `json:"vesselsMMSI"`
			VesselsIMONumber []string                     `json:"vesselsIMONumber"`
		}{
			RawData:         data,
			ReducedProducts: reducedProducts,
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

func extractReducedOceanProducts(database *sql.DB, data models.MaerskPointToPoint) []models.ReducedOceanProduct {
	collectionStart := time.Now()

	// Use a map as a set to collect unique IMO numbers
	imoSet := make(map[string]db.Vessel)

	// Collect unique IMO numbers
	for _, product := range data.OceanProducts {
		for _, schedule := range product.TransportSchedules {
			if schedule.FirstDepartureVessel.VesselIMONumber != "" {
				imoSet[schedule.FirstDepartureVessel.VesselIMONumber] = db.Vessel{
					IMONumber:   schedule.FirstDepartureVessel.VesselIMONumber,
					Name:        schedule.FirstDepartureVessel.VesselName,
					CarrierCode: schedule.FirstDepartureVessel.CarrierVesselCode,
					MMSI:        "",
				}
			}
			for _, leg := range schedule.TransportLegs {
				if leg.Transport.Vessel.VesselIMONumber != "" {
					imoSet[leg.Transport.Vessel.VesselIMONumber] = db.Vessel{
						IMONumber:   leg.Transport.Vessel.VesselIMONumber,
						Name:        leg.Transport.Vessel.VesselName,
						CarrierCode: leg.Transport.Vessel.CarrierVesselCode,
						MMSI:        "",
					}
				}
			}
		}
	}
	log.Printf("IMO collection took: %v, Found %d unique IMOs", time.Since(collectionStart), len(imoSet))

	// Create MMSI cache and results channel
	mmsiStart := time.Now()
	type mmsiResult struct {
		vessel db.Vessel
		err    error
	}
	resultChan := make(chan mmsiResult, len(imoSet))
	mmsiCache := make(map[string]db.Vessel)

	// Launch goroutines for MMSI fetching
	for imo, vessel := range imoSet {
		go func(imo string) {
			start := time.Now()
			mmsi, err := getVesselMMSIbyIMO(imo)
			vessel.MMSI = mmsi
			if err != nil {
				log.Printf("Error fetching MMSI for IMO %s: %v", imo, err)
			} else {
				log.Printf("Fetched MMSI for IMO %s in %v", imo, time.Since(start))
			}
			resultChan <- mmsiResult{vessel, err}
		}(imo)
	}

	// Collect results
	successCount := 0
	for i := 0; i < len(imoSet); i++ {
		result := <-resultChan
		if result.err == nil {
			mmsiCache[result.vessel.IMONumber] = result.vessel
			successCount++
		}
	}
	log.Printf("MMSI fetching took: %v, Successfully fetched %d/%d MMSIs",
		time.Since(mmsiStart), successCount, len(imoSet))

	// Build reduced products
	buildStart := time.Now()
	var reducedProducts []models.ReducedOceanProduct

	// Helper function to safely get MMSI from cache
	getMMSI := func(imo string) string {
		return mmsiCache[imo].MMSI
	}

	go func() {
		for _, vessel := range mmsiCache {
			err := db.UpsertVessel(database, vessel)
			if err != nil {
				log.Printf("Error upserting vessel: %v", err)
			}
		}
	}()

	// 7. Build the reduced products using our cached MMSI values
	for _, product := range data.OceanProducts {
		for _, schedule := range product.TransportSchedules {
			reduced := models.ReducedOceanProduct{
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
				DepartureVesselMMSI:        getMMSI(schedule.FirstDepartureVessel.VesselIMONumber),
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
							VesselMMSI:        getMMSI(leg.Transport.Vessel.VesselIMONumber),
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

	log.Printf("Product building took: %v", time.Since(buildStart))
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
