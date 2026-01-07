package main

import (
	"fmt"
	"sync"
	"time"
)

// Example 1: Basic goroutines with WaitGroup
func basicGoroutines() {
	fmt.Println("\n=== Example 1: Basic Goroutines ===")
	var wg sync.WaitGroup

	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Printf("Goroutine %d started\n", id)
			time.Sleep(time.Millisecond * time.Duration(100*id))
			fmt.Printf("Goroutine %d finished\n", id)
		}(i)
	}

	wg.Wait()
	fmt.Println("All goroutines completed")
}

// Example 2: Worker pool pattern with channels
func workerPool() {
	fmt.Println("\n=== Example 2: Worker Pool Pattern ===")

	jobs := make(chan int, 10)
	results := make(chan int, 10)

	// Start 3 worker goroutines
	var wg sync.WaitGroup
	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// Send 9 jobs
	for j := 1; j <= 9; j++ {
		jobs <- j
	}
	close(jobs)

	// Close results channel after all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		fmt.Printf("Result: %d\n", result)
	}
}

func worker(id int, jobs <-chan int, results chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		fmt.Printf("Worker %d processing job %d\n", id, job)
		time.Sleep(time.Millisecond * 500)
		results <- job * 2
	}
}

// Example 3: Fan-out, Fan-in pattern
func fanOutFanIn() {
	fmt.Println("\n=== Example 3: Fan-Out, Fan-In Pattern ===")

	// Input channel
	input := make(chan int, 5)

	// Create multiple workers (fan-out)
	var outputs []<-chan int
	for i := 0; i < 3; i++ {
		outputs = append(outputs, processNumbers(input, i+1))
	}

	// Send data
	go func() {
		for i := 1; i <= 5; i++ {
			input <- i
		}
		close(input)
	}()

	// Merge results (fan-in)
	for result := range merge(outputs...) {
		fmt.Printf("Final result: %d\n", result)
	}
}

func processNumbers(input <-chan int, id int) <-chan int {
	output := make(chan int)
	go func() {
		defer close(output)
		for num := range input {
			time.Sleep(time.Millisecond * 300)
			result := num * num
			fmt.Printf("Processor %d: %d -> %d\n", id, num, result)
			output <- result
		}
	}()
	return output
}

func merge(channels ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	merged := make(chan int)

	output := func(c <-chan int) {
		defer wg.Done()
		for val := range c {
			merged <- val
		}
	}

	wg.Add(len(channels))
	for _, c := range channels {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

// Example 4: Using select with multiple channels
func selectExample() {
	fmt.Println("\n=== Example 4: Select Statement ===")

	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(time.Second * 1)
		ch1 <- "Message from channel 1"
	}()

	go func() {
		time.Sleep(time.Millisecond * 500)
		ch2 <- "Message from channel 2"
	}()

	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-ch1:
			fmt.Println("Received:", msg1)
		case msg2 := <-ch2:
			fmt.Println("Received:", msg2)
		case <-time.After(time.Second * 2):
			fmt.Println("Timeout!")
		}
	}
}

// Example 5: Concurrent web scraper simulation
func concurrentScraper() {
	fmt.Println("\n=== Example 5: Concurrent Scraper Simulation ===")

	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
		"https://example.com/page4",
		"https://example.com/page5",
	}

	// Channel to collect results
	results := make(chan string, len(urls))
	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Limit concurrent requests to 3
	semaphore := make(chan struct{}, 3)

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Simulate scraping
			fmt.Printf("Fetching %s\n", url)
			time.Sleep(time.Millisecond * 800)
			results <- fmt.Sprintf("Content from %s", url)
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		fmt.Println("Got:", result)
	}
}

// Example 6: Cron job scheduler pattern
func scheduleJobs() {
	fmt.Println("\n=== Example 6: Cron Job Scheduler ===")

	// Create a ticker that fires every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Create a timer for a one-time job
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	// Channel to signal when to stop
	done := make(chan bool)

	// Simulate a job that runs periodically
	go func() {
		jobCounter := 0
		for {
			select {
			case <-ticker.C:
				jobCounter++
				fmt.Printf("[%s] Periodic job executed (count: %d)\n",
					time.Now().Format("15:04:05"), jobCounter)
			case <-done:
				fmt.Println("Stopping periodic job")
				return
			}
		}
	}()

	// Simulate a one-time scheduled job
	go func() {
		<-timer.C
		fmt.Printf("[%s] One-time scheduled job executed\n",
			time.Now().Format("15:04:05"))
	}()

	// Run for 10 seconds then stop
	time.Sleep(10 * time.Second)
	done <- true
	time.Sleep(time.Millisecond * 100) // Give time for cleanup

	fmt.Println("Scheduler stopped")
}

func main() {
	fmt.Println("Go Concurrency Examples")
	fmt.Println("=======================")

	// Run all examples
	// basicGoroutines()
	// workerPool()
	// fanOutFanIn()
	// selectExample()
	// concurrentScraper()
	scheduleJobs()

	fmt.Println("\n=== All Examples Completed ===")
}
