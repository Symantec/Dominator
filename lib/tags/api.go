package tags

type Tag struct {
	Key   string
	Value string
}

func (tag *Tag) String() string {
	return tag.string()
}

func (tag *Tag) Set(value string) error {
	return tag.set(value)
}

type Tags map[string]string // Key: tag key, value: tag value.

func (tags Tags) Copy() Tags {
	return tags.copy()
}

func (left Tags) Equal(right Tags) bool {
	return left.equal(right)
}

func (to Tags) Merge(from Tags) {
	to.merge(from)
}

func (tags *Tags) String() string {
	return tags.string()
}

func (tags *Tags) Set(value string) error {
	return tags.set(value)
}
