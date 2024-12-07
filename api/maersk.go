package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
)

func GetMaerskPointToPoint(params FetchParams, locations []db.Location) (models.MaerskPointToPoint, error) {

	baseUrl := "https://api.maersk.com/products/ocean-products?vesselOperatorCarrierCode=MAEU"

	var url string

	if locations[0].MaerskID != "" && locations[1].MaerskID != "" {
		url = fmt.Sprintf("%s&carrierDeliveryDestinationGeoID=%s&carrierCollectionOriginGeoID=%s",
			baseUrl,
			locations[1].MaerskID,
			locations[0].MaerskID,
		)
	} else {

		url = fmt.Sprintf("%s&collectionOriginCountryCode=%s&collectionOriginCityName=%s&deliveryDestinationCountryCode=%s&deliveryDestinationCityName=%s",
			baseUrl,
			params.OriginPortUnLoCode[:2],
			(params.Origin),
			params.DestinationPortUnLoCode[:2],
			(params.Destination),
		)
	}
	log.Println(url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Consumer-Key", os.Getenv("CONSUMER_KEY"))

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Println(err)
		return models.MaerskPointToPoint{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return models.MaerskPointToPoint{}, err
	}

	var data models.MaerskPointToPoint

	data, err = models.UnmarshalMaerskPointToPoint(body)
	if err != nil {
		return models.MaerskPointToPoint{}, err
	}

	return data, nil
}

func GetMaerskLocations(database *sql.DB, unLoCodes []string) ([]models.MaerskLocation, error) {

	var locations []models.MaerskLocation

	log.Println("Getting Maersk locations")
	log.Println(unLoCodes)
	for _, unLoCode := range unLoCodes {
		url := fmt.Sprintf("https://api.maersk.com/reference-data/locations?vesselOperatorCarrierCode=MAEU&locationType=CITY&UNLocationCode=%s", unLoCode)

		log.Println(url)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("Consumer-Key", os.Getenv("CONSUMER_KEY"))

		res, err := http.DefaultClient.Do(req)

		if err != nil {
			log.Println(err)
			return []models.MaerskLocation{}, err
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)

		if err != nil {
			log.Println("Error reading body")
			log.Println(err)
			continue
		}
		var r []models.MaerskLocation
		err = json.Unmarshal(body, &r)
		if err != nil {
			log.Println("Error unmarshalling body")
			log.Println(err)
			continue
		}
		locations = append(locations, r...)
	}

	db.UpsertLocations(database, locations)

	return locations, nil
}
