package kuma

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gobwas/glob"
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
			"token_name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the token, must match role value. If ommitted, value from role is used",
				Required:    false,
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

// pathCredentialsRead creates a new Kuma Token each time it is called if a
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

	name := req.GetString("token_name")

	if name == "" {
		// if name is not specified check to see if the role does not have a globbed pattern and use that, else return an error
		re := `[\{\}\!\[\]\*\?]`
		r := regexp.MustCompile(re)

		if r.MatchString(role.TokenName) {
			return logical.ErrorResponse(
				"unable to generate token, error: when token_name in the role %s contains a globbed pattern %s, you must pass the 'token_name' parameter with an absolute value when creating credentials",
				name,
				role.TokenName), nil
		}

		name = role.TokenName
	} else {
		// if name is specified check that it matches the role, role allows globbed patterns
		g := glob.MustCompile(role.TokenName)
		if !g.Match(name) {
			return logical.ErrorResponse("unable to generate token, error: token_name %s must match the globbed pattern in the role %s", name, role.TokenName), nil
		}
	}

	b.Logger().Info("Create new token for dataplane", "name", name, "operation", req.Operation)

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
	tokenType := ""

	// if tags generate a dataplane token
	if len(role.Tags) > 0 {
		b.Logger().Info("Generate dataplane token", "tags", role.Tags)
		t, err := client.dpTokenClient.Generate(name, role.Mesh, role.Tags, ProxyTypeDataplane, role.MaxTTL)

		if err != nil {
			return logical.ErrorResponse("unable to generate dataplane token, error:", err), nil
		}

		token = t
		tokenType = kumaTokenDataplane
	}

	// if groups generate a user token
	if len(role.Groups) > 0 {
		b.Logger().Info("Generate user token", "groups", role.Groups)
		t, err := client.userTokenClient.Generate(name, role.Groups, role.MaxTTL)

		if err != nil {
			return logical.ErrorResponse("unable to generate user token, error:", err), nil
		}

		token = t
		tokenType = kumaTokenUser
	}

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
	// add the jti and the expiry time in seconds to the collection so we can use if as part of the revocation process
	resp := b.Secret(kumaTokenAccountType).Response(
		map[string]interface{}{
			"token": token,
		},
		map[string]interface{}{
			"role":   roleName,
			"jti":    tokenID,
			"expiry": time.Now().Add(role.MaxTTL).UnixNano(),
			"type":   tokenType,
			"mesh":   role.Mesh,
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
