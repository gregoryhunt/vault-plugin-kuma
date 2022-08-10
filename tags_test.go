package kuma

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsesFromString(t *testing.T) {
	tString := tagsString("kuma.io/service=backend,kuma.io/service=backend-admin")

	tMap, err := tString.ToMap()
	require.NoError(t, err)

	require.Len(t, tMap["kuma.io/service"], 2)
	require.Equal(t, tMap["kuma.io/service"][0], "backend")
	require.Equal(t, tMap["kuma.io/service"][1], "backend-admin")
}

func TestParseStringReturnsErrorOnInvalidTag(t *testing.T) {
	tString := tagsString("kuma.io/service=backend,kuma.io/service")

	_, err := tString.ToMap()
	require.Error(t, err)
}

func TestReturnsStringRepresentation(t *testing.T) {
	tm := tagsMap{
		"kuma.io/service": []string{
			"backend",
			"backend-admin",
		},
	}

	str := tm.ToString()
	require.Equal(t, "kuma.io/service=backend,kuma.io/service=backend-admin", str)
}
