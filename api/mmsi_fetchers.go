package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

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

// getVesselMMSIbyIMO makes the HTTP request and uses the parser
func getVesselMMSIbyIMO(imo string) (string, error) {
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

func getVesselMMSIFromMarineTraffic(imo string) (string, error) {
	start := time.Now()
	log.Printf("Fetching MMSI for IMO %s", imo)
	url := fmt.Sprintf("https://www.marinetraffic.com/en/ais/details/ships/imo:%s", imo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Exact headers from the browser
	// Only set the essential Cloudflare cookies
	req.Header.Set("Cookie", "__cf_bm=UzPwV_z_ygO..kAYXXkb7Qw5SbjOSB1rR4tjhBRK9mY-1732141891-1.0.1.1-JlJaGpOQVxxcwJR9hylnr5a7n43iATQ_ZfFThd5IBjFW_vBcBeSAKl7G7Fwruk5fY032H2gUZw9CUP7nfFGE7A; _cfuvid=14.eku0poDMh9wfTcL5eEhalRLD3gJbp_D2M5jwtATE-1732141891466-0.0.1.1-604800000")

	// Add the cookies that are required

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, resp.Body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	bodyStr := string(body)

	// Extract MMSI from title
	titleStart := strings.Index(bodyStr, "<title>")
	if titleStart == -1 {
		return "", fmt.Errorf("title tag not found")
	}

	titleEnd := strings.Index(bodyStr[titleStart:], "</title>")
	if titleEnd == -1 {
		return "", fmt.Errorf("closing title tag not found")
	}

	title := bodyStr[titleStart+7 : titleStart+titleEnd] // +7 to skip "<title>"

	// Extract MMSI from format: "Ship NAME (Type) ... - IMO NUMBER, MMSI NUMBER, Call sign XXX"
	mmsiIndex := strings.Index(title, "MMSI ")
	if mmsiIndex == -1 {
		return "", fmt.Errorf("MMSI not found in title")
	}

	mmsiStart := mmsiIndex + len("MMSI ")
	mmsiEnd := strings.Index(title[mmsiStart:], ",")
	if mmsiEnd == -1 {
		// If no comma, try to find Call sign
		mmsiEnd = strings.Index(title[mmsiStart:], " Call sign")
		if mmsiEnd == -1 {
			return "", fmt.Errorf("MMSI format invalid")
		}
	}

	mmsi := strings.TrimSpace(title[mmsiStart : mmsiStart+mmsiEnd])
	log.Printf("Successfully extracted MMSI %s for IMO %s in %v", mmsi, imo, time.Since(start))

	return mmsi, nil
}
