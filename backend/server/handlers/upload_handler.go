// backend/server/handlers/upload_handler.go
package handlers

import (
    "crypto/rand"
    "encoding/hex"
	"encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/google/uuid"

    kit_convert "github.com/adamwestgate/easy-pgs/backend/preprocessing/kit_convert"
    "github.com/adamwestgate/easy-pgs/backend/config"
)

// Directories for user kits
const (
    maxBodySize        = 32 << 20 // 32 MiB
    maxMemParse        = 8 << 20  // 8 MiB
)

func init() {
    for _, p := range []string{config.UploadRawDir, config.UploadProcessedDir} {
        if err := os.MkdirAll(p, 0o755); err != nil {
            panic("failed to create directory: " + err.Error())
        }
    }
}

// UploadKitHandler handles POST /upload.
// It processes a user-uploaded DNA kit, converts it to PLINK2 format using kit_convert,
// and stores the resulting data via the configured KitStore.
// The response includes a unique kit ID and kit type (e.g., Ancestry or 23andMe).
func UploadKitHandler(w http.ResponseWriter, r *http.Request) {
    if kitStore == nil {
        http.Error(w, "server mis-config: kitStore not set", http.StatusInternalServerError)
        return
    }

    // 1. Guard rails & parse multipart form
    r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
    if err := r.ParseMultipartForm(maxMemParse); err != nil {
        http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
        return
    }

    // 2. Retrieve uploaded file
    file, header, err := r.FormFile("kit")
    if err != nil {
        http.Error(w, "Missing kit file", http.StatusBadRequest)
        return
    }
    defer file.Close()
    log.Printf("⇢  upload: received %q\n", header.Filename)

    // 3. Save raw kit
    base := uniqueFilename(filepath.Base(header.Filename))
    rawPath := filepath.Join(config.UploadRawDir, base)
    if err := saveFileToPath(rawPath, file); err != nil {
        http.Error(w, "Failed to save raw kit", http.StatusInternalServerError)
        return
    }
    log.Printf("✓  saved raw kit → %s\n", rawPath)

    // 4. Generate unique IDs: one for DB key, one for folder name
    kitKey := uuid.NewString()       // key for storage in DB
    folderID := uuid.NewString()     // folder under processed
    processedDir := filepath.Join(config.UploadProcessedDir, folderID)
    if err := os.MkdirAll(processedDir, 0o755); err != nil {
        http.Error(w, "Failed to create processed directory", http.StatusInternalServerError)
        return
    }
    log.Printf("• preparing processed directory → %s\n", processedDir)

    // 5. Convert to PLINK2 binary format ➜ get kitType back
    log.Println("• normalise: converting to PLINK2 binary format")
    kitType, err := kit_convert.ConvertFileToPgen(rawPath, processedDir)
    if err != nil {
        http.Error(w, "PLINK2 conversion failed: "+err.Error(), http.StatusBadRequest)
        return
    }
    log.Printf("✓  processed kit files in → %s (type=%s)\n", processedDir, kitType)

    // 6. Persist mapping
    if err := kitStore.Insert(kitKey, processedDir, kitType); err != nil {
        http.Error(w, "Store error: "+err.Error(), http.StatusInternalServerError)
        return
    }
    log.Printf("✓  stored mapping %s → %s (type=%s)\n", kitKey, processedDir, kitType)


	// 7. JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"kit_id":        kitKey,
		"kit_type":      kitType,
	})
	log.Printf("⇠  upload complete, kit_id=%s\n", kitKey)

}

// uniqueFilename generates a timestamped random filename prefix.
func uniqueFilename(orig string) string {
    b := make([]byte, 4)
    if _, err := rand.Read(b); err != nil {
        panic(err)
    }
    return fmt.Sprintf("%d_%s_%s", time.Now().UnixNano(), hex.EncodeToString(b), orig)
}

// saveFileToPath writes a file in src reader to a file at the specified path
func saveFileToPath(dstPath string, src io.Reader) error {
    dst, err := os.Create(dstPath)
    if err != nil {
        return err
    }
    defer dst.Close()
    if seeker, ok := src.(io.Seeker); ok {
        seeker.Seek(0, io.SeekStart)
    }
    _, err = io.Copy(dst, src)
    return err
}

// stripGZExt removes a trailing .gz extension.
func stripGZExt(name string) string {
    return strings.TrimSuffix(name, ".gz")
}
