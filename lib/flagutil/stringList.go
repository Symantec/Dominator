package flagutil

import (
	"strings"
)

func (sl *StringList) String() string {
	return `"` + strings.Join(*sl, ",") + `"`
}

func (sl *StringList) Set(value string) error {
	*sl = strings.Split(value, ",")
	return nil
}
