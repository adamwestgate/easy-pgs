package handlers

import "github.com/adamwestgate/easy-pgs/backend/store"

// kitStore holds the BoltDB-backed KitStore; all handlers use this.
var kitStore store.KitStore

// SetKitStore initializes the package‐level store.
func SetKitStore(ks store.KitStore) {
    kitStore = ks
}
