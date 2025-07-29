// backend/server/handlers/status_handler.go
package handlers

import (
    "encoding/json"
    "net/http"
    "sync"
)

// serverStatus holds the current processing stage.
var (
    statusMu     sync.RWMutex
    currentStage = "downloading"
)

// SetStatus can be called by other packages to update the stage.
func SetStatus(stage string) {
    statusMu.Lock()
    defer statusMu.Unlock()
    currentStage = stage
}

// StatusHandler returns the server's current processing stage in JSON.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
    statusMu.RLock()
    stage := currentStage
    statusMu.RUnlock()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "stage": stage,
    })
}
