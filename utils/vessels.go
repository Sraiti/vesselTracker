package utils

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
)

func addVesselToSet(imoSet map[string]db.Vessel, maerskVessel models.Vessel) {
	if maerskVessel.VesselIMONumber != "" {
		imoSet[maerskVessel.VesselIMONumber] = db.Vessel{
			IMONumber:   maerskVessel.VesselIMONumber,
			Name:        maerskVessel.VesselName,
			CarrierCode: maerskVessel.CarrierVesselCode,
		}
	}
}

func CollectUniqueVessels(data models.MaerskPointToPoint) map[string]db.Vessel {
	imoSet := make(map[string]db.Vessel)
	for _, product := range data.OceanProducts {
		for _, schedule := range product.TransportSchedules {
			addVesselToSet(imoSet, schedule.FirstDepartureVessel)
			for _, leg := range schedule.TransportLegs {
				addVesselToSet(imoSet, leg.Transport.Vessel)
			}
		}
	}
	return imoSet
}

type vesselLookup struct {
	imo       string
	vessel    db.Vessel
	needsHTTP bool
}

type VesselFetcher struct {
	DB        *sql.DB
	MmsiCache map[string]db.Vessel
}

// Fetch vessel data (MMSI and positions)
func (vf *VesselFetcher) FetchVesselData(imoSet map[string]db.Vessel) map[string]db.Vessel {

	imos := make([]string, 0, len(imoSet))
	for imo := range imoSet {
		imos = append(imos, imo)
	}

	// Phase 1: Quick DB lookups

	dbVessels, err := db.GetVesselsByIMOs(vf.DB, imos)
	if err != nil {
		log.Printf("Error fetching vessels from DB: %v", err)
	}

	var httpNeeded []string

	for imo, vessel := range imoSet {
		if dbVessel, exists := dbVessels[imo]; exists {
			vf.MmsiCache[imo] = dbVessel
		} else {
			httpNeeded = append(httpNeeded, imo)
			vf.MmsiCache[imo] = vessel // Store original vessel data
		}
	}

	// 4. Perform HTTP lookups only for missing vessels
	if len(httpNeeded) > 0 {
		vf.performHTTPLookups(httpNeeded)
	}

	//vf.enrichVesselsWithPositions(vf.MmsiCache)

	return vf.MmsiCache
}

// func (vf *VesselFetcher) enrichVesselsWithPositions(vessels map[string]db.Vessel) {
// 	// Create a buffered channel for position results
// 	type positionResult struct {
// 		imo      string
// 		position []float64
// 		err      error
// 	}

// 	posChan := make(chan positionResult, len(vessels))

// 	// Launch position fetches in parallel
// 	for imo := range vessels {
// 		go func(imo string) {
// 			position, err := db.GetVesselLastKnownPosition(vf.DB, imo)
// 			posChan <- positionResult{
// 				imo:      imo,
// 				position: position,
// 				err:      err,
// 			}
// 		}(imo)
// 	}

// 	// Collect results
// 	for i := 0; i < len(vessels); i++ {
// 		result := <-posChan
// 		if result.err == nil && result.position != nil {
// 			if vessel, exists := vessels[result.imo]; exists {
// 				vessel.LastKnownPosition = result.position
// 				vessels[result.imo] = vessel
// 				log.Printf("Found position for vessel IMO %s: %v", result.imo, result.position)
// 			}
// 		}
// 	}
// }

func (vf *VesselFetcher) performHTTPLookups(httpNeeded []string) {
	if len(httpNeeded) == 0 {
		return
	}

	// Create channels for job distribution and result collection
	jobs := make(chan string, len(httpNeeded))
	results := make(chan struct {
		imo  string
		mmis string
		err  error
	}, len(httpNeeded))

	// Create worker pool
	const maxWorkers = 3 // Limit concurrent HTTP requests to avoid rate limiting
	var wg sync.WaitGroup

	// Launch workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for imo := range jobs {
				start := time.Now()
				log.Printf("Worker %d starting HTTP lookup for IMO: %s", workerID, imo)

				// Add retry logic for HTTP requests
				var mmsi string
				var err error
				for retries := 0; retries < 3; retries++ {
					if retries > 0 {
						time.Sleep(time.Duration(retries) * time.Second)
						log.Printf("Worker %d retrying HTTP lookup for IMO %s (attempt %d)",
							workerID, imo, retries+1)
					}

					mmsi, err = getVesselMMSIFromHTTP(imo)
					if err == nil {
						break
					}
				}

				duration := time.Since(start)
				if err != nil {
					log.Printf("Worker %d failed to fetch MMSI for IMO %s after retries: %v (took: %v)",
						workerID, imo, err, duration)
				} else {
					log.Printf("Worker %d successfully fetched MMSI for IMO %s (took: %v)",
						workerID, imo, duration)
				}

				results <- struct {
					imo  string
					mmis string
					err  error
				}{imo, mmsi, err}
			}
			log.Printf("Worker %d finished processing", workerID)
		}(i)
	}

	// Send jobs to workers
	go func() {
		for _, imo := range httpNeeded {
			jobs <- imo
		}
		close(jobs) // Close jobs channel after all work is distributed
	}()

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(results) // Close results channel after all workers are done
	}()

	// Collect results and update cache
	successCount := 0
	failureCount := 0
	start := time.Now()

	for result := range results {
		if result.err == nil && result.mmis != "" {
			// Update the vessel in cache with the new MMSI
			if vessel, exists := vf.MmsiCache[result.imo]; exists {
				vessel.MMSI = result.mmis
				vf.MmsiCache[result.imo] = vessel
				successCount++
			}
		} else {
			failureCount++
		}
	}

	duration := time.Since(start)
	log.Printf("HTTP lookups completed: %d successful, %d failed (took: %v)",
		successCount, failureCount, duration)
}

// getVesselMMSIFromHTTP handles the HTTP lookup separately
func getVesselMMSIFromHTTP(imo string) (string, error) {

	start := time.Now()
	defer func() {
		log.Printf("Total MMSI fetch for IMO %s took: %v", imo, time.Since(start))
	}()

	url := fmt.Sprintf("https://www.vesselfinder.com/vessels/details/%s", imo)

	// Create request with detailed timing
	reqStart := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	log.Printf("Request creation took: %v", time.Since(reqStart))

	// Add browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Make the request with timing
	httpStart := time.Now()
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	log.Printf("HTTP request for IMO %s took: %v", imo, time.Since(httpStart))

	// Read response body with timing
	readStart := time.Now()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}
	bodyStr := string(body)
	log.Printf("Response body reading took: %v", time.Since(readStart))

	// Parse the HTML
	return parseMMSIFromHTML(bodyStr)

}

// parseMMSIFromHTML extracts MMSI from the vesselfinder HTML response
func parseMMSIFromHTML(bodyStr string) (string, error) {
	parseStart := time.Now()
	defer func() {
		log.Printf("Total HTML parsing took: %v", time.Since(parseStart))
	}()

	// Find IMO/MMSI section
	findStart := time.Now()
	mmsiIndex := strings.Index(bodyStr, "IMO / MMSI")
	if mmsiIndex == -1 {
		return "", fmt.Errorf("could not find MMSI in response")
	}
	log.Printf("Finding IMO/MMSI index took: %v", time.Since(findStart))

	// Extract the MMSI value
	extractStart := time.Now()
	mmsiStart := mmsiIndex + len("IMO / MMSI</td><td class=\"v3 v3np\">")
	mmsiEnd := strings.Index(bodyStr[mmsiStart:], "<")
	if mmsiEnd == -1 {
		return "", fmt.Errorf("could not parse MMSI from response")
	}
	log.Printf("Extracting MMSI boundaries took: %v", time.Since(extractStart))

	// Process the MMSI string
	processStart := time.Now()
	mmsi := strings.TrimSpace(bodyStr[mmsiStart : mmsiStart+mmsiEnd])
	mmsiParts := strings.Split(mmsi, "/")
	if len(mmsiParts) != 2 {
		return "", fmt.Errorf("unexpected MMSI format")
	}
	log.Printf("Processing MMSI string took: %v", time.Since(processStart))

	return strings.TrimSpace(mmsiParts[1]), nil
}
