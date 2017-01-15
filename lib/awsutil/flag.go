package awsutil

import (
	"errors"
	"strings"
)

func (list *TargetList) string() string {
	targets := make([]string, 0, len(*list))
	for _, target := range *list {
		targets = append(targets, target.AccountName+","+target.Region)
	}
	return `"` + strings.Join(targets, ";") + `"`
}

func (list *TargetList) set(value string) error {
	newList := make(TargetList, 0)
	if value == "" {
		*list = newList
		return nil
	}
	for _, target := range strings.Split(value, ";") {
		splitTarget := strings.Split(target, ",")
		if len(splitTarget) != 2 {
			return errors.New(`malformed target: "` + target + `"`)
		}
		account := splitTarget[0]
		region := splitTarget[1]
		if account == "*" {
			account = ""
		}
		if region == "*" {
			region = ""
		}
		newList = append(newList, Target{account, region})
	}
	*list = newList
	return nil
}
