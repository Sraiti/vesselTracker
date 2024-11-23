package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// logEvent provides structured logging for different event types
func (a *AISStreamManager) logEvent(eventType string, msg string, extra map[string]interface{}) {
	event := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339),
		"event_type": eventType,
		"message":    msg,
		"uptime":     time.Since(a.startTime).String(),
		"stats": map[string]uint64{
			"messages_received": atomic.LoadUint64(&a.stats.messagesReceived),
			"messages_saved":    atomic.LoadUint64(&a.stats.messagesSaved),
			"errors":            atomic.LoadUint64(&a.stats.errors),
			"reconnects":        atomic.LoadUint64(&a.stats.reconnects),
		},
	}

	// Add any extra fields
	for k, v := range extra {
		event[k] = v
	}

	// Convert to JSON for structured logging
	jsonEvent, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		log.Printf("Error marshaling log event: %v", err)
		return
	}

	// Write to both console and log file
	log.Printf("%s\n", string(jsonEvent))

	// Also write to daily log file
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Error creating log directory: %v", err)
		return
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("ais_stream_%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "%s\n", string(jsonEvent))
}

func (a *AISStreamManager) logStatsPeriodically() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		a.logEvent("statistics", "Periodic statistics update", map[string]interface{}{
			"messages_per_minute": float64(atomic.LoadUint64(&a.stats.messagesReceived)) / time.Since(a.startTime).Minutes(),
			"save_success_rate":   float64(atomic.LoadUint64(&a.stats.messagesSaved)) / float64(atomic.LoadUint64(&a.stats.messagesReceived)) * 100,
		})
	}
}
