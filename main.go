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

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/sirupsen/logrus"
)

// MNISTData represents the structure of MNIST input data
type MNISTData struct {
	Instances [][]float64 `json:"instances"`
}

var (
	mnistSamples [][]float64
	logger       = logrus.New()

	// Metrics
	totalRequests   int
	successRequests int
	failedRequests  int
	averageLatency  float64
	latencies       []float64
	metricsMutex    sync.Mutex

	// Logs
	logEntries []string
	logMutex   sync.Mutex
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

	startTime := time.Now()

	// Prepare the request body
	requestBody := MNISTData{
		Instances: [][]float64{data},
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logToWidget(fmt.Sprintf("Error marshaling JSON: %v", err))
		return
	}

	// Send the POST request
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logToWidget(fmt.Sprintf("Error sending request: %v", err))
		metricsMutex.Lock()
		failedRequests++
		metricsMutex.Unlock()
		return
	}
	defer resp.Body.Close()

	latency := time.Since(startTime).Seconds() * 1000 // Convert to milliseconds

	metricsMutex.Lock()
	totalRequests++
	if resp.StatusCode == http.StatusOK {
		successRequests++
		latencies = append(latencies, latency)
		averageLatency = calculateAverageLatency()
	} else {
		failedRequests++
		logToWidget(fmt.Sprintf("Request failed with status: %s", resp.Status))
	}
	metricsMutex.Unlock()

	// Read and log the response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logToWidget(fmt.Sprintf("Error reading response: %v", err))
		return
	}

	// Ensure the directory exists
	dir := "Assets/Results"
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		logToWidget(fmt.Sprintf("Error creating directory: %v", err))
		return
	}

	// Append response to file with timestamp
	filename := fmt.Sprintf("%s/responses.txt", dir)
	entry := fmt.Sprintf("Time: %s\nResponse: %s\n\n", time.Now().Format(time.RFC3339), string(responseBody))

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logToWidget(fmt.Sprintf("Error opening file: %v", err))
		return
	}
	defer file.Close()

	if _, err := file.WriteString(entry); err != nil {
		logToWidget(fmt.Sprintf("Error writing response to file: %v", err))
		return
	}

	logToWidget(fmt.Sprintf("Request sent and response saved successfully (Latency: %.2f ms)", latency))
}

// calculateAverageLatency computes the average latency from the recorded latencies
func calculateAverageLatency() float64 {
	sum := 0.0
	for _, latency := range latencies {
		sum += latency
	}
	return sum / float64(len(latencies))
}

// startBot starts sending random MNIST data at the specified rate
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
			logToWidget("Bot stopping gracefully...")
			return
		}
	}
}

// logToWidget adds a log entry to the log widget
func logToWidget(message string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	logEntries = append(logEntries, message)
	if len(logEntries) > 10 { // Keep only the last 10 log entries
		logEntries = logEntries[1:]
	}
}

// renderMetricsTable creates a terminal-based table to display metrics
func renderMetricsTable() *widgets.Table {
	table := widgets.NewTable()
	table.Rows = [][]string{
		{"Metric", "Value"},
		{"Total Requests", fmt.Sprintf("%d", totalRequests)},
		{"Success Requests", fmt.Sprintf("%d", successRequests)},
		{"Average Latency (ms)", fmt.Sprintf("%.2f", averageLatency)},
		{"Failed Requests", fmt.Sprintf("%d", failedRequests)},
	}
	table.TextStyle = termui.NewStyle(termui.ColorWhite)
	table.Title = "MNIST Bot Metrics"
	table.SetRect(0, 0, 50, 18) // Adjust the height of the table
	return table
}

// renderLogWidget creates a terminal-based widget to display logs
func renderLogWidget() *widgets.List {
	list := widgets.NewList()
	list.Title = "Logs"
	list.Rows = logEntries
	list.TextStyle = termui.NewStyle(termui.ColorWhite)
	list.WrapText = true
	list.SetRect(0, 8, 100, 18) // Position below the metrics table
	return list
}

func main() {
	apiURL := flag.String("api", "", "API endpoint URL")
	numBots := flag.Int("bots", 1, "Number of concurrent bots")
	interval := flag.Int("interval", 1, "Interval between requests (seconds)")
	dataFile := flag.String("data", "./Assets/Data/data.json", "Path to MNIST data file")
	flag.Parse()

	if err := loadMNISTData(*dataFile); err != nil {
		logger.Fatalf("Failed to load MNIST data: %v", err)
	}

	if err := termui.Init(); err != nil {
		logger.Fatalf("Failed to initialize termui: %v", err)
	}
	defer termui.Close()

	quitChan := make(chan struct{}) // Broadcast channel
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	logToWidget(fmt.Sprintf("Starting %d MNIST bots with %d-second intervals...", *numBots, *interval))

	var wg sync.WaitGroup
	for i := 0; i < *numBots; i++ {
		wg.Add(1)
		go startBot(*apiURL, time.Duration(*interval)*time.Second, &wg, quitChan)
	}

	// UI Event Handling
	uiEvents := termui.PollEvents()
	table := renderMetricsTable()
	fmt.Print("\n\n\n")
	logWidget := renderLogWidget()

	// Render initial UI
	termui.Render(table, logWidget)

	// Update the metrics table and log widget dynamically
	go func() {
		for {
			select {
			case <-quitChan:
				return
			case e := <-uiEvents:
				if e.Type == termui.KeyboardEvent && e.ID == "q" {
					logToWidget("Received 'q' key. Shutting down MNIST bots...")
					close(quitChan) // Stop all bots
					return
				}
			default:
				metricsMutex.Lock()
				table.Rows = [][]string{
					{"Metric", "Value"},
					{"Total Requests", fmt.Sprintf("%d", totalRequests)},
					{"Success Requests", fmt.Sprintf("%d", successRequests)},
					{"Average Latency (ms)", fmt.Sprintf("%.2f", averageLatency)},
					{"Failed Requests", fmt.Sprintf("%d", failedRequests)},
				}
				metricsMutex.Unlock()
				fmt.Println("\n\n")
				logMutex.Lock()
				logWidget.Rows = logEntries
				logMutex.Unlock()

				// Ensure proper UI rendering
				termui.Render(table, logWidget)
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// Wait for stop signal (Ctrl+C or 'q')
	select {
	case <-stopChan:
		logToWidget("\nReceived stop signal (Ctrl+C). Shutting down MNIST bots...")
		close(quitChan) // Signal goroutines to stop
	case <-quitChan:
		// 'q' key was pressed, and quitChan was closed
	}

	wg.Wait() // Wait for all bots to exit
	logToWidget("All bots stopped.\n")
	termui.Render(logWidget)    // Render final logs
	time.Sleep(2 * time.Second) // Allow time to see the final logs
}
