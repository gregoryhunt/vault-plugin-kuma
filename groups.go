package kuma

import "strings"

type groupsString string

func (g groupsString) ToList() []string {
	return strings.Split(string(g), ",")
}

type groupsList []string

func (g groupsList) ToString() string {
	return strings.Join(g, ",")
}
