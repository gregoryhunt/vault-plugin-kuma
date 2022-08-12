package kuma

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

const (
	roleServiceName = "kuma-test-service-role"
	roleTesterName  = "kuma-tester-role"
	testTokenName   = "kuma-service"
	testerTokenName = "kuma-tester"
	testMesh        = "testMesh"
	testTTL         = float64(120)
	testMaxTTL      = float64(3600)
	testTags        = "kuma.io/service=test"
	testGroups      = "mesh-system:tester"
)

// TestUserRole uses a mock backend to check
// role create, read, update, and delete.
func TestUserRole(t *testing.T) {
	k, s := getTestBackend(t)

	t.Run("List All Roles", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			_, err := testTokenRoleCreate(t, k, s,
				roleServiceName+strconv.Itoa(i),
				map[string]interface{}{
					"token_name": testTokenName + strconv.Itoa(i),
					"tags":       testTags,
					"ttl":        testTTL,
					"max_ttl":    testMaxTTL,
				})
			require.NoError(t, err)
		}

		resp, err := testTokenRoleList(t, k, s)
		require.NoError(t, err)
		require.Len(t, resp.Data["keys"].([]string), 10)
	})

	t.Run("Create Service Role-pass", func(t *testing.T) {
		resp, err := testTokenRoleCreate(t, k, s, roleServiceName, map[string]interface{}{
			"token_name": testTokenName,
			"mesh":       testMesh,
			"tags":       testTags,
			"ttl":        testTTL,
			"max_ttl":    testMaxTTL,
		})

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.Nil(t, resp)
	})

	t.Run("Create User Role-pass", func(t *testing.T) {
		resp, err := testTokenRoleCreate(t, k, s, roleTesterName, map[string]interface{}{
			"token_name": testerTokenName,
			"mesh":       testMesh,
			"groups":     testGroups,
			"ttl":        testTTL,
			"max_ttl":    testMaxTTL,
		})

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.Nil(t, resp)
	})

	t.Run("Create Service Role-fail", func(t *testing.T) {
		resp, err := testTokenRoleCreate(t, k, s, roleServiceName, map[string]interface{}{
			"token_name": testTokenName,
			"tags":       testTags,
			"groups":     testGroups,
			"ttl":        testTTL,
			"max_ttl":    testMaxTTL,
		})

		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("Read Role", func(t *testing.T) {
		resp, err := testTokenRoleRead(t, k, s)

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.Equal(t, testTTL, resp.Data["ttl"])
	})

	t.Run("Update Role", func(t *testing.T) {
		resp, err := testTokenRoleUpdate(t, k, s, map[string]interface{}{
			"token_name": testTokenName,
			"mesh":       testMesh,
			"tags":       testTags,
			"ttl":        "1m",
			"max_ttl":    "5h",
		})

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.Nil(t, resp)
	})

	t.Run("Re-read Role", func(t *testing.T) {
		resp, err := testTokenRoleRead(t, k, s)

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.Equal(t, string(testTags), resp.Data["tags"])
	})

	t.Run("Delete Role", func(t *testing.T) {
		_, err := testTokenRoleDelete(t, k, s)

		require.NoError(t, err)
	})

}

// Utility function to create a role while, returning any response (including errors)
func testTokenRoleCreate(
	t *testing.T,
	k *kumaBackend,
	s logical.Storage,
	name string,
	d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	resp, err := k.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "roles/" + name,
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// Utility function to update a role while, returning any response (including errors)
func testTokenRoleUpdate(t *testing.T, k *kumaBackend, s logical.Storage, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	resp, err := k.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "roles/" + roleServiceName,
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return nil, err
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}
	return resp, nil
}

// Utility function to read a role and return any errors
func testTokenRoleRead(t *testing.T, k *kumaBackend, s logical.Storage) (*logical.Response, error) {
	t.Helper()
	return k.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "roles/" + roleServiceName,
		Storage:   s,
	})
}

// Utility function to list roles and return any errors
func testTokenRoleList(t *testing.T, k *kumaBackend, s logical.Storage) (*logical.Response, error) {
	t.Helper()
	return k.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "roles/",
		Storage:   s,
	})
}

// Utility function to delete a role and return any errors
func testTokenRoleDelete(t *testing.T, k *kumaBackend, s logical.Storage) (*logical.Response, error) {
	t.Helper()
	return k.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "roles/" + roleServiceName,
		Storage:   s,
	})
}

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
	require.Equal(t, kmeMap["ttl"], float64(3600))
	require.Equal(t, kmeMap["max_ttl"], float64(86400))

}
