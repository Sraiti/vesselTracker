package main

import (
	"log"
	"net/http"

	"github.com/Sraiti/vesselTracker/api"
	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/seeder"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting vessel tracker server...")

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize the database
	database, err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	log.Println("Seeding locations...")
	metrics, err := seeder.SeedLocations(database, 12000)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Seeding completed: %+v", metrics)

	// // Set up HTTP handlers
	// // http.HandleFunc("/fetch", api.FetchHandler(database))
	// // http.HandleFunc("/vessels", api.VesselsHandler(database))

	// // // Get MMSIs from your database
	// vessels, err := db.GetTopVessels(database, 50)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// log.Printf("Found %d vessels", len(vessels))

	// if len(vessels) > 0 {
	// 	mmsis := make([]string, 0, len(vessels))
	// 	for _, v := range vessels {
	// 		if v.MMSI != "" {
	// 			mmsis = append(mmsis, v.MMSI)
	// 		}
	// 	}

	// 	// Start AIS streaming
	// 	aisManager := services.NewAISStreamManager(os.Getenv("AIS_STREAM_API_KEY"), database)
	// 	if err := aisManager.StartStreaming(mmsis); err != nil {
	// 		log.Fatal(err)
	// 	}
	// } else {
	// 	log.Println("No vessels found, skipping AIS streaming")
	// }
	// Add a new handler for the POST request
	http.HandleFunc("/search", api.FetchHandler(database))

	http.HandleFunc("/autocomplete", api.AutoCompleteHandler(database))

	http.HandleFunc("/vessels/route", api.GetVesselRoute(database))

	http.HandleFunc("/vessels/route/geojson", api.GetVesselRouteGeoJSON(database))

	http.HandleFunc("/vessels/tracked", api.GetTrackedVesselsHandler(database))

	http.HandleFunc("/vessels/last-known-position", api.GetVesselLastKnownPosition(database))

	http.HandleFunc("/files", api.FilesExaminerHandler(database))

	// Start the server
	log.Fatal(http.ListenAndServe(":3058", nil))
}
