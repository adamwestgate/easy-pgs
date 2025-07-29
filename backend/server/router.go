// server/router.go
package main

import (
    "net/http"

    "github.com/gorilla/handlers" // CORS middleware
    "github.com/gorilla/mux"
    "github.com/adamwestgate/easy-pgs/backend/config"

    apihandlers "github.com/adamwestgate/easy-pgs/backend/server/handlers"
)

func NewRouter() http.Handler {
    r := mux.NewRouter().StrictSlash(true)

    // Allow GET for search/status and POST for downloads
    corsOpts := handlers.CORS(
        handlers.AllowedOrigins([]string{config.FrontendOrigin}),
        handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
        handlers.AllowedHeaders([]string{"Content-Type"}),
        handlers.AllowCredentials(),
    )

    // search endpoint
    r.HandleFunc("/search", apihandlers.SearchHandler).
        Methods("GET", "OPTIONS")

    // bulk-download endpoint (expects JSON { pgsIds: [...] })
    r.HandleFunc("/download", apihandlers.DownloadHandler).
        Methods("POST", "OPTIONS")

    // status endpoint returns current server stage
    r.HandleFunc("/status", apihandlers.StatusHandler).
        Methods("GET", "OPTIONS")

    // upload kit endpoint returns upload success/failure
    r.HandleFunc("/upload-kit", apihandlers.UploadKitHandler).
        Methods("POST", "OPTIONS")

    // retrieve results for results page
    r.HandleFunc("/results", apihandlers.ResultsHandler).
        Methods("GET",  "OPTIONS")

    return corsOpts(r)
}
