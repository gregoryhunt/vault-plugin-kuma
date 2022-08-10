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

	var token string
	// We passed the jti using InternalData from when we first created
	// the secret.
	jti, ok := req.Secret.InternalData["token_id"]
	if !ok {
		return nil, fmt.Errorf("token_id is missing on the lease")
	}

	token, ok = jti.(string)
	if !ok {
		return nil, fmt.Errorf("unable convert token_id")
	}

	b.Logger().Warn("Token revocation is not yet implemented", "jti", jti)
	if err := revokeToken(ctx, client, token); err != nil {
		return nil, fmt.Errorf("error revoking kuma token: %w", err)
	}

	return nil, nil
}

// revokeToken checks to see if a token has expired, if not it adds it to Kuma's revocation list
// if the token is expired this operation is a noop
func revokeToken(ctx context.Context, c *kumaClient, jti string) error {
	//err := c.RESTClient.DeleteRobotAccountByName(ctx, robotAccountName)

	//if err != nil {
	//	return err
	//}

	return nil
}

// tokenRenew
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
