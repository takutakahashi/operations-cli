package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/config.yaml", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("config.yaml")
		if err != nil {
			http.Error(w, "Failed to read config file", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/x-yaml")
		if _, err := w.Write(data); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
			return
		}
	})

	port := "8080"
	fmt.Printf("Starting test server on port %s...\n", port)
	fmt.Printf("Access the config file at: http://localhost:%s/config.yaml\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
