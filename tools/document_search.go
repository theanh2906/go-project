package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed templates/*
var content embed.FS

type SearchResult struct {
	FilePath     string `json:"filePath"`
	LineNumber   int    `json:"lineNumber"`
	LineContent  string `json:"lineContent"`
	MatchStart   int    `json:"matchStart"`
	MatchEnd     int    `json:"matchEnd"`
	KeywordMatch string `json:"keywordMatch"`
}

type SearchRequest struct {
	FolderPath    string `json:"folderPath"`
	Keyword       string `json:"keyword"`
	CaseSensitive bool   `json:"caseSensitive"`
	WholeWord     bool   `json:"wholeWord"`
	UseRegex      bool   `json:"useRegex"`
}

type SearchResponse struct {
	Results  []SearchResult `json:"results"`
	Duration float64        `json:"duration"`
	Count    int            `json:"count"`
	Error    string         `json:"error,omitempty"`
}

type SearchEngine struct {
	keyword       string
	caseSensitive bool
	wholeWord     bool
	useRegex      bool
	results       []SearchResult
	resultsMutex  sync.Mutex
	semaphore     chan struct{}
	documentExts  map[string]bool
}

func NewSearchEngine() *SearchEngine {
	return &SearchEngine{
		semaphore: make(chan struct{}, 100), // 10 concurrent tasks
		documentExts: map[string]bool{
			".md":   true,
			".txt":  true,
			".doc":  true,
			".docx": true,
			".pdf":  true,
			".rtf":  true,
			".odt":  true,
			".log":  true,
			".csv":  true,
			".json": true,
			".xml":  true,
			".exe":  true,
		},
	}
}

func (se *SearchEngine) isDocumentFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return se.documentExts[ext]
}

func (se *SearchEngine) searchInFile(filePath string) {
	se.semaphore <- struct{}{}        // Acquire semaphore
	defer func() { <-se.semaphore }() // Release semaphore

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		matches := se.findMatches(line)

		for _, match := range matches {
			result := SearchResult{
				FilePath:     filePath,
				LineNumber:   lineNumber,
				LineContent:  line,
				MatchStart:   match[0],
				MatchEnd:     match[1],
				KeywordMatch: line[match[0]:match[1]],
			}

			se.resultsMutex.Lock()
			se.results = append(se.results, result)
			se.resultsMutex.Unlock()
		}
	}
}

func (se *SearchEngine) findMatches(line string) [][]int {
	var matches [][]int

	if se.useRegex {
		re, err := regexp.Compile(se.keyword)
		if err != nil {
			return matches
		}
		regexMatches := re.FindAllStringIndex(line, -1)
		return regexMatches
	}

	searchLine := line
	searchKeyword := se.keyword

	if !se.caseSensitive {
		searchLine = strings.ToLower(line)
		searchKeyword = strings.ToLower(se.keyword)
	}

	if se.wholeWord {
		words := strings.Fields(searchLine)
		currentPos := 0
		for _, word := range words {
			wordStart := strings.Index(searchLine[currentPos:], word)
			if wordStart == -1 {
				break
			}
			wordStart += currentPos

			cleanWord := strings.Trim(word, ".,!?;:\"'()[]{}|/\\")
			if cleanWord == searchKeyword {
				actualStart := wordStart
				actualEnd := wordStart + len(word)
				matches = append(matches, []int{actualStart, actualEnd})
			}
			currentPos = wordStart + len(word)
		}
	} else {
		// Simple substring search with all occurrences
		offset := 0
		for {
			index := strings.Index(searchLine[offset:], searchKeyword)
			if index == -1 {
				break
			}
			actualStart := offset + index
			actualEnd := actualStart + len(se.keyword)
			matches = append(matches, []int{actualStart, actualEnd})
			offset = actualEnd
		}
	}

	return matches
}

func (se *SearchEngine) searchDirectory(rootPath string) error {
	se.results = []SearchResult{}

	var wg sync.WaitGroup
	var filesProcessed int

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil
		}

		if !info.IsDir() && se.isDocumentFile(info.Name()) {
			filesProcessed++
			log.Printf("Processing file: %s", path)
			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()
				se.searchInFile(filePath)
			}(path)
		}

		return nil
	})

	wg.Wait()
	log.Printf("Total files processed: %d, Results found: %d", filesProcessed, len(se.results))
	return err
}

type Server struct {
	engine *SearchEngine
}

func NewServer() *Server {
	return &Server{
		engine: NewSearchEngine(),
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(content, "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(SearchResponse{Error: "Invalid request"})
		return
	}

	if req.FolderPath == "" || req.Keyword == "" {
		json.NewEncoder(w).Encode(SearchResponse{Error: "Folder path and keyword are required"})
		return
	}

	if _, err := os.Stat(req.FolderPath); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(SearchResponse{Error: "Folder does not exist"})
		return
	}

	// Configure search engine
	s.engine.keyword = req.Keyword
	s.engine.caseSensitive = req.CaseSensitive
	s.engine.wholeWord = req.WholeWord
	s.engine.useRegex = req.UseRegex

	startTime := time.Now()
	err := s.engine.searchDirectory(req.FolderPath)
	duration := time.Since(startTime).Seconds()

	response := SearchResponse{
		Results:  s.engine.results,
		Duration: duration,
		Count:    len(s.engine.results),
	}

	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	server := NewServer()

	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/search", server.handleSearch)

	port := "8080"
	fmt.Printf("Document Search Tool started at http://localhost:%s\n", port)
	fmt.Println("Press Ctrl+C to stop the server")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
