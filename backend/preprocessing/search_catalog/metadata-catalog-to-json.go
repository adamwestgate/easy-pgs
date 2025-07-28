// Turns the metadata catalog from PGS Catalog into json files to make catalog searching quicker.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 💾 Change these constants if your file paths change
const (
	MetadataFilePath         = "./pgs_all_metadata.xlsx"
	OntologyTraitsOutputFile = "backend/data/ontology_traits.json"
	ScoresMetadataOutputFile = "backend/data/scores_metadata.json"
)

// sheetToRecords reads a sheet into a slice of map[string]interface{}
func sheetToRecords(f *excelize.File, sheetName string) ([]map[string]interface{}, error) {
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows for sheet %s: %w", sheetName, err)
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("sheet %s is empty", sheetName)
	}
	headers := rows[0]
	var records []map[string]interface{}
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rec := make(map[string]interface{})
		for j, h := range headers {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			rec[h] = val
		}
		records = append(records, rec)
	}
	return records, nil
}

func main() {
	// Open Excel file
	f, err := excelize.OpenFile(filepath.Clean(MetadataFilePath))
	if err != nil {
		log.Fatalf("❌ Failed to open Excel file: %v", err)
	}

	// Read EFO Traits and Scores metadata
	efoTraits, err := sheetToRecords(f, "EFO Traits")
	if err != nil {
		log.Fatalf("❌ Error reading EFO Traits: %v", err)
	}
	scoresMeta, err := sheetToRecords(f, "Scores")
	if err != nil {
		log.Fatalf("❌ Error reading Scores sheet: %v", err)
	}

	// Build map of trait ID -> list of PGS IDs
	traitToPGS := make(map[string][]string)
	for _, rec := range scoresMeta {
		pgsID, _ := rec["Polygenic Score (PGS) ID"].(string)
		mapped, _ := rec["Mapped Trait(s) (EFO ID)"].(string)
		for _, tid := range strings.Split(mapped, "|") {
			tid = strings.TrimSpace(tid)
			if tid != "" {
				traitToPGS[tid] = append(traitToPGS[tid], pgsID)
			}
		}
	}

	// Attach PGS list to each trait record
	for _, rec := range efoTraits {
		id, _ := rec["Ontology Trait ID"].(string)
		if pgsList, ok := traitToPGS[id]; ok {
			rec["PGS Files"] = pgsList
		} else {
			rec["PGS Files"] = []string{}
		}
	}

	// Write outputs
	writeJSON(OntologyTraitsOutputFile, efoTraits)
	writeJSON(ScoresMetadataOutputFile, scoresMeta)

	fmt.Printf("✅ JSON files written: %s and %s\n", OntologyTraitsOutputFile, ScoresMetadataOutputFile)
}

// writeJSON marshals data and writes to a file
func writeJSON(filename string, data interface{}) {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("❌ Failed to marshal JSON for %s: %v", filename, err)
	}
	if err := os.WriteFile(filename, out, 0644); err != nil {
		log.Fatalf("❌ Failed to write file %s: %v", filename, err)
	}
}
