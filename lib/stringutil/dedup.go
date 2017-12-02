package stringutil

func (d *StringDeduplicator) clear() {
	if d.lock {
		d.mutex.Lock()
		defer d.mutex.Unlock()
	}
	d.mapping = make(map[string]string)
	d.statistics = StringDuplicationStatistics{}
}

func (d *StringDeduplicator) deDuplicate(str string) string {
	if d.lock {
		d.mutex.Lock()
		defer d.mutex.Unlock()
	}
	if cached, ok := d.mapping[str]; ok {
		d.statistics.DuplicateBytes += uint64(len(str))
		d.statistics.DuplicateStrings++
		return cached
	} else {
		d.mapping[str] = str
		d.statistics.UniqueBytes += uint64(len(str))
		d.statistics.UniqueStrings++
		return str
	}
}

func (d *StringDeduplicator) getStatistics() StringDuplicationStatistics {
	if d.lock {
		d.mutex.Lock()
		defer d.mutex.Unlock()
	}
	return d.statistics
}
