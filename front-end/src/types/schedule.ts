interface Position {
  lat: number;
  lng: number;
}

interface TransportLeg {
  DepartureDateTime: string;
  ArrivalDateTime: string;
  VesselCarrierCode: string;
  VesselName: string;
  VesselIMONumber: string;
  VesselMMSI: string;
  LastKnownPosition: number[];
  OriginCity: string;
  OriginName: string;
  OriginCountry: string;
  OriginPortUnLoCode: string;
  OriginCarrierSiteGeoID: string;
  OriginCarrierCityGeoID: string;
  DestinationCity: string;
  DestinationName: string;
  DestinationCountry: string;
  DestinationPortUnLoCode: string;
  DestinationCarrierSiteGeoID: string;
  DestinationCarrierCityGeoID: string;
}

interface Schedule {
  ID: number;
  CarrierProductID: string;
  ProductValidToDate: string;
  ProductValidFromDate: string;
  OriginCity: string;
  OriginName: string;
  OriginCountry: string;
  OriginPortUnLoCode: string;
  OriginCarrierSiteGeoID: string;
  OriginCarrierCityGeoID: string;
  DestinationCity: string;
  DestinationName: string;
  DestinationCountry: string;
  DestinationPortUnLoCode: string;
  DestinationCarrierSiteGeoID: string;
  DestinationCarrierCityGeoID: string;
  DepartureVesselCarrierCode: string;
  DepartureVesselName: string;
  DepartureVesselIMONumber: string;
  DepartureVesselMMSI: string;
  LastKnownPosition: number[];
  DepartureDateTime: string;
  ArrivalDateTime: string;
  TransitTime: number;
  TransportLegs: TransportLeg[];
}

interface ScheduleResponse {
  schedules: Schedule[];
  vesselsMMSI: string[];
  vesselsIMONumber: string[];
}

export type { Schedule, ScheduleResponse, TransportLeg, Position }; 