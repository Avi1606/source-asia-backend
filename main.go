package main

import (
	"encoding/json"
	"log"
	"net/http"

	"source-asia-backend/catalog"
	"source-asia-backend/ratelimit"
)

func main() {
	rlStore := ratelimit.NewRateLimitStore()
	rlHandler := ratelimit.NewHandler(rlStore)

	catStore := catalog.NewCatalogStore()
	catHandler := catalog.NewHandler(catStore)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /request", rlHandler.HandleRequest)
	mux.HandleFunc("GET /stats", rlHandler.HandleStats)
	mux.HandleFunc("POST /products", catHandler.CreateProduct)
	mux.HandleFunc("GET /products", catHandler.ListProducts)
	mux.HandleFunc("GET /products/{id}", catHandler.GetProduct)
	mux.HandleFunc("POST /products/{id}/media", catHandler.AddMedia)
	mux.HandleFunc("/", handleNotFound)

	log.Println("server listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
}
