package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Sraiti/vesselTracker/models"
)

func fetchMaerskData(
	params FetchParams,
) (models.MaerskPointToPoint, error) {

	baseUrl := "https://api.maersk.com/products/ocean-products?vesselOperatorCarrierCode=MAEU"

	url := fmt.Sprintf("%s&collectionOriginCountryCode=%s&collectionOriginCityName=%s&deliveryDestinationCountryCode=%s&deliveryDestinationCityName=%s",
		baseUrl,
		params.OriginPortUnLoCode[:2],
		(params.Origin),
		params.DestinationPortUnLoCode[:2],
		(params.Destination),
	)

	log.Println("base Url", url)

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
