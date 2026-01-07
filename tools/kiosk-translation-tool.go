package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
)

// This tool generates translations for the kiosk application.
// It reads the translate.csv file as json line by line where key is the name of specific language file in folder languages
// for each key, find the associated json file in the languages folder then find the key with path languages.data
// and add new key that define in "key" column and value is the other column value with format : [key]: { text: [value] }. After that, sort all
// the languages.data by key with alphabetical order and update in that file.

func main() {
	csvFile, err := os.Open("translate.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	if len(records) < 2 {
		fmt.Println("No data in CSV")
		return
	}

	headers := records[0]
	langFiles := headers[1:]

	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 2 {
			continue
		}
		key := row[0]
		for j, lang := range langFiles {
			langFile := filepath.Join("languages", lang+".json")
			updateLanguageFile(langFile, key, row[j+1])
		}
	}
}

type LanguageData struct {
	LangID string                 `json:"lang_id"`
	Data   map[string]interface{} `json:"data"`
}

type LanguageFile struct {
	Languages []LanguageData `json:"languages"`
}

func updateLanguageFile(filename, key, value string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", filename, err)
		return
	}
	var langFile LanguageFile
	err = json.Unmarshal(data, &langFile)
	if err != nil || len(langFile.Languages) == 0 {
		fmt.Printf("Invalid JSON in %s\n", filename)
		return
	}
	ld := &langFile.Languages[0]
	if ld.Data == nil {
		ld.Data = make(map[string]interface{})
	}
	ld.Data[key] = map[string]string{"text": value}

	// Sort keys
	keys := make([]string, 0, len(ld.Data))
	for k := range ld.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedData := make(map[string]interface{}, len(ld.Data))
	for _, k := range keys {
		sortedData[k] = ld.Data[k]
	}
	ld.Data = sortedData

	out, err := json.MarshalIndent(langFile, "", "    ")
	if err != nil {
		fmt.Printf("Failed to marshal %s: %v\n", filename, err)
		return
	}
	err = ioutil.WriteFile(filename, out, 0644)
	if err != nil {
		fmt.Printf("Failed to write %s: %v\n", filename, err)
	}
}
