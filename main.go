package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/Sraiti/vesselTracker/api"
	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/middleware"
	"github.com/Sraiti/vesselTracker/seeder"
	"github.com/Sraiti/vesselTracker/services"
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

	mux := http.NewServeMux()

	// cors.Default() setup the middleware with default options being
	// all origins accepted with simple methods (GET, POST). See
	// documentation below for more options.

	log.Println("Seeding locations...")
	metrics, err := seeder.SeedLocations(database, 12000)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Seeding completed: %+v", metrics)

	// Initialize AIS streaming service
	if err := initializeAISStreaming(database); err != nil {
		log.Fatal(err)
	}

	// Function to initialize AIS streaming
	mux.Handle("/search", middleware.CorsMiddleware(http.HandlerFunc(api.FetchHandler(database))))
	mux.Handle("/autocomplete", middleware.CorsMiddleware(http.HandlerFunc(api.AutoCompleteHandler(database))))
	mux.Handle("/vessels/route", middleware.CorsMiddleware(http.HandlerFunc(api.GetVesselRoute(database))))
	mux.Handle("/vessels/route/geojson", middleware.CorsMiddleware(http.HandlerFunc(api.GetVesselRouteGeoJSON(database))))
	mux.Handle("/vessels/tracked", middleware.CorsMiddleware(http.HandlerFunc(api.GetTrackedVesselsHandler(database))))
	mux.Handle("/vessels/last-known-position", middleware.CorsMiddleware(http.HandlerFunc(api.GetVesselLastKnownPosition(database))))
	mux.Handle("/files", middleware.CorsMiddleware(http.HandlerFunc(api.FilesExaminerHandler(database))))

	// Start the server
	log.Fatal(http.ListenAndServe(":3058", mux))
}

func initializeAISStreaming(database *sql.DB) error {
	vessels, err := db.GetTopVessels(database, 50)
	if err != nil {
		return err
	}

	log.Printf("Found %d vessels", len(vessels))

	if len(vessels) > 0 {
		mmsis := make([]string, 0, len(vessels))
		for _, v := range vessels {
			if v.MMSI != "" {
				mmsis = append(mmsis, v.MMSI)
			}
		}

		// Start AIS streaming
		aisManager := services.NewAISStreamManager(os.Getenv("AIS_STREAM_API_KEY"), database)
		if err := aisManager.StartStreaming(mmsis); err != nil {
			return err
		}
	} else {
		log.Println("No vessels found, skipping AIS streaming")
	}
	return nil
}
