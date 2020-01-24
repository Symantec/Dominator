package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func mergeFiltersSubcommand(args []string, logger log.DebugLogger) error {
	if err := mergeFilters(args); err != nil {
		return fmt.Errorf("Error merging filter: %s", err)
	}
	return nil
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
