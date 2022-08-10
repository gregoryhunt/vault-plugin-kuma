package kuma

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGeneratesUserToken(t *testing.T) {
	if os.Getenv("TEST_ACC") != "1" {
		t.Skip()
	}

	config := &kumaConfig{
		Token: os.Getenv("KUMA_TOKEN"),
		URL:   "http://localhost:5681",
	}

	c, err := newClient(config)
	require.NoError(t, err)

	token, err := c.userTokenClient.Generate("test", []string{"mesh-system:admin"}, 24*time.Hour)
	require.NoError(t, err)

	require.NotEmpty(t, token)
}

func TestGeneratesDataplaneToken(t *testing.T) {
	if os.Getenv("TEST_ACC") != "1" {
		t.Skip()
	}

	config := &kumaConfig{
		Token: os.Getenv("KUMA_TOKEN"),
		URL:   "http://localhost:5681",
	}

	c, err := newClient(config)
	require.NoError(t, err)

	token, err := c.dpTokenClient.Generate("test", "default", map[string][]string{"mine": []string{"hola"}}, ProxyTypeDataplane, 24*time.Hour)
	require.NoError(t, err)

	require.NotEmpty(t, token)
}
