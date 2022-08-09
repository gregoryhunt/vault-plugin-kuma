package kuma

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// nolint:gosec
	pathCredsHelpSyn  = `Generate a Kuma token from a specific Vault role.`
	pathCredsHelpDesc = `This path generates a Kuma token based on a particular role.`
	dayHours          = float64(24)
)

// pathCreds extends the Vault API with a `/creds`
// endpoint for a role.
func pathCreds(b *kumaBackend) *framework.Path {
	return &framework.Path{
		Pattern: "creds/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the role",
				Required:    true,
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation:   b.pathCredsRead,
			logical.UpdateOperation: b.pathCredsRead,
		},
		HelpSynopsis:    pathCredsHelpSyn,
		HelpDescription: pathCredsHelpDesc,
	}
}

// pathCredentialsRead creates a new Harbor robot account each time it is called if a
// role exists.
func (b *kumaBackend) pathCredsRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)

	roleEntry, err := b.getRole(ctx, req.Storage, roleName)
	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if roleEntry == nil {
		return nil, errors.New("error retrieving role: role is nil")
	}

	return b.createCreds(ctx, req, roleName, roleEntry)
}

// createCreds creates a new Kuma access token to store into the Vault backend, generates
// a response with the robot account information, and checks the TTL and MaxTTL attributes.
func (b *kumaBackend) createCreds(
	ctx context.Context,
	req *logical.Request,
	roleName string,
	role *kumaRoleEntry) (*logical.Response, error) {

	var displayName string

	b.Logger().Info("Create new token", "name", req.DisplayName, "operation", req.Operation)

	if req.DisplayName != "" {
		re := regexp.MustCompile("[^[:alnum:]._-]")
		dn := re.ReplaceAllString(req.DisplayName, "-")
		displayName = fmt.Sprintf("%s.", dn)
	}

	// fetch a client instance, client is a property of backend but there is not guarantee that
	// it has been instantiated
	client, err := b.getClient(ctx, req.Storage)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("error getting Kuma client")
	}

	token := ""
	if len(role.Tags) > 0 {
		b.Logger().Info("Generate dataplane token", "tags", role.Tags)
		t, err := client.dpTokenClient.Generate(displayName, role.Mesh, role.Tags, ProxyTypeDataplane, role.MaxTTL)

		if err != nil {
			return logical.ErrorResponse("unable to generate token, error:", err), nil
		}

		token = t
	}

	// if role.Groups
	//b.client.clientTokenClient.Generate

	// parse the token to get the jti, we need this to revoke tokens
	parts := strings.Split(token, ".")

	var body map[string]interface{}
	bodyBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	json.Unmarshal(bodyBytes, &body)

	tokenID := ""
	if tid, ok := body["jti"].(string); ok {
		tokenID = tid
	} else {
		return logical.ErrorResponse("generated token does not contain a jti"), nil
	}

	// The response is divided into two objects (1) internal data and (2) data.
	resp := b.Secret(kumaTokenAccountType).Response(
		map[string]interface{}{
			"token": token,
		},
		map[string]interface{}{
			"role":     roleName,
			"token_id": tokenID,
		},
	)

	if role.TTL > 0 {
		resp.Secret.TTL = role.TTL
	}

	if role.MaxTTL > 0 {
		resp.Secret.MaxTTL = role.MaxTTL
	}

	return resp, nil
}
