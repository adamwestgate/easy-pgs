package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/adamwestgate/easy-pgs/backend/data"
)

// TraitResult is the shape we return to the client
type TraitResult struct {
    ID          string                   `json:"id"`
    Label       string                   `json:"label"`
    Description string                   `json:"description"`
    URL         string                   `json:"url"`
    Metadata    []map[string]interface{} `json:"metadata"`
}

// SearchHandler returns ontology traits plus PGS metadata, omitting GRCh38 weights
func SearchHandler(w http.ResponseWriter, r *http.Request) {
    // CORS pre‑flight
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }

    q := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("q")))
    var results []TraitResult

    for _, trait := range data.LoadedTraits {
        if strings.Contains(strings.ToLower(trait.Label), q) ||
            strings.Contains(strings.ToLower(trait.Description), q) ||
            strings.Contains(strings.ToLower(trait.ID), q) {

            var metas []map[string]interface{}

            // Collect all PGS metadata rows linked to this trait, skipping GRCh38 builds
            for _, pgsID := range trait.PGSFiles {
                for _, meta := range data.LoadedScores {
                    if id, ok := meta["Polygenic Score (PGS) ID"].(string); ok && id == pgsID {
                        // If the metadata explicitly states GRCh38, skip it --> Ancestry and 23andMe only use GRCh37
                        if build, ok := meta["Original Genome Build"].(string); ok && strings.EqualFold(build, "GRCh38") {
                            continue
                        }
                        if build, ok := meta["Genome Build"].(string); ok && strings.EqualFold(build, "GRCh38") {
                            continue
                        }
                        metas = append(metas, meta)
                    }
                }
            }

            results = append(results, TraitResult{
                ID:          trait.ID,
                Label:       trait.Label,
                Description: trait.Description,
                URL:         trait.URL,
                Metadata:    metas,
            })
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
}
