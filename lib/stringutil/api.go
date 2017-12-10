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

func NewStringDeduplicator(lock bool) *StringDeduplicator {
	return &StringDeduplicator{lock: lock, mapping: make(map[string]string)}
}

func (d *StringDeduplicator) Clear() {
	d.clear()
}

func (d *StringDeduplicator) DeDuplicate(str string) string {
	return d.deDuplicate(str)
}

func (d *StringDeduplicator) GetStatistics() StringDuplicationStatistics {
	return d.getStatistics()
}
