package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	// Remove quotes from the string
	str := string(b)
	str = str[1 : len(str)-1]

	// Try parsing with different time formats
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02",
	}

	var parseErr error
	for _, format := range formats {
		t, err := time.Parse(format, str)
		if err == nil {
			ct.Time = t
			return nil
		}
		parseErr = err
	}

	return fmt.Errorf("could not parse time %s: %v", str, parseErr)
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ct.Time.Format(time.RFC3339) + `"`), nil
}

func UnmarshalMaerskPointToPoint(data []byte) (MaerskPointToPoint, error) {
	var r MaerskPointToPoint
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *MaerskPointToPoint) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type MaerskPointToPoint struct {
	OceanProducts []OceanProduct `json:"oceanProducts"`
}

type OceanProduct struct {
	CarrierProductID          string              `json:"carrierProductId"`
	CarrierProductSequenceID  string              `json:"carrierProductSequenceId"`
	ProductValidFromDate      CustomTime          `json:"productValidFromDate"`
	ProductValidToDate        CustomTime          `json:"productValidToDate"`
	NumberOfProductLinks      string              `json:"numberOfProductLinks"`
	TransportSchedules        []TransportSchedule `json:"transportSchedules"`
	VesselOperatorCarrierCode string              `json:"vesselOperatorCarrierCode"`
}

type TransportSchedule struct {
	DepartureDateTime    CustomTime                  `json:"departureDateTime"`
	ArrivalDateTime      CustomTime                  `json:"arrivalDateTime"`
	Facilities           TransportScheduleFacilities `json:"facilities"`
	TransitTime          string                      `json:"transitTime"`
	FirstDepartureVessel Vessel                      `json:"firstDepartureVessel"`
	TransportLegs        []TransportLeg              `json:"transportLegs"`
}

type TransportScheduleFacilities struct {
	CollectionOrigin    CollectionOrigin `json:"collectionOrigin"`
	DeliveryDestination CollectionOrigin `json:"deliveryDestination"`
}

type CollectionOrigin struct {
	CarrierCityGeoID   string  `json:"carrierCityGeoID"`
	CityName           string  `json:"cityName"`
	CarrierSiteGeoID   string  `json:"carrierSiteGeoID"`
	LocationName       string  `json:"locationName"`
	CountryCode        string  `json:"countryCode"`
	LocationType       string  `json:"locationType"`
	UNLocationCode     string  `json:"UNLocationCode"`
	SiteUNLocationCode string  `json:"siteUNLocationCode"`
	CityUNLocationCode string  `json:"cityUNLocationCode"`
	UNRegionCode       *string `json:"UNRegionCode,omitempty"`
}

type Vessel struct {
	VesselIMONumber   string `json:"vesselIMONumber"`
	CarrierVesselCode string `json:"carrierVesselCode"`
	VesselName        string `json:"vesselName"`
}

type TransportLeg struct {
	DepartureDateTime CustomTime             `json:"departureDateTime"`
	ArrivalDateTime   CustomTime             `json:"arrivalDateTime"`
	Facilities        TransportLegFacilities `json:"facilities"`
	Transport         Transport              `json:"transport"`
}

type TransportLegFacilities struct {
	StartLocation CollectionOrigin `json:"startLocation"`
	EndLocation   CollectionOrigin `json:"endLocation"`
}

type Transport struct {
	TransportMode                string `json:"transportMode"`
	Vessel                       Vessel `json:"vessel"`
	CarrierTradeLaneName         string `json:"carrierTradeLaneName"`
	CarrierDepartureVoyageNumber string `json:"carrierDepartureVoyageNumber"`
	InducementLinkFlag           string `json:"inducementLinkFlag"`
	CarrierServiceCode           string `json:"carrierServiceCode"`
	CarrierServiceName           string `json:"carrierServiceName"`
	LinkDirection                string `json:"linkDirection"`
	CarrierCode                  string `json:"carrierCode"`
	RoutingType                  string `json:"routingType"`
}
