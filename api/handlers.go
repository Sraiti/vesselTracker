package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
	"github.com/Sraiti/vesselTracker/utils"
	aisstream "github.com/aisstream/ais-message-models/golang/aisStream"
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

func getFileContent(filePath string) VesselMessageSummary {

	log.Println("ais_data/" + filePath)

	content, err := os.ReadFile("ais_data/" + filePath)

	if err != nil {
		log.Fatal(err)
	}

	var packet aisstream.AisStreamMessage

	err = json.Unmarshal(content, &packet)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(packet.MessageType)

	timeStr := packet.MetaData["time_utc"].(string)

	log.Println(timeStr)
	parsedTime, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr)
	log.Println(parsedTime)

	return VesselMessageSummary{
		EventTypes: string(packet.MessageType),
		MMSIs:      packet.MetaData["MMSI_String"].(float64),
		TimeStamp:  models.CustomTime{Time: parsedTime},
	}

}

func FilesExaminerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vesselInfo := map[string]VesselsMessagesSummary{}

		/// reading the files that exists in the folder of ais data and get all the unique mmsi numbers and all the messages types we got and last date we got an event fora each mmsi .
		files, err := os.ReadDir("ais_data")

		if err != nil {
			log.Fatal(err)
		}
		fileNames := []struct {
			Name string
		}{}

		for _, file := range files {

			if file.IsDir() {
				log.Println("Reading directory:", file.Name())
				files, err := os.ReadDir("ais_data/" + file.Name())
				if err != nil {
					log.Fatal(err)
				}
				for _, subFile := range files {

					if subFile.IsDir() {

						subSubFile, err := os.ReadDir("ais_data/" + file.Name() + "/" + subFile.Name())
						if err != nil {
							log.Fatal(err)
						}
						for _, line := range subSubFile {

							fileNames = append(fileNames, struct {
								Name string
							}{Name: line.Name()})

							summary := getFileContent(file.Name() + "/" + subFile.Name() + "/" + line.Name())

							log.Println(summary.TimeStamp)

							vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")] = VesselsMessagesSummary{
								EventTypes: func(existing []string, new string) []string {
									for _, v := range existing {
										if v == new {
											return existing
										}
									}
									return append(existing, new)
								}(vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")].EventTypes, summary.EventTypes),
								MMSIs:     []float64{summary.MMSIs},
								LastEvent: summary.TimeStamp,
								Count:     vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")].Count + 1,
							}
						}
					}

				}
			}
		}

		json.NewEncoder(w).Encode(vesselInfo)
	}
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
