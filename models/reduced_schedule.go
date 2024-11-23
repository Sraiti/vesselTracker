package models

type ReducedOceanProduct struct {
	ID int64
	// maersk product id
	// might b e empty for other companies
	CarrierProductID     string
	productValidToDate   CustomTime
	productValidFromDate CustomTime
	//origin
	OriginCity             string
	OriginName             string
	OriginCountry          string
	OriginPortUnLoCode     string
	OriginCarrierSiteGeoID string
	OriginCarrierCityGeoID string
	//destination
	DestinationCity             string
	DestinationName             string
	DestinationCountry          string
	DestinationPortUnLoCode     string
	DestinationCarrierSiteGeoID string
	DestinationCarrierCityGeoID string
	//departure vessel
	DepartureVesselCarrierCode string
	DepartureVesselName        string
	DepartureVesselIMONumber   string
	DepartureVesselMMSI        string
	//time
	DepartureDateTime CustomTime
	ArrivalDateTime   CustomTime
	//transit time
	TransitTime int32
	//transport legs
	TransportLegs []ReducedTransportLeg
}

type ReducedTransportLeg struct {
	//time
	DepartureDateTime CustomTime
	ArrivalDateTime   CustomTime

	//vessel
	VesselCarrierCode string
	VesselName        string
	VesselIMONumber   string
	VesselMMSI        string
	//origin
	OriginCity             string
	OriginName             string
	OriginCountry          string
	OriginPortUnLoCode     string
	OriginCarrierSiteGeoID string
	OriginCarrierCityGeoID string
	//destination
	DestinationCity             string
	DestinationName             string
	DestinationCountry          string
	DestinationPortUnLoCode     string
	DestinationCarrierSiteGeoID string
	DestinationCarrierCityGeoID string
}
