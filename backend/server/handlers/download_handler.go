package handlers

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamwestgate/easy-pgs/backend/data"
	pgs_convert "github.com/adamwestgate/easy-pgs/backend/preprocessing/pgs_convert"
	"github.com/adamwestgate/easy-pgs/backend/preprocessing/scoring"
	"github.com/adamwestgate/easy-pgs/backend/config"
)

// DownloadRequest is the JSON payload shape the frontend sends.
type DownloadRequest struct {
	KitID  string   `json:"kitId"`
	PgsIds []string `json:"pgsIds"`
}

// DownloadResponse contains scoring results for each PGS ID.
type DownloadResponse struct {
	User map[string]scoring.BatchResult `json:"user"`
	Pop  map[string]scoring.BatchResult `json:"pop"`
}

// DownloadHandler handles kit download and scoring.
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("DownloadHandler: called")
	SetStatus("downloading")

	// CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode request
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("DownloadHandler: JSON decode error: %v", err)
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate payload
	if req.KitID == "" {
		http.Error(w, "kitId is required", http.StatusBadRequest)
		return
	}
	if len(req.PgsIds) == 0 {
		http.Error(w, "no pgsIds provided", http.StatusBadRequest)
		return
	}

	// Locate processed kit prefix (for logging)
	userPfilePrefix, _, ok := kitStore.Lookup(req.KitID)
	if !ok {
		http.Error(w, "invalid kit_id", http.StatusBadRequest)
		return
	}
	log.Printf("DownloadHandler: Resolved kit prefix: %s", userPfilePrefix)

	// Download and normalize each PGS file
	normPaths := make([]string, 0, len(req.PgsIds))
	for _, pgsID := range req.PgsIds {
		link := findScoreURL(pgsID)
		if strings.HasPrefix(link, "ftp://") {
			link = "http://" + strings.TrimPrefix(link, "ftp://")
		}

		// Use config for download directory
		dir := filepath.Join(config.PGSDownloadDir, pgsID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("DownloadHandler: mkdir %s failed: %v", dir, err)
			continue
		}

		// Download if missing
		fname := filepath.Base(link)
		gzPath := filepath.Join(dir, fname)
		if _, err := os.Stat(gzPath); os.IsNotExist(err) {
			if err := fetchFile(link, gzPath); err != nil {
				log.Printf("DownloadHandler: fetchFile error for %s: %v", pgsID, err)
				continue
			}
		}

		// Decompress and normalize
		src, err := os.Open(gzPath)
		if err != nil {
			continue
		}
		gzReader, err := gzip.NewReader(src)
		if err != nil {
			src.Close()
			continue
		}

		normPath := filepath.Join(dir, strings.TrimSuffix(fname, ".txt.gz")+".norm.tsv")
		out, err := os.Create(normPath)
		if err != nil {
			gzReader.Close()
			src.Close()
			continue
		}

		if err := pgs_convert.Normalize(gzReader, out, pgs_convert.Options{}); err != nil {
			gzReader.Close()
			src.Close()
			out.Close()
			continue
		}
		gzReader.Close()
		src.Close()
		out.Close()

		normPaths = append(normPaths, normPath)
	}

	// Perform scoring
	results, err := ScoreKitWithPGS(req.KitID, normPaths)
	if err != nil {
		http.Error(w, "scoring error", http.StatusInternalServerError)
		return
	}
	storeResults(req.KitID, results)
	SetStatus("ready")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// fetchFile downloads a file via HTTP GET
func fetchFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// findScoreURL looks up the FTP link for a PGS ID in LoadedScores
func findScoreURL(id string) string {
	for _, m := range data.LoadedScores {
		if sid, ok := m["Polygenic Score (PGS) ID"].(string); ok && sid == id {
			if link, ok2 := m["FTP link"].(string); ok2 {
				return link
			}
		}
	}
	return ""
}
