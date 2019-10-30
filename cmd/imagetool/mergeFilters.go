package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/filter"
)

func mergeFiltersSubcommand(args []string) {
	err := mergeFilters(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging filter: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func mergeFilters(filterFiles []string) error {
	mergeableFilter := &filter.MergeableFilter{}
	for _, filterFile := range filterFiles {
		filt, err := filter.Load(filterFile)
		if err != nil {
			return err
		}
		mergeableFilter.Merge(filt)
	}
	filt := mergeableFilter.ExportFilter()
	for _, filterLine := range filt.FilterLines {
		if _, err := fmt.Println(filterLine); err != nil {
			return err
		}
	}
	return nil
}
