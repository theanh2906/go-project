package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	// Remove the command from os.Args so sub-commands see clean args
	os.Args = append(os.Args[:1], os.Args[2:]...)

	switch command {
	case "file-search":
		fileSearchMain()
	case "gap-filling":
		gapFillingMain()
	case "unit-converter":
		unitConverterMain()
	case "port-management":
		portManagementMain()
	case "winsearch":
		winsearchMain()
	case "mongo":
		mongoUtilsMain()
	case "jenkins":
		jenkinsCliMain()
	case "drive":
		ggDriveMain()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: go-project <command>")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  file-search      File search TUI")
	fmt.Println("  gap-filling      Interactive template generator")
	fmt.Println("  unit-converter   Unit converter (px to rem, etc.)")
	fmt.Println("  port-management  Port manager TUI")
	fmt.Println("  winsearch        Windows file search tool")
	fmt.Println("  mongo            MongoDB CLI utils")
	fmt.Println("  jenkins          Jenkins CLI")
	fmt.Println("  drive            Google Drive CLI")
}
