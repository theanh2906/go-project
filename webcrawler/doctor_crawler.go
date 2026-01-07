package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type DoctorInfo struct {
	Name            string   `json:"name"`
	Url             string   `json:"url"`
	Degree          string   `json:"degree"`
	Specialty       string   `json:"specialty"`
	Experience      []string `json:"experience"`
	TrainingProcess []string `json:"training_process"`
}

func htmlCrawler(url string, selector string, start int, end int) {
	// Control concurrency with a semaphore
	allDoctors := make([]DoctorInfo, 0)
	const maxConcurrent = 20
	semaphore := make(chan struct{}, maxConcurrent)

	// Create a custom HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Define retry parameters
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	// Rate limiter to avoid overloading the server
	rl := time.Tick(200 * time.Millisecond) // ~5 requests per second

	// Track results for reporting
	var (
		mutex      sync.Mutex
		successful int
		failed     int
	)

	// Wait group to track completion
	var wg sync.WaitGroup

	// Function to fetch a single page with retries
	fetchPage := func(pageNum int) {
		defer func() {
			wg.Done()
			<-semaphore // Release the semaphore slot
		}()

		pageUrl := strings.Replace(url, "{page}", fmt.Sprintf("%d", pageNum), 1)

		// Implement retry logic
		var doc *html.Node
		success := false

		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				fmt.Printf("Retrying page %d (attempt %d/%d)...\n", pageNum, attempt+1, maxRetries)
				time.Sleep(retryDelay * time.Duration(attempt)) // Exponential backoff
			}

			<-rl // Rate limiting
			//fmt.Printf("Fetching page %d: %s\n", pageNum, pageUrl)

			resp, err := client.Get(pageUrl)
			if err != nil {
				fmt.Printf("Error fetching page %d (attempt %d/%d): %v\n",
					pageNum, attempt+1, maxRetries, err)
				continue // Try again
			}

			// Use a safe way to close the body
			func() {
				defer resp.Body.Close()

				// Check for non-successful status code
				if resp.StatusCode != http.StatusOK {
					fmt.Printf("Error on page %d (attempt %d/%d): status code %d\n",
						pageNum, attempt+1, maxRetries, resp.StatusCode)
					return
				}

				// Try to parse the HTML
				var parseErr error
				doc, parseErr = html.Parse(resp.Body)
				if parseErr != nil {
					fmt.Printf("Error parsing page %d (attempt %d/%d): %v\n",
						pageNum, attempt+1, maxRetries, parseErr)
					return
				}

				// If we reach here, we succeeded
				success = true
			}()

			if success {
				break // Exit retry loop on success
			}
		}

		// If we still couldn't fetch the page after all retries
		if !success {
			fmt.Printf("Failed to fetch page %d after %d attempts\n", pageNum, maxRetries)
			mutex.Lock()
			failed++
			mutex.Unlock()
			return
		}

		// Process the HTML document
		var f func(*html.Node)
		elementCount := 0
		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == selector {
				elementCount++
				//fmt.Printf("Found matching element on page %d\n", pageNum)
				if len(n.FirstChild.Parent.Parent.Attr) > 1 {
					fmt.Println("Element found:", n.FirstChild.Parent.Parent.Attr[1].Val)
					doctorInfo, err := crawlEachPage(n.FirstChild.Parent.Parent.Attr[1].Val, client)
					if err == nil {
						allDoctors = append(allDoctors, doctorInfo)
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
		f(doc)
		//
		//fmt.Printf("Successfully processed page %d, found %d %s elements\n",
		//	pageNum, elementCount, selector)

		mutex.Lock()
		successful++
		mutex.Unlock()
	}

	// Start crawling pages
	startTime := time.Now()

	for page := start; page <= end; page++ {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire a semaphore slot
		go fetchPage(page)
	}

	// Wait for all pages to be processed
	wg.Wait()

	// Report results
	duration := time.Since(startTime)
	totalPages := end - start + 1
	fmt.Printf("\nCrawling summary:\n")
	fmt.Printf("Total pages attempted: %d\n", totalPages)
	fmt.Printf("Successfully crawled: %d (%.1f%%)\n", successful, float64(successful)/float64(totalPages)*100)
	fmt.Printf("Failed pages: %d (%.1f%%)\n", failed, float64(failed)/float64(totalPages)*100)
	fmt.Printf("Total time: %s\n", duration)
	fmt.Printf("Total doctors found: %d\n", len(allDoctors))
	if len(allDoctors) > 0 {
		// Save results to JSON file
		err := saveToJson(allDoctors, "doctors.json")
		if err != nil {
			fmt.Printf("Error saving data to JSON: %v\n", err)
		}
	} else {
		fmt.Println("No doctors found.")
	}
}

func crawlEachPage(url string, client *http.Client) (DoctorInfo, error) {
	// Define retry parameters
	const maxRetries = 3
	const retryDelay = 2 * time.Second

	info := DoctorInfo{}
	info.Url = url
	var doc *html.Node
	var success bool

	// Implement retry logic
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("Retrying doctor page %s (attempt %d/%d)...\n", url, attempt+1, maxRetries)
			time.Sleep(retryDelay * time.Duration(attempt)) // Exponential backoff
		}

		// Fetch the doctor's page
		res, err := client.Get(url)
		if err != nil {
			fmt.Printf("Error fetching doctor page %s (attempt %d/%d): %v\n", url, attempt+1, maxRetries, err)
			continue // Try again
		}

		// Use a safe way to close the body and process the response
		func() {
			defer res.Body.Close()

			// Check for non-successful status code
			if res.StatusCode != http.StatusOK {
				fmt.Printf("Error fetching doctor page %s (attempt %d/%d): status code %d\n",
					url, attempt+1, maxRetries, res.StatusCode)
				return
			}

			// Try to parse the HTML
			var parseErr error
			doc, parseErr = html.Parse(res.Body)
			if parseErr != nil {
				fmt.Printf("Error parsing doctor page %s (attempt %d/%d): %v\n",
					url, attempt+1, maxRetries, parseErr)
				return
			}

			// If we reach here, we succeeded
			success = true
		}()

		if success {
			break // Exit retry loop on success
		}
	}

	// If we still couldn't fetch the page after all retries
	if !success {
		fmt.Printf("Failed to fetch doctor page %s after %d attempts\n", url, maxRetries)
		return DoctorInfo{}, fmt.Errorf("failed to fetch doctor page after %d attempts", maxRetries)
	}

	// Extract doctor information from the HTML document
	extractDoctorInfo(doc, &info)

	return info, nil
}

// extractDoctorInfo parses the HTML document to extract doctor information
func extractDoctorInfo(doc *html.Node, info *DoctorInfo) {
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		// Extract doctor name from h1 tag
		if n.Type == html.ElementNode && n.Data == "h1" {
			if n.FirstChild != nil {
				info.Name = strings.TrimSpace(n.FirstChild.Data)
				info.Degree = strings.Split(info.Name, " ")[0]
			}
		}

		// Extract doctor degree if available
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "sss") {
			if n.FirstChild != nil {
				info.Specialty = strings.TrimSpace(n.FirstChild.Data)
			}
		}

		// Extract doctor specialty if available
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "specialty") {
			if n.FirstChild != nil {
				info.Specialty = strings.TrimSpace(n.FirstChild.Data)
				fmt.Printf("Doctor Specialty: %s\n", info.Specialty)
			}
		}

		// Extract doctor experience
		if n.Type == html.ElementNode && n.Data == "div" && len(n.Attr) > 0 && n.Attr[0].Val == "collapsekinhnghiemct" {
			extractExperience(n, info)
		}

		// Continue traversing the DOM
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)
}

// extractExperience extracts the experience items from the experience div
func extractExperience(n *html.Node, info *DoctorInfo) {
	// Find list items within the experience section
	var findListItems func(*html.Node)
	findListItems = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "li" {
			if node.FirstChild != nil {
				expText := strings.TrimSpace(extractTextContent(node))
				if expText != "" {
					info.Experience = append(info.Experience, expText)
					//fmt.Printf("Experience: %s\n", expText)
				}
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			findListItems(c)
		}
	}

	findListItems(n)
}

// extractTextContent extracts all text content from a node and its children
func extractTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractTextContent(c)
	}
	return result
}

// hasClass checks if an HTML element has a specific class
func hasClass(n *html.Node, className string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			classes := strings.Fields(attr.Val)
			if slices.Contains(classes, className) {
				return true
			}
		}
	}
	return false
}

func hasId(n *html.Node, id string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "id" && attr.Val == id {
			return true
		}
	}
	return false

}

func saveToJson(allDoctors []DoctorInfo, filename string) error {
	// Convert allDoctors to JSON and save to file
	data, err := json.MarshalIndent(allDoctors, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling data to JSON: %v", err)
	}
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing JSON to file: %v", err)
	}
	fmt.Printf("Data saved to %s\n", filename)
	return nil

}

func main() {
	htmlCrawler("https://tamanhhospital.vn/chuyen-gia/page/{page}/?filter_search&filter_diadiem=36&filter_chuyenkhoa&filter_chucvu&filter_ngonngu&filter_hocham&filter_hocvi", "h2", 1, 63)
}
