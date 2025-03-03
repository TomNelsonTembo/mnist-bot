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
	maxLogs    = 10 // Limit logs displayed in UI
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
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&mnistSamples); err != nil {
			return fmt.Errorf("failed to decode JSON: %v", err)
		}
	} else {
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

	logToWidget(fmt.Sprintf("Loaded %d MNIST samples", len(mnistSamples)))
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

	requestBody := MNISTData{Instances: [][]float64{data}}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logToWidget(fmt.Sprintf("Error marshaling JSON: %v", err))
		return
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		logToWidget(fmt.Sprintf("Error sending request: %v", err))
		metricsMutex.Lock()
		failedRequests++
		metricsMutex.Unlock()
		return
	}
	defer resp.Body.Close()

	latency := time.Since(startTime).Seconds() * 1000

	metricsMutex.Lock()
	totalRequests++
	if resp.StatusCode == http.StatusOK {
		successRequests++
		latencies = append(latencies, latency)
		averageLatency = calculateAverageLatency()
	} else {
		failedRequests++
		logToWidget(fmt.Sprintf("Request failed: %s", resp.Status))
	}
	metricsMutex.Unlock()

	logToWidget(fmt.Sprintf("Request sent and Saved Successfully, Latency: %.2f ms", latency))
}

// calculateAverageLatency computes the average latency from recorded values
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

// logToWidget adds a log entry while ensuring it doesn't overflow the UI
func logToWidget(message string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	logEntries = append(logEntries, message)
	if len(logEntries) > maxLogs {
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
		{"Failed Requests", fmt.Sprintf("%d", failedRequests)},
		{"Average Latency (ms)", fmt.Sprintf("%.2f", averageLatency)},
	}
	table.TextStyle = termui.NewStyle(termui.ColorWhite)
	table.Title = "MNIST Bot Metrics"
	table.SetRect(0, 0, 50, 12)
	return table
}

// renderLogWidget creates a terminal-based widget to display logs
func renderLogWidget() *widgets.List {
	list := widgets.NewList()
	list.Title = "Logs"
	list.TextStyle = termui.NewStyle(termui.ColorWhite)
	list.WrapText = true
	list.SetRect(0, 9, 100, 18) // Positioned below metrics
	return list
}

// starts sending randomly selected data at a specific rate
func main() {
	apiURL := flag.String("api", "", "API endpoint URL")
	numBots := flag.Int("bots", 1, "Number of concurrent bots")
	interval := flag.Int("interval", 1, "Interval between requests (seconds)")
	dataFile := flag.String("data", "./Assets/Data/data.json", "Path to MNIST data file")
	flag.Parse()

	// loads MNIST Data
	if err := loadMNISTData(*dataFile); err != nil {
		logger.Fatalf("Failed to load MNIST data: %v", err)
	}

	if err := termui.Init(); err != nil {
		logger.Fatalf("Failed to initialize termui: %v", err)
	}
	defer termui.Close()

	quitChan := make(chan struct{})
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	logToWidget(fmt.Sprintf("Starting %d MNIST bots at %d-second intervals...", *numBots, *interval))

	var wg sync.WaitGroup
	for i := 0; i < *numBots; i++ {
		wg.Add(1)
		go startBot(*apiURL, time.Duration(*interval)*time.Second, &wg, quitChan)
	}

	uiEvents := termui.PollEvents()
	table := renderMetricsTable()
	logWidget := renderLogWidget()

	// Initial UI rendering
	termui.Render(table, logWidget)

	go func() {
		for {
			select {
			case <-quitChan:
				return
			case e := <-uiEvents:
				if e.Type == termui.KeyboardEvent && e.ID == "q" {
					logToWidget("Received 'q'. Stopping bots...")
					close(quitChan)
					return
				}
			default:
				metricsMutex.Lock()
				table.Rows = [][]string{
					{"Metric", "Value"},
					{"Total Requests", fmt.Sprintf("%d", totalRequests)},
					{"Success Requests", fmt.Sprintf("%d", successRequests)},
					{"Average Latency (ms)", fmt.Sprintf("%.2f", averageLatency)},
				}
				metricsMutex.Unlock()

				logMutex.Lock()
				logWidget.Rows = append([]string{}, logEntries...) // Prevent infinite growth
				logMutex.Unlock()

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
