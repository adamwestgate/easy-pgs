package boltstore

import (
    "encoding/json"
    "path/filepath"

    "go.etcd.io/bbolt"

    "github.com/adamwestgate/easy-pgs/backend/store"
)

const bucket = "kits"

// kitRecord bundles the on-disk metadata.
type kitRecord struct {
    Path string `json:"path"`
    Type string `json:"type"` // e.g. "ancestry" or "23andme"
}

// Store is a Bolt-backed KitStore.
type Store struct{ db *bbolt.DB }

// Open opens (or creates) backend/data/kits.db and ensures the “kits” bucket.
func Open(dir string) (store.KitStore, error) {
    db, err := bbolt.Open(filepath.Join(dir, "kits.db"), 0o600, nil)
    if err != nil {
        return nil, err
    }
    if err := db.Update(func(tx *bbolt.Tx) error {
        _, e := tx.CreateBucketIfNotExists([]byte(bucket))
        return e
    }); err != nil {
        return nil, err
    }
    return &Store{db: db}, nil
}

// Insert stores the JSON‐encoded kitRecord under key=id.
func (s *Store) Insert(id, processedPath, kitType string) error {
    rec := kitRecord{Path: processedPath, Type: kitType}
    data, err := json.Marshal(rec)
    if err != nil {
        return err
    }
    return s.db.Update(func(tx *bbolt.Tx) error {
        return tx.Bucket([]byte(bucket)).Put([]byte(id), data)
    })
}

// Lookup returns (path, type, true) if the key exists and unmarshalling succeeds.
func (s *Store) Lookup(id string) (string, string, bool) {
    var data []byte
    _ = s.db.View(func(tx *bbolt.Tx) error {
        data = tx.Bucket([]byte(bucket)).Get([]byte(id))
        return nil
    })
    if data == nil {
        return "", "", false
    }
    var rec kitRecord
    if err := json.Unmarshal(data, &rec); err != nil {
        return "", "", false
    }
    return rec.Path, rec.Type, true
}

// Delete removes the record for this kitID.
func (s *Store) Delete(id string) error {
    return s.db.Update(func(tx *bbolt.Tx) error {
        return tx.Bucket([]byte(bucket)).Delete([]byte(id))
    })
}
