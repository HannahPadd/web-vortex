package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type SongData struct {
	Title  string     `json:"title"`
	Artist string     `json:"artist"`
	BPMS   string     `json:"bpms"`
	Charts []ChartData `json:"charts"`
}

type ChartData struct {
	ChartHeader   string     `json:"chartHeader"`
	Type          string     `json:"type"`
	Tag           string     `json:"tag"`
	DifficultyTag string     `json:"difficultyTag"`
	Difficulty    string     `json:"difficulty"`
	Notes         [][]string `json:"notes"`
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // Max file size: 10MB

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	tempFile, err := ioutil.TempFile("uploads", "upload-*.sm")
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the file", http.StatusInternalServerError)
		return
	}

	tempFile.Write(fileBytes)
	songData, err := parseSMFile(string(fileBytes))
	if err != nil {
		http.Error(w, "Error parsing the file", http.StatusInternalServerError)
		return
	}

	// Test output
	fmt.Printf("Title: %s\n", songData.Title)
	fmt.Printf("Artist: %s\n", songData.Artist)
	fmt.Printf("BPMS: %s\n", songData.BPMS)
	for _, chart := range songData.Charts {
		fmt.Printf("\nChart Header: %s\n", chart.ChartHeader)
		fmt.Printf("Type: %s\n", chart.Type)
		fmt.Printf("Tag: %s\n", chart.Tag)
		fmt.Printf("Difficulty: %s\n", chart.Difficulty)
		fmt.Printf("Difficulty Tag: %s\n", chart.DifficultyTag)
		fmt.Println("Notes:")
		for _, noteGroup := range chart.Notes {
			fmt.Printf("    %s\n", strings.Join(noteGroup, ", "))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(songData)
}

func parseSMFile(data string) (*SongData, error) {
	scanner := bufio.NewScanner(strings.NewReader(data))
	songData := &SongData{}
	var currentType, currentTag, currentDifficulty, currentDifficultyTag string
	var notes [][]string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#TITLE:") {
			songData.Title = strings.TrimSpace(strings.TrimPrefix(line, "#TITLE:"))
		} else if strings.HasPrefix(line, "#ARTIST:") {
			songData.Artist = strings.TrimSpace(strings.TrimPrefix(line, "#ARTIST:"))
		} else if strings.HasPrefix(line, "#BPMS:") {
			songData.BPMS = strings.TrimSpace(strings.TrimPrefix(line, "#BPMS:"))
		} else if strings.HasPrefix(line, "#NOTES:") {
			if len(notes) > 0 {
				songData.Charts = append(songData.Charts, ChartData{
					Type:          currentType,
					Tag:           currentTag,
					Difficulty:    currentDifficulty,
					DifficultyTag: currentDifficultyTag,
					Notes:         notes,
				})
				notes = [][]string{} // Reset notes for the next chart
			}

			// Parse metadata from #NOTES line
			fields := strings.SplitN(line, ":", 2)
			if len(fields) == 2 {
				metaData := strings.TrimSpace(fields[1])
				metaFields := strings.Split(metaData, ":")

				if len(metaFields) >= 4 {
					currentType = strings.TrimSpace(metaFields[0])
					currentTag = strings.TrimSpace(metaFields[1])
					currentDifficulty = strings.TrimSpace(metaFields[2])
					currentDifficultyTag = strings.TrimSpace(metaFields[3])
				}
			}
		} else if strings.HasPrefix(line, ",") {
			// Parse note lines
			noteGroup := strings.Split(strings.TrimSpace(line), ",")
			notes = append(notes, noteGroup)
		}
	}

	// Append the last chart
	if len(notes) > 0 {
		songData.Charts = append(songData.Charts, ChartData{
			Type:          currentType,
			Tag:           currentTag,
			Difficulty:    currentDifficulty,
			DifficultyTag: currentDifficultyTag,
			Notes:         notes,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return songData, nil
}

// convertNotesTo2DArray converts a slice of note lines into a 2D array of strings
func convertNotesTo2DArray(notes []string) [][]string {
	result := make([][]string, len(notes))
	for i, note := range notes {
		result[i] = []string{note}
	}
	return result
}

func trimSuffixColon(s string) string {
	return strings.TrimSuffix(strings.TrimSpace(s), ":")
}


func main() {
	router := mux.NewRouter()
	router.HandleFunc("/upload", uploadFileHandler).Methods("POST")

	handler := cors.Default().Handler(router)
	fmt.Println("Server is running on port 5000")
	http.ListenAndServe(":5000", handler)
}
