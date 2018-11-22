package flagutil

import (
	"strconv"
	"strings"
)

type unitType struct {
	suffix     string
	multiplier uint64
}

var units = []unitType{ // In order of preference.
	{"EiB", 1 << 60},
	{"PiB", 1 << 50},
	{"TiB", 1 << 40},
	{"GiB", 1 << 30},
	{"MiB", 1 << 20},
	{"KiB", 1 << 10},
	{"E", 1 << 60},
	{"P", 1 << 50},
	{"T", 1 << 40},
	{"G", 1 << 30},
	{"M", 1 << 20},
	{"K", 1 << 10},
	{"B", 1},
	{"EB", 1000000000000000000},
	{"PB", 1000000000000000},
	{"TB", 1000000000000},
	{"GB", 1000000000},
	{"MB", 1000000},
	{"kB", 1000},
}

func (size *Size) String() string {
	for _, unit := range units {
		if unit.multiplier == 1 {
			continue
		}
		pretty := uint64(*size) / unit.multiplier
		if pretty*unit.multiplier == uint64(*size) {
			return strconv.FormatUint(pretty, 10) + unit.suffix
		}
	}
	return strconv.FormatUint(uint64(*size), 10) + "B"
}

func (size *Size) Set(value string) error {
	for _, unit := range units {
		if strings.HasSuffix(value, unit.suffix) {
			val, err := strconv.ParseUint(value[:len(value)-len(unit.suffix)],
				10, 64)
			if err != nil {
				return err
			} else {
				*size = Size(val * unit.multiplier)
				return nil
			}
		}
	}
	if val, err := strconv.ParseUint(value, 10, 64); err != nil {
		return err
	} else {
		*size = Size(val)
		return nil
	}
}
