package flagutil

import (
	"strings"
)

func (sl *StringList) String() string {
	return `"` + strings.Join(*sl, ",") + `"`
}

func (sl *StringList) Set(value string) error {
	if value == "" {
		*sl = make(StringList, 0)
	} else {
		*sl = strings.Split(value, ",")
	}
	return nil
}
