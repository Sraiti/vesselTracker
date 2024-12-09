export interface Location {
  id: number;
  unlocode: string;
  name: string;
  country_code: string;
  location: [number, number] | null;
  is_airport: boolean;
  is_port: boolean;
  is_train_station: boolean;
  created_at: string;
  maersk_id: string;
}

export interface VesselGeoData {
  features?: Feature[];
}

export interface Feature {
  geometry?: Geometry;
  type?: string;
}

export interface Geometry {
  coordinates?: number[];
  type?: string;
}
