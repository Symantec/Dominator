/*
	Package verstr supports comparing and sorting of version strings.

	Version strings contain substrings of numbers which should be sorted
	numerically rather than lexographically. For example, "file.9.ext" and
	"file.10.ext" sort lexographically such that "file.10.ext" comes first which
	is generally not the desired result. When treated as version strings,
	"file.9.ext" should come first.
*/
package verstr

// Less compares two version strings and returns true if left should be
// considered lesser than right.
func Less(left, right string) bool {
	return less(left, right)
}

// Sort sorts a slice of strings in-place, treating them as version strings.
func Sort(list []string) {
	sort(list)
}
