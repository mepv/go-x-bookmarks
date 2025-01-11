package main

import (
	"github.com/joho/godotenv"
	"github.com/mepv/go-x-bookmarks/cmd/config"
	"github.com/mepv/go-x-bookmarks/cmd/routes"
	"log"
	"net/http"
)

func main() {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Initialize configuration
	_ = config.NewConfig()

	// Setup mux
	mux := routes.SetupRouter()
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
