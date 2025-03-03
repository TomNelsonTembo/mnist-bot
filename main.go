package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// MNISTData represents the structure of MNIST input data
type MNISTData struct {
	Instances [][]float64 `json:"instances"`
}

var (
	mnistSamples [][]float64
	logger       = logrus.New()
)

// loadMNISTData loads MNIST samples from a CSV or JSON file
func loadMNISTData(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Check the file extension
	if len(filename) > 5 && filename[len(filename)-5:] == ".json" {
		// Load JSON file
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&mnistSamples); err != nil {
			return fmt.Errorf("failed to decode JSON: %v", err)
		}
	} else {
		// Load CSV file
		reader := csv.NewReader(file)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read CSV: %v", err)
			}

			var sample []float64
			for _, value := range record {
				pixel, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("failed to parse pixel value: %v", err)
				}
				sample = append(sample, pixel)
			}
			mnistSamples = append(mnistSamples, sample)
		}
	}

	logger.Infof("Loaded %d MNIST samples", len(mnistSamples))
	return nil
}

// generateRandomMNISTData selects a random sample from the predefined list
func generateRandomMNISTData() []float64 {
	index := rand.Intn(len(mnistSamples))
	return mnistSamples[index]
}

// sendData sends MNIST data to the specified API endpoint
func sendData(apiURL string, data []float64, wg *sync.WaitGroup) {
	defer wg.Done()

	// Prepare the request body
	requestBody := MNISTData{
		Instances: [][]float64{data},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Errorf("Error marshaling JSON: %v", err)
		return
	}

	// Send the POST request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Errorf("Error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Request failed with status: %s", resp.Status)
		return
	}

	// Read and log the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Error reading response: %v", err)
		return
	}

	// Ensure the directory exists
	dir := "Assets/Results"
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		logger.Errorf("Error creating directory: %v", err)
		return
	}

	// Append response to file with timestamp
	filename := fmt.Sprintf("%s/responses.txt", dir)
	entry := fmt.Sprintf("Time: %s\nResponse: %s\n\n", time.Now().Format(time.RFC3339), string(responseBody))

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Errorf("Error opening file: %v", err)
		return
	}
	defer file.Close()

	if _, err := file.WriteString(entry); err != nil {
		logger.Errorf("Error writing response to file: %v", err)
		return
	}

	logger.Info("Request sent and response saved successfully")
}

func startBot(apiURL string, interval time.Duration, wg *sync.WaitGroup, quitChan <-chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := generateRandomMNISTData()
			wg.Add(1)
			go sendData(apiURL, data, wg)

		case <-quitChan:
			logger.Println("Bot stopping gracefully...")
			return
		}
	}
}

// startBot starts sending random MNIST data at the specified rate
func main() {
	apiURL := flag.String("api", "", "API endpoint URL")
	numBots := flag.Int("bots", 5, "Number of concurrent bots")
	interval := flag.Int("interval", 30, "Interval between requests (seconds)")
	dataFile := flag.String("data", "./Assets/Data/data.json", "Path to MNIST data file")
	flag.Parse()

	// Load MNIST data
	if err := loadMNISTData(*dataFile); err != nil {
		logger.Fatalf("Failed to load MNIST data: %v", err)
	}

	quitChan := make(chan struct{})
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Printf("Starting %d MNIST bots with %d-second intervals...", *numBots, *interval)

	var wg sync.WaitGroup
	for i := 0; i < *numBots; i++ {
		wg.Add(1)
		go startBot(*apiURL, time.Duration(*interval)*time.Second, &wg, quitChan)
	}

	<-stopChan
	logger.Println("\nReceived stop signal. Shutting down MNIST bots...")
	close(quitChan)
	wg.Wait()
	logger.Println("All bots stopped.")
}
