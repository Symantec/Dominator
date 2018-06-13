package tags

import (
	"errors"
	"strings"

	"github.com/Symantec/Dominator/lib/json"
)

func (tag *Tag) string() string {
	return tag.Key + "=" + tag.Value
}

func (tag *Tag) set(value string) error {
	splitValue := strings.Split(value, "=")
	if len(splitValue) != 2 {
		return errors.New(`malformed tag: "` + value + `"`)
	}
	*tag = Tag{splitValue[0], splitValue[1]}
	return nil
}

func (tags Tags) copy() Tags {
	newTags := make(Tags, len(tags))
	for key, value := range tags {
		newTags[key] = value
	}
	return newTags
}

func (left Tags) equal(right Tags) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if leftValue != right[key] {
			return false
		}
	}
	return true
}

func (to Tags) merge(from Tags) {
	for key, value := range from {
		to[key] = value
	}
}

func (tags *Tags) string() string {
	pairs := make([]string, 0, len(*tags))
	for key, value := range *tags {
		pairs = append(pairs, key+"="+value)
	}
	return strings.Join(pairs, ",")
}

func (tags *Tags) set(value string) error {
	newTags := make(Tags)
	if value == "" {
		*tags = newTags
		return nil
	}
	for _, tag := range strings.Split(value, ",") {
		if len(tag) < 3 {
			return errors.New(`malformed tag: "` + tag + `"`)
		}
		if tag[0] == '@' {
			var fileTags Tags
			if err := json.ReadFromFile(tag[1:], &fileTags); err != nil {
				return errors.New("error loading tags file: " + err.Error())
			}
			newTags.Merge(fileTags)
			continue
		}
		splitTag := strings.Split(tag, "=")
		if len(splitTag) != 2 {
			return errors.New(`malformed tag: "` + tag + `"`)
		}
		if splitTag[0] == "" {
			return errors.New("empty tag key")
		}
		newTags[splitTag[0]] = splitTag[1]
	}
	*tags = newTags
	return nil
}
