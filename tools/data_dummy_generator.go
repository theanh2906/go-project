package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type DummyTool struct {
	FilePath       string
	UniqueFields   []string
	FieldsToRemove []string
	RecordNumber   int
	OutputFilePath string
	OutputFileName string
}

func main() {
	filePath := flag.String("path", "data.json", "Path to the output JSON file")
	uniqueFields := flag.String("unique_fields", "id", "Comma-separated list of unique fields")
	fieldsToRemove := flag.String("fields_to_remove", "created_at,updated_at", "Comma-separated list of fields to remove")
	recordNumber := flag.Int("record_number", 100, "Number of records to generate")
	outputFilePath := flag.String("output_file_path", ".", "Path to save the output file")
	outputFileName := flag.String("output_file_name", "dummy_data.json", "Name of the output file")
	flag.Parse()
	dummyTool := DummyTool{
		FilePath:       *filePath,
		UniqueFields:   splitString(*uniqueFields),
		FieldsToRemove: splitString(*fieldsToRemove),
		RecordNumber:   *recordNumber,
		OutputFilePath: *outputFilePath,
		OutputFileName: *outputFileName,
	}
	dummyTool.generateDummy()
}

func splitString(s string) []string {
	return strings.Split(s, ",")
}

func (t *DummyTool) generateDummy() {
	data, _ := t.readInputFileAsJson()
	fmt.Println(data)
}

func (t *DummyTool) readInputFileAsJson() (any, error) {
	data, err := os.ReadFile(t.FilePath)
	if err != nil {
		return "", err
	}
	// Validate JSON format
	if !strings.HasPrefix(string(data), "{") && !strings.HasPrefix(string(data), "[") {
		return "", os.ErrInvalid
	}
	parsedJsonString, _ := parseToAny(string(data))
	prettyJsonFormar, _ := prettyJSON(parsedJsonString)
	fmt.Println("Parsed JSON:", prettyJsonFormar)
	// Remove specified fields
	result, err := removeFields(parsedJsonString, t.FieldsToRemove)
	return result, nil
}

func parseToAny(jsonString string) (any, error) {
	var v any
	err := json.Unmarshal([]byte(jsonString), &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func prettyJSON(v any) (string, error) {
	var obj any
	// If input is a string, try to unmarshal it first
	switch vv := v.(type) {
	case string:
		if err := json.Unmarshal([]byte(vv), &obj); err != nil {
			return "", fmt.Errorf("prettyJSON: input string is not valid JSON: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(vv, &obj); err != nil {
			return "", fmt.Errorf("prettyJSON: input bytes are not valid JSON: %w", err)
		}
	default:
		obj = v
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("prettyJSON: failed to marshal: %w", err)
	}
	return string(data), nil
}

func removeFields(v any, fieldsToRemove []string) (any, error) {
	switch vv := v.(type) {
	case map[string]any:
		for _, field := range fieldsToRemove {
			// Check nested fields
			if strings.Contains(field, ".") {
				parts := strings.Split(field, ".")
				if len(parts) > 1 {
					nestedMap := vv
					for i := 0; i < len(parts)-1; i++ {
						if val, ok := nestedMap[parts[i]]; ok {
							if nextMap, ok := val.(map[string]any); ok {
								nestedMap = nextMap
							} else {
								break
							}
						} else {
							break
						}
					}
					delete(nestedMap, parts[len(parts)-1])
				}
			} else {
				delete(vv, field)
			}
		}
		return vv, nil
	case []any:
		for i, item := range vv {
			// Recursively remove fields from each item in the array
			cleanedItem, err := removeFields(item, fieldsToRemove)
			if err != nil {
				return nil, err
			}
			vv[i] = cleanedItem
		}
		return vv, nil
	default:
		return v, nil // No fields to remove for non-map types
	}
}
