package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Sraiti/vesselTracker/models"

	aisstream "github.com/aisstream/ais-message-models/golang/aisStream"
)

func getFileContent(filePath string) VesselMessageSummary {

	log.Println("ais_data/" + filePath)

	content, err := os.ReadFile("ais_data/" + filePath)

	if err != nil {
		log.Fatal(err)
	}

	var packet aisstream.AisStreamMessage

	err = json.Unmarshal(content, &packet)

	if err != nil {
		log.Fatal(err)
	}

	log.Println(packet.MessageType)

	timeStr := packet.MetaData["time_utc"].(string)

	log.Println(timeStr)
	parsedTime, _ := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr)
	log.Println(parsedTime)

	return VesselMessageSummary{
		EventTypes: string(packet.MessageType),
		MMSIs:      packet.MetaData["MMSI_String"].(float64),
		TimeStamp:  models.CustomTime{Time: parsedTime},
	}

}

func FilesExaminerHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vesselInfo := map[string]VesselsMessagesSummary{}

		/// reading the files that exists in the folder of ais data and get all the unique mmsi numbers and all the messages types we got and last date we got an event fora each mmsi .
		files, err := os.ReadDir("ais_data")

		if err != nil {
			log.Println(err)
		}
		fileNames := []struct {
			Name string
		}{}

		for _, file := range files {

			if file.IsDir() {
				log.Println("Reading directory:", file.Name())
				files, err := os.ReadDir("ais_data/" + file.Name())
				if err != nil {
					log.Fatal(err)
				}
				for _, subFile := range files {

					if subFile.IsDir() {

						subSubFile, err := os.ReadDir("ais_data/" + file.Name() + "/" + subFile.Name())
						if err != nil {
							log.Fatal(err)
						}
						for _, line := range subSubFile {

							fileNames = append(fileNames, struct {
								Name string
							}{Name: line.Name()})

							summary := getFileContent(file.Name() + "/" + subFile.Name() + "/" + line.Name())

							log.Println(summary.TimeStamp)

							vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")] = VesselsMessagesSummary{
								EventTypes: func(existing []string, new string) []string {
									for _, v := range existing {
										if v == new {
											return existing
										}
									}
									return append(existing, new)
								}(vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")].EventTypes, summary.EventTypes),
								MMSIs:     []float64{summary.MMSIs},
								LastEvent: summary.TimeStamp,
								Count:     vesselInfo[strings.Trim(fmt.Sprintf("%d", int(summary.MMSIs)), " ")].Count + 1,
							}
						}
					}

				}
			}
		}

		json.NewEncoder(w).Encode(vesselInfo)
	}
}
