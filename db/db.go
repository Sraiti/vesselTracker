package db

import (
	"database/sql"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func InitDB() (*sql.DB, error) {

	connStr := "user=" + os.Getenv("POSTGRES_USER") +
		" password=" + os.Getenv("POSTGRES_PASSWORD") +
		" dbname=" + os.Getenv("POSTGRES_DB") +
		" sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Create table with PostGIS support
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS vessel_locations (
		id SERIAL PRIMARY KEY,
		name TEXT,
		location GEOGRAPHY(POINT, 4326)
	)`)

	if err != nil {
		return nil, err
	}

	// Create table for reduced ocean products
	_, err = db.Exec(`

		CREATE TABLE IF NOT EXISTS vessels (
			id SERIAL PRIMARY KEY,
			imo_number TEXT UNIQUE,
			mmsi TEXT UNIQUE,
			name TEXT,
			is_tracked BOOLEAN DEFAULT false,
			carrier_code TEXT,
			appearance_count INT DEFAULT 0,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_known_position GEOGRAPHY(POINT, 4326)
		);


		CREATE TABLE IF NOT EXISTS ocean_products (
			id SERIAL PRIMARY KEY,
			carrier_product_id TEXT,
			product_valid_to_date TIMESTAMP,
			product_valid_from_date TIMESTAMP,
			origin_city TEXT,
			origin_name TEXT,
			origin_country TEXT,
			origin_port_un_lo_code TEXT,
			origin_carrier_site_geo_id TEXT,
			origin_carrier_city_geo_id TEXT,
			destination_city TEXT,
			destination_name TEXT,
			destination_country TEXT,
			destination_port_un_lo_code TEXT,
			destination_carrier_site_geo_id TEXT,
			destination_carrier_city_geo_id TEXT,
			departure_vessel_carrier_code TEXT,
			departure_vessel_name TEXT,
			departure_vessel_imo_number TEXT REFERENCES vessels(imo_number),
			departure_vessel_mmsi TEXT REFERENCES vessels(mmsi),
			departure_date_time TIMESTAMP,
			arrival_date_time TIMESTAMP,
			transit_time INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS transport_legs (
			id SERIAL PRIMARY KEY,
			ocean_product_id INTEGER REFERENCES ocean_products(id) ON DELETE CASCADE,
			departure_date_time TIMESTAMP,
			arrival_date_time TIMESTAMP,
			vessel_carrier_code TEXT,
			vessel_name TEXT,
			vessel_imo_number TEXT REFERENCES vessels(imo_number),
			vessel_mmsi TEXT REFERENCES vessels(mmsi),
			origin_city TEXT,
			origin_name TEXT,
			origin_country TEXT,
			origin_port_un_lo_code TEXT,
			origin_carrier_site_geo_id TEXT,
			origin_carrier_city_geo_id TEXT,
			destination_city TEXT,
			destination_name TEXT,
			destination_country TEXT,
			destination_port_un_lo_code TEXT,
			destination_carrier_site_geo_id TEXT,
			destination_carrier_city_geo_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);


		CREATE TABLE IF NOT EXISTS vessel_positions (
			id SERIAL PRIMARY KEY,
			vessel_id INTEGER REFERENCES vessels(id),
			mmsi TEXT REFERENCES vessels(mmsi),
			latitude DECIMAL(10,8),
			longitude DECIMAL(11,8), 
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS vessel_positions_vessel_id_idx ON vessel_positions(vessel_id);
		CREATE INDEX IF NOT EXISTS vessel_positions_mmsi_idx ON vessel_positions(mmsi);
		CREATE INDEX IF NOT EXISTS vessel_positions_timestamp_idx ON vessel_positions(timestamp);
		CREATE INDEX IF NOT EXISTS vessels_imo_idx ON vessels(imo_number);
		CREATE INDEX IF NOT EXISTS vessels_mmsi_idx ON vessels(mmsi);	
	`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS vessel_routes (
        id SERIAL PRIMARY KEY,
        vessel_id INT REFERENCES vessels(id),
        ocean_product_id INT REFERENCES ocean_products(id),
        route_type TEXT, -- 'DEPARTURE' or 'TRANSPORT_LEG'
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

type Vessel struct {
	ID                int       `db:"id"`
	IMONumber         string    `db:"imo_number"`
	MMSI              string    `db:"mmsi"`
	Name              string    `db:"name"`
	IsTracked         bool      `db:"is_tracked"`
	CarrierCode       string    `db:"carrier_code"`
	AppearanceCount   int       `db:"appearance_count"`
	LastSeen          time.Time `db:"last_seen"`
	CreatedAt         time.Time `db:"created_at"`
	LastKnownPosition []float64 `db:"last_known_position"`
}

type OceanProduct struct {
	ID                          int       `db:"id"`
	CarrierProductID            string    `db:"carrier_product_id"`
	ProductValidToDate          time.Time `db:"product_valid_to_date"`
	ProductValidFromDate        time.Time `db:"product_valid_from_date"`
	OriginCity                  string    `db:"origin_city"`
	OriginName                  string    `db:"origin_name"`
	OriginCountry               string    `db:"origin_country"`
	OriginPortUNLoCode          string    `db:"origin_port_un_lo_code"`
	OriginCarrierSiteGeoID      string    `db:"origin_carrier_site_geo_id"`
	OriginCarrierCityGeoID      string    `db:"origin_carrier_city_geo_id"`
	DestinationCity             string    `db:"destination_city"`
	DestinationName             string    `db:"destination_name"`
	DestinationCountry          string    `db:"destination_country"`
	DestinationPortUNLoCode     string    `db:"destination_port_un_lo_code"`
	DestinationCarrierSiteGeoID string    `db:"destination_carrier_site_geo_id"`
	DestinationCarrierCityGeoID string    `db:"destination_carrier_city_geo_id"`
	DepartureVesselCarrierCode  string    `db:"departure_vessel_carrier_code"`
	DepartureVesselName         string    `db:"departure_vessel_name"`
	DepartureVesselIMONumber    string    `db:"departure_vessel_imo_number"`
	DepartureVesselMMSI         string    `db:"departure_vessel_mmsi"`
	DepartureDateTime           time.Time `db:"departure_date_time"`
	ArrivalDateTime             time.Time `db:"arrival_date_time"`
	TransitTime                 int       `db:"transit_time"`
	CreatedAt                   time.Time `db:"created_at"`
}

type TransportLeg struct {
	ID                          int       `db:"id"`
	OceanProductID              int       `db:"ocean_product_id"`
	DepartureDateTime           time.Time `db:"departure_date_time"`
	ArrivalDateTime             time.Time `db:"arrival_date_time"`
	VesselCarrierCode           string    `db:"vessel_carrier_code"`
	VesselName                  string    `db:"vessel_name"`
	VesselIMONumber             string    `db:"vessel_imo_number"`
	VesselMMSI                  string    `db:"vessel_mmsi"`
	OriginCity                  string    `db:"origin_city"`
	OriginName                  string    `db:"origin_name"`
	OriginCountry               string    `db:"origin_country"`
	OriginPortUNLoCode          string    `db:"origin_port_un_lo_code"`
	OriginCarrierSiteGeoID      string    `db:"origin_carrier_site_geo_id"`
	OriginCarrierCityGeoID      string    `db:"origin_carrier_city_geo_id"`
	DestinationCity             string    `db:"destination_city"`
	DestinationName             string    `db:"destination_name"`
	DestinationCountry          string    `db:"destination_country"`
	DestinationPortUNLoCode     string    `db:"destination_port_un_lo_code"`
	DestinationCarrierSiteGeoID string    `db:"destination_carrier_site_geo_id"`
	DestinationCarrierCityGeoID string    `db:"destination_carrier_city_geo_id"`
	CreatedAt                   time.Time `db:"created_at"`
}

type VesselPosition struct {
	ID        int       `db:"id"`
	VesselID  int       `db:"vessel_id"`
	MMSI      string    `db:"mmsi"`
	Latitude  float64   `db:"latitude"`
	Longitude float64   `db:"longitude"`
	Timestamp time.Time `db:"timestamp"`
	CreatedAt time.Time `db:"created_at"`
}

type VesselRoute struct {
	ID             int       `db:"id"`
	VesselID       int       `db:"vessel_id"`
	OceanProductID int       `db:"ocean_product_id"`
	RouteType      string    `db:"route_type"`
	CreatedAt      time.Time `db:"created_at"`
}
