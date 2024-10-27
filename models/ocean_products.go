package models

import (
	"encoding/json"
	"time"
)

type CustomTime struct {
	time.Time
}

type ReducedOceanProduct struct {
	ID int64
	// maersk product id
	// might b e empty for other companies
	CarrierProductID     string
	productValidToDate   time.Time
	productValidFromDate time.Time
	OriginCity           string
	OriginCountry        string
	DestinationCity      string
	DestinationCountry   string
	VesselCarrierCode    string
	VesselName           string
	TransitTime          int32
	VesselIMONumber      string
	DepartureDateTime    time.Time
	ArrivalDateTime      time.Time
}

// const customTimeLayout = "2006-01-02T15:04:05"

// func (ct *CustomTime) UnmarshalJSON(b []byte) error {
// str := string(b)
// // str = str[1 : len(str)-1] // Remove quotes

// t, err := time.Parse(time.RFC1123Z, str)

// if err != nil {
// 	log.Println(err)
// 	return err
// }
// ct.Time = t
// return nil
// }

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
	ProductValidFromDate      string              `json:"productValidFromDate"`
	ProductValidToDate        string              `json:"productValidToDate"`
	NumberOfProductLinks      string              `json:"numberOfProductLinks"`
	TransportSchedules        []TransportSchedule `json:"transportSchedules"`
	VesselOperatorCarrierCode string              `json:"vesselOperatorCarrierCode"`
}

type TransportSchedule struct {
	DepartureDateTime    CustomTime                  `json:"departureDateTime"`
	ArrivalDateTime      CustomTime                  `json:"arrivalDateTime"`
	Facilities           TransportScheduleFacilities `json:"facilities"`
	FirstDepartureVessel Vessel                      `json:"firstDepartureVessel"`
	TransportLegs        []TransportLeg              `json:"transportLegs"`
}

type TransportScheduleFacilities struct {
	CollectionOrigin    CollectionOrigin `json:"collectionOrigin"`
	DeliveryDestination CollectionOrigin `json:"deliveryDestination"`
}

type CollectionOrigin struct {
	CarrierCityGeoID   CarrierCityGeoID `json:"carrierCityGeoID"`
	CityName           CityName         `json:"cityName"`
	CarrierSiteGeoID   CarrierSiteGeoID `json:"carrierSiteGeoID"`
	LocationName       LocationName     `json:"locationName"`
	CountryCode        CountryCode      `json:"countryCode"`
	LocationType       LocationType     `json:"locationType"`
	UNLocationCode     UnLocationCode   `json:"UNLocationCode"`
	SiteUNLocationCode UnLocationCode   `json:"siteUNLocationCode"`
	CityUNLocationCode UnLocationCode   `json:"cityUNLocationCode"`
	UNRegionCode       *string          `json:"UNRegionCode,omitempty"`
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

type CarrierCityGeoID string

const (
	The0C29F4Lwxiito CarrierCityGeoID = "0C29F4LWXIITO"
	The2Iw9P6J7Xaw72 CarrierCityGeoID = "2IW9P6J7XAW72"
)

type CarrierSiteGeoID string

const (
	The0Ke79A8Ug7Opa CarrierSiteGeoID = "0KE79A8UG7OPA"
	The37O5Hq17Xcl3X CarrierSiteGeoID = "37O5HQ17XCL3X"
)

type CityName string

const (
	CityNamePortTangierMediterranee CityName = "Port Tangier Mediterranee"
	Shanghai                        CityName = "Shanghai"
)

type UnLocationCode string

const (
	Cnsha UnLocationCode = "CNSHA"
	Maptm UnLocationCode = "MAPTM"
)

type CountryCode string

const (
	CN CountryCode = "CN"
	Ma CountryCode = "MA"
)

type LocationName string

const (
	LocationNamePortTangierMediterranee LocationName = "Port Tangier Mediterranee"
	YangshanSghGuandongTerminal         LocationName = "YANGSHAN SGH GUANDONG TERMINAL"
)

type LocationType string

const (
	Terminal LocationType = "TERMINAL"
)
