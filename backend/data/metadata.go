package data

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/adamwestgate/easy-pgs/backend/config"
)

// Paths for metadata files
var (
  OntologyTraitsPath = filepath.Join(config.DataDir, config.OntologyTraitsFile)
  ScoresMetadataPath = filepath.Join(config.DataDir, config.ScoresMetadataFile)
)

// OntologyTrait mirrors one entry in ontology_traits.json
// and includes the list of associated PGS file IDs.
type OntologyTrait struct {
    ID          string   `json:"Ontology Trait ID"`
    Label       string   `json:"Ontology Trait Label"`
    Description string   `json:"Ontology Trait Description"`
    URL         string   `json:"Ontology URL"`
    PGSFiles    []string `json:"PGS Files"`
}

// LoadedScores holds the parsed JSON from scores_metadata.json
var LoadedScores []map[string]interface{}

// LoadedTraits holds the parsed JSON from ontology_traits.json
var LoadedTraits []OntologyTrait

// LoadScores reads and parses the scores metadata JSON file into LoadedScores.
// Returns an error if opening or parsing the file fails.
func LoadScores() error {
    f, err := os.Open(ScoresMetadataPath)
    if err != nil {
        return fmt.Errorf("unable to open %s: %w", ScoresMetadataPath, err)
    }
    defer f.Close()

    var tmp []map[string]interface{}
    if err := json.NewDecoder(f).Decode(&tmp); err != nil {
        return fmt.Errorf("unable to parse %s: %w", ScoresMetadataPath, err)
    }

    LoadedScores = tmp
    return nil
}

// LoadTraits reads and parses the ontology traits JSON file into LoadedTraits.
// Returns an error if opening or parsing the file fails.
func LoadTraits() error {
    f, err := os.Open(OntologyTraitsPath)
    if err != nil {
        return fmt.Errorf("unable to open %s: %w", OntologyTraitsPath, err)
    }
    defer f.Close()

    var tmp []OntologyTrait
    if err := json.NewDecoder(f).Decode(&tmp); err != nil {
        return fmt.Errorf("unable to parse %s: %w", OntologyTraitsPath, err)
    }

    LoadedTraits = tmp
    return nil
}

// LoadMetadata loads both scores and traits metadata. Returns on first error encountered.
func LoadMetadata() error {
    if err := LoadScores(); err != nil {
        return err
    }
    return LoadTraits()
}
