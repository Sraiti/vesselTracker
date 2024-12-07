package models

type MaerskLocation struct {
	CountryCode    string `json:"countryCode"`
	CountryName    string `json:"countryName"`
	CityName       string `json:"cityName"`
	LocationType   string `json:"locationType"`
	LocationName   string `json:"locationName"`
	CarrierGeoID   string `json:"carrierGeoID"`
	UNLocationCode string `json:"UNLocationCode"`
	UNRegionCode   string `json:"UNRegionCode"`
	UNRegionName   string `json:"UNRegionName"`
}
