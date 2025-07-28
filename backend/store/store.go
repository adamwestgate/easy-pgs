package store

// KitStore persists a mapping <kitID → (path, type)>.
type KitStore interface {
    // Insert saves the processedPath and kitType for this kitID.
    Insert(id, processedPath, kitType string) error
    // Lookup returns (processedPath, kitType, true) if found.
    Lookup(id string) (processedPath, kitType string, ok bool)
    // Delete removes the record for this kitID.
    Delete(id string) error
}
