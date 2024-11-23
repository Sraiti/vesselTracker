package db

import (
	"database/sql"
	"os"

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
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
