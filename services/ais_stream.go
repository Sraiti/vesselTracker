package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/Sraiti/vesselTracker/db"
	"github.com/Sraiti/vesselTracker/models"
	aisstream "github.com/aisstream/ais-message-models/golang/aisStream"
	"github.com/gorilla/websocket"
)

// Simple message type if we don't want to use aisstream package
type AISMessage struct {
	MessageType string          `json:"messageType"`
	Message     json.RawMessage `json:"message"`
	MetaData    json.RawMessage `json:"metaData"`
}

type AISStreamManager struct {
	apiKey string
	db     *sql.DB
	conn   *websocket.Conn
	stats  struct {
		messagesReceived uint64
		messagesSaved    uint64
		errors           uint64
		reconnects       uint64
	}
	startTime time.Time
}

func NewAISStreamManager(apiKey string, database *sql.DB) *AISStreamManager {
	return &AISStreamManager{
		apiKey:    apiKey,
		db:        database,
		startTime: time.Now(),
	}
}

func (a *AISStreamManager) StartStreaming(mmsis []string) error {
	a.logEvent("startup", "Starting AIS stream manager", map[string]interface{}{
		"mmsi_count": len(mmsis),
		"mmsis":      mmsis,
	})

	// Start statistics logger
	go a.logStatsPeriodically()

	return a.connect(mmsis)
}

func (a *AISStreamManager) connect(mmsis []string) error {
	url := "wss://stream.aisstream.io/v0/stream"

	a.logEvent("connection_attempt", "Attempting to connect to AIS stream", map[string]interface{}{
		"url": url,
	})

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		atomic.AddUint64(&a.stats.errors, 1)
		a.logEvent("connection_error", "Failed to connect", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to connect: %v", err)
	}

	a.conn = conn
	a.logEvent("connection_success", "Successfully connected to AIS stream", nil)

	// Subscribe to vessels
	subMsg := map[string]interface{}{
		"APIKey":          a.apiKey,
		"BoundingBoxes":   [][][]float64{{{-90.0, -180.0}, {90.0, 180.0}}},
		"FiltersShipMMSI": mmsis,
	}

	if err := conn.WriteJSON(subMsg); err != nil {
		atomic.AddUint64(&a.stats.errors, 1)
		a.logEvent("subscription_error", "Failed to subscribe", map[string]interface{}{
			"error": err.Error(),
		})
		conn.Close()
		return fmt.Errorf("failed to subscribe: %v", err)
	}

	a.logEvent("subscription_success", "Successfully subscribed to AIS stream", map[string]interface{}{
		"mmsi_count": len(mmsis),
	})

	_, err = db.UpdateTrackedVessels(a.db, mmsis)

	// Update tracked vessels in database
	if err != nil {
		atomic.AddUint64(&a.stats.errors, 1)
		a.logEvent("database_error", "Failed to update tracked vessels", map[string]interface{}{
			"error": err.Error(),
		})
		// Don't return error - continue with connection even if DB update fails
	} else {
		a.logEvent("database_success", "Updated tracked vessels in database", map[string]interface{}{
			"count": len(mmsis),
		})
	}
	go a.handleMessages()
	return nil
}

func (a *AISStreamManager) handleMessages() {
	for {
		_, message, err := a.conn.ReadMessage()
		if err != nil {
			atomic.AddUint64(&a.stats.errors, 1)
			a.logEvent("websocket_error", "Error reading message", map[string]interface{}{
				"error": err.Error(),
			})

			// Attempt to reconnect
			a.reconnect()
			return
		}

		atomic.AddUint64(&a.stats.messagesReceived, 1)

		// Parse message for logging
		var msg aisstream.AisStreamMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			atomic.AddUint64(&a.stats.errors, 1)
			a.logEvent("parse_error", "Failed to parse message", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		mmsi := fmt.Sprintf("%d", int64(msg.MetaData["MMSI"].(float64)))

		go a.saveMessageToFile(message, msg)

		log.Printf("Timestamp: %s", msg.MetaData["time_utc"])

		var timestamp models.CustomTime
		if timestampStr, ok := msg.MetaData["time_utc"]; ok {
			parsedTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timestampStr.(string))
			if err != nil {
				log.Printf("Error parsing timestamp: %s", err)
			}

			log.Printf("Parsed time: %s", parsedTime)
			timestamp = models.CustomTime{Time: parsedTime}
		}

		log.Printf("Timestamp: %s", timestamp)

		switch msg.MessageType {
		case aisstream.POSITION_REPORT:
			log.Printf("Position report received for mmsi %s", mmsi)

			var positionReport aisstream.PositionReport
			positionReport = *msg.Message.PositionReport

			log.Println("Position report:")
			log.Printf("Position report for MMSI %s: %+v", mmsi, positionReport)

			go db.InsertPositionReport(a.db, mmsi, positionReport, timestamp)
		}

	}
}

func (a *AISStreamManager) saveMessageToFile(message []byte, msg aisstream.AisStreamMessage) error {
	// Create directory structure for current time
	now := time.Now()
	dirPath := filepath.Join(
		"ais_data",
		now.Format("2006-01-02"),
		now.Format("15"),
	)

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		atomic.AddUint64(&a.stats.errors, 1)
		a.logEvent("filesystem_error", "Error creating directory", map[string]interface{}{
			"error": err.Error(),
			"path":  dirPath,
		})
		return err
	}

	// Save message to file
	filename := filepath.Join(dirPath, fmt.Sprintf("%d.json", now.UnixNano()))
	if err := os.WriteFile(filename, message, 0644); err != nil {
		atomic.AddUint64(&a.stats.errors, 1)
		a.logEvent("filesystem_error", "Error writing file", map[string]interface{}{
			"error":    err.Error(),
			"filename": filename,
		})
		return err
	}

	atomic.AddUint64(&a.stats.messagesSaved, 1)

	// Log message details
	a.logEvent("message_saved", "Successfully saved AIS message", map[string]interface{}{
		"message_type": msg.MessageType,
		"filename":     filename,
	})

	return nil
}

func (a *AISStreamManager) reconnect() {
	atomic.AddUint64(&a.stats.reconnects, 1)
	a.logEvent("reconnection", "Attempting to reconnect", nil)

	// Implementation of reconnection logic here
	// You might want to add exponential backoff, etc.
}
