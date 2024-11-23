package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Sraiti/vesselTracker/api"
	"github.com/Sraiti/vesselTracker/db"
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

	// Set up HTTP handlers
	// http.HandleFunc("/fetch", api.FetchHandler(database))
	// http.HandleFunc("/vessels", api.VesselsHandler(database))

	// Get MMSIs from your database
	vessels, err := db.GetTopVessels(database, 50)
	if err != nil {
		log.Fatal(err)
	}

	mmsis := make([]string, 0, len(vessels))
	for _, v := range vessels {
		if v.MMSI != "" {
			mmsis = append(mmsis, v.MMSI)
		}
	}

	// Start AIS streaming
	aisManager := services.NewAISStreamManager(os.Getenv("AIS_STREAM_API_KEY"), database)
	if err := aisManager.StartStreaming(mmsis); err != nil {
		log.Fatal(err)
	}

	// Add a new handler for the POST request
	http.HandleFunc("/search", api.FetchHandler(database))

	http.HandleFunc("/vessels/tracked", api.GetTrackedVesselsHandler(database))

	// Start the server
	log.Fatal(http.ListenAndServe(":3058", nil))
}
