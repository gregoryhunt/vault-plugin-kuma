package kuma

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	kumaTokenAccountType = "kuma_token"
)

// kumaToken defines a secret to store for a given role
// and how it should be revoked or renewed.
func (b *kumaBackend) kumaToken() *framework.Secret {
	return &framework.Secret{
		Type: kumaTokenAccountType,
		Fields: map[string]*framework.FieldSchema{
			"kuma_token": {
				Type:        framework.TypeString,
				Description: "Kuma access token",
			},
		},
		Revoke: b.tokenRevoke,
		Renew:  b.tokenRenew,
	}
}

// tokenRevoke removes the token from the Vault storage API and calls the client to revoke the robot account
func (b *kumaBackend) tokenRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("error getting Kuma client")
	}

	var account string
	// We passed the account using InternalData from when we first created
	// the secret. This is because the Harbor API uses the exact robot account name
	// for revocation.
	accountRaw, ok := req.Secret.InternalData["kuma_token_name"]
	if !ok {
		return nil, fmt.Errorf("kuma_token_name is missing on the lease")
	}

	account, ok = accountRaw.(string)
	if !ok {
		return nil, fmt.Errorf("unable convert kuma_token_name")
	}

	if err := deleteRobotAccount(ctx, client, account); err != nil {
		return nil, fmt.Errorf("error revoking kuma token: %w", err)
	}

	return nil, nil
}

// deleteToken calls the Harbor client to delete the robot account
func deleteRobotAccount(ctx context.Context, c *kumaClient, robotAccountName string) error {
	//err := c.RESTClient.DeleteRobotAccountByName(ctx, robotAccountName)

	//if err != nil {
	//	return err
	//}

	return nil
}

// robotAccountRenew
func (b *kumaBackend) tokenRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleRaw, ok := req.Secret.InternalData["role"]
	if !ok {
		return nil, fmt.Errorf("secret is missing role internal data")
	}

	// get the role entry
	role := roleRaw.(string)
	roleEntry, err := b.getRole(ctx, req.Storage, role)
	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if roleEntry == nil {
		return nil, errors.New("error retrieving role: role is nil")
	}

	resp := &logical.Response{Secret: req.Secret}

	if roleEntry.TTL > 0 {
		resp.Secret.TTL = roleEntry.TTL
	}
	if roleEntry.MaxTTL > 0 {
		resp.Secret.MaxTTL = roleEntry.MaxTTL
	}

	return resp, nil
}
