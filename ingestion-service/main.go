package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Println("starting ingestion service")
	args := os.Args
	if len(args) < 2 {
		log.Fatal("no port specified.")
	}
	port := args[1]

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/events", apiEventsHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", port),
		Handler: mux,
	}

	log.Printf("ingestion service is listening on port: %v", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("failed to start server, error: %v", err)
	}
}

func apiEventsHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if request.Body == nil {
		log.Print("body is nil")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	apiEvent, err := io.ReadAll(request.Body)
	if err != nil {
		log.Printf("failed to read request body, error: %v", err)
		http.Error(writer, "failed to read request body", http.StatusInternalServerError)
		return
	}

	log.Printf("API Event received\n%v", string(apiEvent))
}
