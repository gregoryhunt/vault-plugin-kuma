package kuma

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// nolint:gosec
	pathCredsHelpSyn  = `Generate a Harbor robot account from a specific Vault role.`
	pathCredsHelpDesc = `This path generates a Harbor robot account
based on a particular role.`

	dayHours = float64(24)
)

// harborRobotAccount defines a secret for the Harbor token
type harborRobotAccount struct {
	ID        int64  `json:"robot_account_id"`
	Name      string `json:"robot_account_name"`
	Secret    string `json:"robot_account_secret"`
	AuthToken string `json:"robot_account_auth_token"`
}

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

// createCreds creates a new Harbor robot account to store into the Vault backend, generates
// a response with the robot account information, and checks the TTL and MaxTTL attributes.
func (b *kumaBackend) createCreds(
	ctx context.Context,
	req *logical.Request,
	roleName string,
	role *harborRoleEntry) (*logical.Response, error) {
	var displayName string

	if req.DisplayName != "" {
		re := regexp.MustCompile("[^[:alnum:]._-]")
		dn := re.ReplaceAllString(req.DisplayName, "-")
		displayName = fmt.Sprintf("%s.", dn)
	}

	robotAccountName := fmt.Sprintf("vault.%s.%s%d", roleName, displayName, time.Now().UnixNano())

	//	robotAccount, err := b.createRobotAccount(ctx, req.Storage, robotAccountName, role)
	//	if err != nil {
	//		return nil, err
	//	}

	// The response is divided into two objects (1) internal data and (2) data.
	resp := b.Secret(harborRobotAccountType).Response(map[string]interface{}{}, map[string]interface{}{
		"role":               roleName,
		"robot_account_name": robotAccountName,
	})

	if role.TTL > 0 {
		resp.Secret.TTL = role.TTL
	}

	if role.MaxTTL > 0 {
		resp.Secret.MaxTTL = role.MaxTTL
	}

	return resp, nil
}
