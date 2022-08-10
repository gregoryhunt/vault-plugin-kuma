package kuma

import (
	"fmt"
	"strings"
)

type tagsString string

func (t tagsString) ToMap() (map[string][]string, error) {
	parsedTags := make(map[string][]string, 0)

	ta := strings.Split(string(t), ",")
	for _, tag := range ta {
		parts := strings.Split(tag, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag %s, tags should be specified as key=value", t)
		}

		val, ok := parsedTags[parts[0]]
		if !ok {
			parsedTags[parts[0]] = []string{parts[1]}
		} else {
			parsedTags[parts[0]] = append(val, parts[1])
		}
	}

	return parsedTags, nil
}

type tagsMap map[string][]string

func (t tagsMap) ToString() string {
	rs := []string{}

	for k, vals := range t {
		for _, v := range vals {
			rs = append(rs, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return strings.Join(rs, ",")
}
