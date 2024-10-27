package main

import (
	"log"
	"net/http"

	"github.com/Sraiti/vesselTracker/api"
	"github.com/Sraiti/vesselTracker/db"
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


	// Add a new handler for the POST request
	http.HandleFunc("/search", api.FetchHandler(database))

	// Start the server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
