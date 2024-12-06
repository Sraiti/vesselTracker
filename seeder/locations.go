package seeder

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
)

type SeederMetrics struct {
	TotalRecords       int64
	ValidCoordinates   int64
	InvalidCoordinates int64
	SkippedRecords     int
	ProcessingDuration time.Duration
	DatabaseDuration   time.Duration
	BatchSize          int
}

type Location struct {
	UnLoCode       string
	Name           string
	CountryCode    string
	IsPort         bool
	IsAirport      bool
	IsTrainStation bool
	Latitude       float64
	Longitude      float64
	HasCoordinates bool
}

func SeedLocations(db *sql.DB, batchSize int) (*SeederMetrics, error) {
	startTime := time.Now()
	metrics := &SeederMetrics{BatchSize: batchSize}

	// Process CSV
	locations, err := ProcessCSV("code-list.csv", metrics)
	if err != nil {
		return metrics, fmt.Errorf("failed to process CSV: %w", err)
	}

	// Record processing duration
	metrics.ProcessingDuration = time.Since(startTime)
	log.Printf("CSV Processing completed: Records=%d, Valid Coordinates=%d, Processing Time=%v",
		metrics.TotalRecords, metrics.ValidCoordinates, metrics.ProcessingDuration)

	log.Printf("Seeding %d locations", len(locations))
	// Start database operations
	dbStart := time.Now()
	totalLocations := len(locations)
	processedLocations := 0

	// Process in batches
	for i := 0; i < len(locations); i += batchSize {
		end := i + batchSize
		if end > len(locations) {
			end = len(locations)
		}

		batch := locations[i:end]
		if err := bulkInsert(db, batch); err != nil {
			return metrics, fmt.Errorf("failed to insert batch %d-%d: %w", i, end, err)
		}

		processedLocations += len(batch)
		log.Printf("Progress: %d/%d locations inserted (%.1f%%)",
			processedLocations, totalLocations,
			float64(processedLocations)/float64(totalLocations)*100)
	}

	metrics.DatabaseDuration = time.Since(dbStart)
	log.Printf("Seeding completed: %+v", metrics)
	return metrics, nil
}

func ProcessCSV(filePath string, metrics *SeederMetrics) ([]Location, error) {
	fileStart := time.Now()
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()
	log.Printf("File open time: %v", time.Since(fileStart))

	bufferedReader := bufio.NewReaderSize(file, 256*1024)
	reader := csv.NewReader(bufferedReader)

	// Optimize CSV reader settings
	reader.ReuseRecord = true
	reader.FieldsPerRecord = 12
	reader.LazyQuotes = true
	// Skip header
	headerStart := time.Now()
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}
	log.Printf("Header read time: %v", time.Since(headerStart))

	// Read all records
	readStart := time.Now()
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records: %w", err)
	}
	log.Printf("CSV read time: %v for %d records", time.Since(readStart), len(records))

	numWorkers := 4
	chunkSize := len(records) / numWorkers
	log.Printf("Total records: %d, Workers: %d, Chunk size: %d",
		len(records), numWorkers, chunkSize)

	// Create channels
	locationChan := make(chan []Location, numWorkers)
	errorChan := make(chan error, numWorkers)

	// Process chunks in parallel
	processStart := time.Now()
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == numWorkers-1 {
			end = len(records) // Last chunk takes remaining records
		}

		go func(workerId int, chunk [][]string) {
			workerStart := time.Now()
			log.Printf("Worker %d starting to process records %d to %d",
				workerId, start, end)

			// Pre-allocate slice with exact capacity needed
			chunkLocations := make([]Location, 0, len(chunk))
			var coordParseTime time.Duration
			var functionParseTime time.Duration

			for _, record := range chunk {
				atomic.AddInt64(&metrics.TotalRecords, 1)

				coordStart := time.Now()
				lat, lon, valid := parseCoordinates(record[10])
				coordParseTime += time.Since(coordStart)

				if valid {
					atomic.AddInt64(&metrics.ValidCoordinates, 1)
				} else {
					atomic.AddInt64(&metrics.InvalidCoordinates, 1)
				}

				funcStart := time.Now()
				functionCode := record[7]
				location := Location{
					UnLoCode:       record[1] + record[2],
					Name:           record[3],
					CountryCode:    record[1],
					IsPort:         strings.Contains(functionCode, "1"),
					IsAirport:      strings.Contains(functionCode, "4"),
					IsTrainStation: strings.Contains(functionCode, "2"),
					Latitude:       lat,
					Longitude:      lon,
					HasCoordinates: valid,
				}
				functionParseTime += time.Since(funcStart)

				chunkLocations = append(chunkLocations, location)
			}

			log.Printf("Worker %d completed in %v. Stats: records=%d, coordParse=%v, funcParse=%v",
				workerId, time.Since(workerStart), len(chunkLocations),
				coordParseTime, functionParseTime)

			locationChan <- chunkLocations
			errorChan <- nil
		}(i, records[start:end])
	}

	// Pre-allocate the final slice with exact total size
	collectStart := time.Now()
	allLocations := make([]Location, 0, len(records))

	// Single-phase collection with pre-allocated capacity
	for i := 0; i < numWorkers; i++ {
		chunkLocations := <-locationChan
		if err := <-errorChan; err != nil {
			return nil, err
		}
		log.Printf("Collected %d records from worker result %d",
			len(chunkLocations), i)
		allLocations = append(allLocations, chunkLocations...)
	}

	totalTime := time.Since(fileStart)
	log.Printf("Processing summary:\n"+
		"- Total time: %v\n"+
		"- File operations: %v\n"+
		"- Parallel processing: %v\n"+
		"- Collection time: %v\n"+
		"- Records processed: %d\n"+
		"- Processing rate: %.2f records/sec\n"+
		"- Valid coordinates: %d\n"+
		"- Invalid coordinates: %d",
		totalTime,
		readStart.Sub(fileStart),
		time.Since(processStart),
		time.Since(collectStart),
		len(allLocations),
		float64(len(allLocations))/totalTime.Seconds(),
		metrics.ValidCoordinates,
		metrics.InvalidCoordinates)

	return allLocations, nil
}

func parseCoordinates(coord string) (float64, float64, bool) {

	// Quick initial checks
	if len(coord) < 11 { // Minimum length for two coordinates
		return 0, 0, false
	}

	parts := strings.Split(coord, " ")
	if len(parts) != 2 {
		return 0, 0, false
	}

	// Length checks before parsing
	if len(parts[0]) != 5 || len(parts[1]) != 6 { // "DDMMN" and "DDMMMW" formats
		return 0, 0, false
	}

	// Check last characters first (faster than parsing numbers)
	latDir := parts[0][4]
	lonDir := parts[1][5]
	if !((latDir == 'N' || latDir == 'S') && (lonDir == 'E' || lonDir == 'W')) {
		return 0, 0, false
	}
	// Parse latitude (optimized)
	latDeg := float64((coord[0]-'0')*10 + (coord[1] - '0'))
	latMin := float64((coord[2]-'0')*10 + (coord[3] - '0'))
	lat := latDeg + (latMin / 60.0)
	if latDir == 'S' {
		lat = -lat
	}

	// Parse longitude (optimized)
	lonDeg := float64((coord[6]-'0')*100 + (coord[7]-'0')*10 + (coord[8] - '0'))
	lonMin := float64((coord[9]-'0')*10 + (coord[10] - '0'))
	lon := lonDeg + (lonMin / 60.0)
	if lonDir == 'W' {
		lon = -lon
	}

	return lat, lon, true
}

func parseFloat(s string) float64 {
	val := 0.0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			val = val*10 + float64(ch-'0')
		}
	}
	return val
}

func bulkInsert(db *sql.DB, locations []Location) error {

	tx, err := db.Begin()

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	// Create temp table
	_, err = tx.Exec(`
	   CREATE TEMPORARY TABLE temp_locations (
            unlocode TEXT NOT NULL,
            name TEXT NOT NULL,
            country_code TEXT NOT NULL,
            location GEOGRAPHY(POINT, 4326),
            is_port BOOLEAN DEFAULT FALSE,
            is_airport BOOLEAN DEFAULT FALSE,
            is_train_station BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        ) ON COMMIT DROP
	`)
	if err != nil {
		return fmt.Errorf("failed to create temp table: %w", err)
	}

	// Prepare COPY into temp table
	stmt, err := tx.Prepare(pq.CopyIn(
		"temp_locations",
		"unlocode", "name", "country_code",
		"location", "is_port", "is_airport", "is_train_station",
	))
	if err != nil {
		return fmt.Errorf("failed to prepare COPY statement: %w", err)
	}
	defer stmt.Close()

	for _, loc := range locations {
		var point interface{}
		if loc.HasCoordinates {
			point = fmt.Sprintf("SRID=4326;POINT(%f %f)", loc.Longitude, loc.Latitude)
		}

		_, err := stmt.Exec(
			loc.UnLoCode,
			loc.Name,
			loc.CountryCode,
			point,
			loc.IsPort,
			loc.IsAirport,
			loc.IsTrainStation,
		)
		if err != nil {
			return fmt.Errorf("failed to COPY location %s: %w", loc.UnLoCode, err)
		}
	}

	if _, err := stmt.Exec(); err != nil {
		return fmt.Errorf("failed to flush COPY buffer: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO locations (
			unlocode, name, country_code,
			location, is_port, is_airport, is_train_station
		)
		SELECT 
			unlocode, name, country_code,
			location, is_port, is_airport, is_train_station
		FROM temp_locations
		ON CONFLICT (unlocode, name) DO UPDATE SET
			country_code = EXCLUDED.country_code,
			location = COALESCE(EXCLUDED.location, locations.location),
			is_port = EXCLUDED.is_port,
			is_airport = EXCLUDED.is_airport,
			is_train_station = EXCLUDED.is_train_station
	`)
	if err != nil {
		return fmt.Errorf("failed to insert from temp table: %w", err)
	}

	return tx.Commit()
}
