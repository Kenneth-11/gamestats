package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gamestats/go-service/internal/api"
	"gamestats/go-service/internal/ingestion"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	port := getenv("GO_SERVICE_PORT", "8080")
	scalaURL := getenv("SCALA_SERVICE_URL", "http://localhost:8081")
	workers := 4

	ing := ingestion.NewIngestor("", scalaURL)
	ing.Start(workers)
	defer ing.Stop()

	srv := api.NewServer(ing)

	addr := ":" + port
	fmt.Printf("gamestats-go escuchando en %s (scala=%s workers=%d)\n", addr, scalaURL, workers)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
