package kuma

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDataplaneResponseData(t *testing.T) {
	kme := kumaRoleEntry{
		TokenName: "backend-1",
		Mesh:      "default",
		Tags: tagsMap{
			"kuma.io/service": []string{
				"backend",
				"backend-admin",
			},
		},
		TTL:    time.Hour,
		MaxTTL: 24 * time.Hour,
	}
	kmeMap := kme.toResponseData()
	//fmt.Println("%v", kmeMap)
	require.Len(t, kmeMap, 5)
	require.Equal(t, kmeMap["token_name"], "backend-1")
	require.Equal(t, kmeMap["mesh"], "default")
	require.Equal(t, kmeMap["tags"], "kuma.io/service=backend,kuma.io/service=backend-admin")
	require.Equal(t, kmeMap["ttl"], "1h0m0s")
	require.Equal(t, kmeMap["max_ttl"], "24h0m0s")

}
