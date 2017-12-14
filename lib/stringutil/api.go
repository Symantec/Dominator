package stringutil

import "sync"

type StringDeduplicator struct {
	lock       bool
	mutex      sync.Mutex
	mapping    map[string]string
	statistics StringDuplicationStatistics
}

type StringDuplicationStatistics struct {
	DuplicateBytes   uint64
	DuplicateStrings uint64
	UniqueBytes      uint64
	UniqueStrings    uint64
}

// NewStringDeduplicator will create a StringDeduplicator which may be used to
// eliminate duplicate string contents. It maintains an internal map of unique
// strings. If lock is true then each method call will take an exclusive lock.
func NewStringDeduplicator(lock bool) *StringDeduplicator {
	return &StringDeduplicator{lock: lock, mapping: make(map[string]string)}
}

// Clear will clear the internal map and statistics.
func (d *StringDeduplicator) Clear() {
	d.clear()
}

// DeDuplicate will return a string which has the same contents as str. This
// method should be called for every string in the application.
func (d *StringDeduplicator) DeDuplicate(str string) string {
	return d.deDuplicate(str)
}

// GetStatistics will return de-duplication statistics.
func (d *StringDeduplicator) GetStatistics() StringDuplicationStatistics {
	return d.getStatistics()
}
