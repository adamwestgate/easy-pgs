// cmd/server/main.go
package main

import (
    "log"
    "net/http"
    "fmt"

    "github.com/adamwestgate/easy-pgs/backend/data"
    "github.com/adamwestgate/easy-pgs/backend/config"
    "github.com/adamwestgate/easy-pgs/backend/server/handlers"
    "github.com/adamwestgate/easy-pgs/backend/store/boltstore"
)

// dataDir is where kits.db and other local data files live.
const dataDir = "backend/data"

func main() {
    // 1) Load static metadata used by search endpoints
    if err := data.LoadMetadata(); err != nil {
        log.Fatalf("could not load scores metadata: %v", err)
    }

    // 2) Initialise the kit‑mapping store (embedded BoltDB)
    kitStore, err := boltstore.Open(dataDir)
    if err != nil {
        log.Fatalf("could not open kit store: %v", err)
    }
    handlers.SetKitStore(kitStore)

    // 3) Build HTTP router and start server
    router := NewRouter()
    addr := fmt.Sprintf("%s:%s", config.ServerHost, config.ServerPort)
    log.Printf("Server listening on %s", addr)
    if err := http.ListenAndServe(addr, router); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}
