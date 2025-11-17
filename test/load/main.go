package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type LoadTestResult struct {
	TotalRequests    int64
	SuccessfulRequests int64
	FailedRequests   int64
	RateLimitedRequests int64
	TotalTime        time.Duration
	MinResponseTime  time.Duration
	MaxResponseTime  time.Duration
	AvgResponseTime  time.Duration
}

type LoadTester struct {
	baseURL      string
	concurrent   int
	duration     time.Duration
	client       *http.Client
	results      LoadTestResult
	responseTimes []time.Duration
	mu           sync.Mutex
}

func NewLoadTester(baseURL string, concurrent int, duration time.Duration) *LoadTester {
	return &LoadTester{
		baseURL:    baseURL,
		concurrent: concurrent,
		duration:   duration,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		responseTimes: make([]time.Duration, 0),
	}
}

func (lt *LoadTester) makeRequest(endpoint string) {
	start := time.Now()
	
	resp, err := lt.client.Get(lt.baseURL + endpoint)
	elapsed := time.Since(start)
	
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	lt.responseTimes = append(lt.responseTimes, elapsed)
	atomic.AddInt64(&lt.results.TotalRequests, 1)
	
	if err != nil {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusTooManyRequests {
		atomic.AddInt64(&lt.results.RateLimitedRequests, 1)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddInt64(&lt.results.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&lt.results.FailedRequests, 1)
	}
}

func (lt *LoadTester) worker(endpoints []string, stop chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for {
		select {
		case <-stop:
			return
		default:
			for _, endpoint := range endpoints {
				lt.makeRequest(endpoint)
			}
		}
	}
}

func (lt *LoadTester) Run() LoadTestResult {
	endpoints := []string{
		"/healthz",
		"/api/rooms",
		"/api/records",
		"/metrics",
	}
	
	stop := make(chan bool)
	var wg sync.WaitGroup
	
	start := time.Now()
	
	// Start workers
	for i := 0; i < lt.concurrent; i++ {
		wg.Add(1)
		go lt.worker(endpoints, stop, &wg)
	}
	
	// Let it run for the specified duration
	time.Sleep(lt.duration)
	
	// Stop all workers
	close(stop)
	wg.Wait()
	
	lt.results.TotalTime = time.Since(start)
	lt.calculateStats()
	
	return lt.results
}

func (lt *LoadTester) calculateStats() {
	if len(lt.responseTimes) == 0 {
		return
	}
	
	var totalTime time.Duration
	lt.results.MinResponseTime = lt.responseTimes[0]
	lt.results.MaxResponseTime = lt.responseTimes[0]
	
	for _, rt := range lt.responseTimes {
		totalTime += rt
		if rt < lt.results.MinResponseTime {
			lt.results.MinResponseTime = rt
		}
		if rt > lt.results.MaxResponseTime {
			lt.results.MaxResponseTime = rt
		}
	}
	
	lt.results.AvgResponseTime = totalTime / time.Duration(len(lt.responseTimes))
}

func (lt *LoadTester) PrintResults() {
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Base URL: %s\n", lt.baseURL)
	fmt.Printf("Concurrent Users: %d\n", lt.concurrent)
	fmt.Printf("Duration: %s\n", lt.duration)
	fmt.Printf("Total Requests: %d\n", lt.results.TotalRequests)
	fmt.Printf("Successful Requests: %d\n", lt.results.SuccessfulRequests)
	fmt.Printf("Failed Requests: %d\n", lt.results.FailedRequests)
	fmt.Printf("Rate Limited Requests: %d\n", lt.results.RateLimitedRequests)
	fmt.Printf("Total Time: %s\n", lt.results.TotalTime)
	fmt.Printf("Requests per Second: %.2f\n", float64(lt.results.TotalRequests)/lt.results.TotalTime.Seconds())
	fmt.Printf("Average Response Time: %s\n", lt.results.AvgResponseTime)
	fmt.Printf("Min Response Time: %s\n", lt.results.MinResponseTime)
	fmt.Printf("Max Response Time: %s\n", lt.results.MaxResponseTime)
	
	successRate := float64(lt.results.SuccessfulRequests) / float64(lt.results.TotalRequests) * 100
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	
	if successRate < 95 {
		fmt.Println("⚠️  WARNING: Success rate is below 95%")
	} else {
		fmt.Println("✅ Success rate is acceptable")
	}
	
	if lt.results.AvgResponseTime > 1*time.Second {
		fmt.Println("⚠️  WARNING: Average response time is above 1 second")
	} else {
		fmt.Println("✅ Average response time is acceptable")
	}
}

func main() {
	var (
		baseURL    = flag.String("url", "http://localhost:8080", "Base URL of the server")
		concurrent = flag.Int("concurrent", 10, "Number of concurrent users")
		duration   = flag.Duration("duration", 30*time.Second, "Test duration")
	)
	
	flag.Parse()
	
	fmt.Printf("Starting load test...\n")
	fmt.Printf("URL: %s\n", *baseURL)
	fmt.Printf("Concurrent users: %d\n", *concurrent)
	fmt.Printf("Duration: %s\n", *duration)
	
	// Test server connectivity first
	fmt.Println("Testing server connectivity...")
	resp, err := http.Get(*baseURL + "/healthz")
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server health check failed: %d", resp.StatusCode)
	}
	
	fmt.Println("✅ Server is accessible")
	fmt.Printf("Starting load test with %d concurrent users for %s...\n", *concurrent, *duration)
	
	lt := NewLoadTester(*baseURL, *concurrent, *duration)
	results := lt.Run()
	
	lt.PrintResults()
	
	// Exit with error code if success rate is too low
	if float64(results.SuccessfulRequests)/float64(results.TotalRequests) < 0.95 {
		os.Exit(1)
	}
}