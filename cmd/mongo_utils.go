package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	ConnectionString string `json:"connection_string"`
}

func mongoConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mongo-cli"), nil
}

func loadMongoConfig() (*MongoConfig, error) {
	dir, err := mongoConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var cfg MongoConfig
	if err := dec.Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveMongoConfig(cfg *MongoConfig) error {
	dir, err := mongoConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.json")
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}

func askForMongoConfig() (*MongoConfig, error) {
	cyan := color.New(color.FgCyan).SprintFunc()
	fmt.Println(cyan("\nMongoDB CLI setup — let's connect to your database."))

	connPrompt := promptui.Prompt{
		Label: "MongoDB Connection String (e.g. mongodb://localhost:27017 or mongodb+srv://...)",
		Validate: func(s string) error {
			if s == "" {
				return fmt.Errorf("connection string is required")
			}
			return nil
		},
	}
	connStr, err := connPrompt.Run()
	if err != nil {
		return nil, err
	}

	cfg := &MongoConfig{ConnectionString: connStr}
	if err := saveMongoConfig(cfg); err != nil {
		return nil, err
	}
	color.New(color.FgGreen).Println("Saved configuration to ~/.mongo-cli/config.json")
	return cfg, nil
}

func ensureMongoConfig() (*MongoConfig, error) {
	cfg, err := loadMongoConfig()
	if err == nil {
		return cfg, nil
	}
	return askForMongoConfig()
}

func connectMongoDB(cfg *MongoConfig) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.ConnectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	// Ping the database to verify connection
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := client.Ping(ctx2, nil); err != nil {
		client.Disconnect(context.Background())
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return client, nil
}

func testConnection(cfg *MongoConfig) error {
	color.Yellow("Testing connection to MongoDB...")
	client, err := connectMongoDB(cfg)
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())

	color.Green("✔ Successfully connected to MongoDB")

	// List databases to show connection is working
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	databases, err := client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("connected but failed to list databases: %v", err)
	}

	fmt.Println("\nAvailable databases:")
	for _, db := range databases {
		fmt.Printf("  - %s\n", db)
	}

	return nil
}

func mongoClearScreen() {
	// ANSI clear: clear entire screen and move cursor to home
	fmt.Print("\033[2J\033[H")
}

func mongoMainMenu() (string, error) {
	green := color.New(color.FgGreen).SprintFunc()
	mag := color.New(color.FgMagenta).SprintFunc()
	title := fmt.Sprintf("%s %s", mag("MongoDB"), green("CLI"))
	prompt := promptui.Select{
		Label: title + " — select action",
		Items: []string{"Test connection", "Configure", "Exit"},
		Size:  5,
	}
	_, v, err := prompt.Run()
	return v, err
}

func mongoConfigureMenu() (*MongoConfig, error) {
	return askForMongoConfig()
}

func removeFirstUser() (bool, error) {
	return false, nil
}

func main() {
	mongoClearScreen()
	color.Cyan("✨ MongoDB CLI — Manage your MongoDB data. Hello!")
	cfg, err := ensureMongoConfig()
	if err != nil {
		color.Red("Unable to load configuration: %v", err)
		return
	}

	for {
		// New screen for main menu
		mongoClearScreen()
		choice, err := mongoMainMenu()
		if err != nil {
			fmt.Println()
			return
		}

		switch choice {
		case "Test connection":
			mongoClearScreen()
			if err := testConnection(cfg); err != nil {
				color.Red("Connection failed: %v", err)
			}
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "Configure":
			mongoClearScreen()
			newCfg, err := mongoConfigureMenu()
			if err != nil {
				color.Red("Configuration failed: %v", err)
			} else {
				cfg = newCfg
			}
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()

		case "Exit":
			color.Cyan("Goodbye 👋")
			return
		}
	}
}
