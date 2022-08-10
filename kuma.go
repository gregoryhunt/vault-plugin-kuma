package kuma

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	kumaTokenAccountType                = "kuma_token"
	kumaRevocationSecret                = "kuma_revocations"
	kumaTokenUser                       = "token_user"
	kumaTokenDataplane                  = "token_dataplane"
	kumaGlobalSecretDataplaneRevocation = "dataplane-token-revocations-"
	kumaGlobalSecretUserRevocation      = ""
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

	var tokenJTI string
	// We passed the jti using InternalData from when we first created
	// the secret.
	jti, ok := req.Secret.InternalData["jti"]
	if !ok {
		return nil, fmt.Errorf("jti is missing on the lease")
	}

	tokenJTI, ok = jti.(string)
	if !ok {
		return nil, fmt.Errorf("unable convert jti")
	}

	var tokenMesh string
	// We passed the mesh using InternalData from when we first created
	// the secret.
	mes, ok := req.Secret.InternalData["mesh"]
	if !ok {
		return nil, fmt.Errorf("mesh is missing on the lease")
	}

	tokenMesh, ok = mes.(string)
	if !ok {
		return nil, fmt.Errorf("unable convert mesh")
	}

	var tokenType string
	// We passed the jti using InternalData from when we first created
	// the secret.
	typ, ok := req.Secret.InternalData["type"]
	if !ok {
		return nil, fmt.Errorf("type is missing on the lease")
	}

	tokenType, ok = typ.(string)
	if !ok {
		return nil, fmt.Errorf("unable convert type")
	}

	var tokenExpiry int64
	// We passed the jti using InternalData from when we first created
	// the secret.
	exp, ok := req.Secret.InternalData["expiry"]
	if !ok {
		return nil, fmt.Errorf("expiry is missing on the lease")
	}

	// convert the expiry
	switch exp.(type) {
	case int:
		tokenExpiry = int64(exp.(int))
	case int64:
		tokenExpiry = exp.(int64)
	case float64:
		tokenExpiry = int64(exp.(float64))
	default:
		return nil, fmt.Errorf("unable to convert expiry to int64 %s", reflect.TypeOf(exp))
	}

	// if the ttl on the token has not elapsed we need to add this token to
	// the revocation list
	b.Logger().Info("Revoking Token", "jti", tokenJTI, "type", tokenType, "mesh", tokenMesh, "expiry", time.Unix(0, tokenExpiry).String())

	if time.Now().Sub(time.Unix(0, tokenExpiry)) < 0 {
		b.Logger().Info("Token has not expired, revoke token in Kuma API", "jti", tokenJTI, "type", tokenType, "mesh", tokenMesh, "expiry", time.Unix(0, tokenExpiry).String())

		if err := revokeToken(ctx, client, req.Storage, tokenType, tokenJTI, tokenMesh, tokenExpiry); err != nil {
			return nil, fmt.Errorf("error revoking kuma token: %w", err)
		}
	}

	return nil, nil
}

type RevocationList struct {
	Tokens []RevocationToken `json:"tokens"`
}

type RevocationToken struct {
	JTI    string `json:"jti"`
	Mesh   string `json:"mesh"`
	Expiry int64  `json:"expiry"`
}

// revokeToken adds the jti for the token to the revocation list and
func revokeToken(ctx context.Context, c *kumaClient, storage logical.Storage, tokenType, jti, mesh string, expiry int64) error {
	revList := &RevocationList{}

	// first get the existing revocation list secret
	if tokenType == kumaTokenDataplane {
		err := revokeDataPlaneToken(ctx, c, jti, mesh)
		if err != nil {
			return err
		}
	}

	// we now need to add the token details to the internal secret, so that we can clean the
	// revocation list later
	se, err := storage.Get(ctx, kumaRevocationSecret)
	if err != nil {
		return fmt.Errorf("unable to get revocations from internal storage: %s", err)
	}

	if se != nil {
		se.DecodeJSON(revList)
	}

	// add this token to the list
	revList.Tokens = append(
		revList.Tokens,
		RevocationToken{
			JTI:    jti,
			Mesh:   mesh,
			Expiry: expiry,
		})

	// update the secret
	se, err = logical.StorageEntryJSON(kumaRevocationSecret, revList)
	if err != nil {
		return fmt.Errorf("unable to marshal revocation list: %s", err)
	}

	err = storage.Put(ctx, se)
	if err != nil {
		return fmt.Errorf("unable to store revocation list: %s", err)
	}

	return nil
}

func revokeDataPlaneToken(ctx context.Context, c *kumaClient, jti, mesh string) error {
	jtis := []string{}
	data, err := c.secretClient.Get(kumaGlobalSecretDataplaneRevocation + mesh)
	if err != nil && err != GlobalSecretNotFound {
		return fmt.Errorf("unable to get revocation token secret: %s", err)
	}

	jtiBytes, _ := base64.StdEncoding.DecodeString(data)
	jtis = strings.Split(string(jtiBytes), ",")

	jtis = append(jtis, jti)

	data = base64.StdEncoding.EncodeToString([]byte(strings.Join(jtis, ",")))
	err = c.secretClient.Put(kumaGlobalSecretDataplaneRevocation+mesh, string(data))
	if err != nil {
		return fmt.Errorf("unable to add jti to revocation list: %s", err)
	}

	return nil
}

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
