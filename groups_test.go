package kuma

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupsStringReturnsList(t *testing.T) {
	gs := groupsString("abc,123")

	s := gs.ToList()
	require.Len(t, s, 2)

	require.Equal(t, "123", s[1])
}

func TestGroupsListReturnsString(t *testing.T) {
	gl := groupsList{"abc", "123"}

	s := gl.ToString()

	require.Equal(t, "abc,123", s)
}
