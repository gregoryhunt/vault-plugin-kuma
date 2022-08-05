package kuma

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathRoleHelpSynopsis    = `Manages the Vault role for generating Kuma tokens.`
	pathRoleHelpDescription = `
This path allows you to read and write roles used to generate Kuma tokens.
You can configure a role to manage a user or service token by setting the permissions field.
`

	pathRoleListHelpSynopsis    = `List the existing roles in Harbor backend`
	pathRoleListHelpDescription = `Roles will be listed by the role name.`
)

// kumaRoleEntry defines the data required
// for a Vault role to access and call the Harbor
// token endpoints
type kumaRoleEntry struct {
	Mesh   string            `json:"mesh"`
	Tags   map[string]string `json:"tags"`
	TTL    time.Duration     `json:"ttl"`
	MaxTTL time.Duration     `json:"max_ttl"`
	//Permissions []*harborModel.RobotPermission `json:"permissions"`
}

// toResponseData returns response data for a role
func (r *kumaRoleEntry) toResponseData() map[string]interface{} {
	t, _ := json.Marshal(r.Tags)
	respData := map[string]interface{}{
		"mesh":    r.Mesh,
		"tags":    string(t),
		"ttl":     r.TTL.Seconds(),
		"max_ttl": r.MaxTTL.Seconds(),
	}
	return respData
}

// pathRoles extends the Vault API with a `/roles`
// endpoint for the backend.
func pathRoles(b *kumaBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "roles/" + framework.GenericNameRegex("name"),
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:        framework.TypeLowerCaseString,
					Description: "Name of the role",
					Required:    true,
				},
				"mesh": {
					Type:        framework.TypeString,
					Description: "The Mesh to provision token in",
				},
				"tags": {
					Type:        framework.TypeString,
					Description: "The tags for the Kuma token",
					Required:    true,
				},
				"ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Default lease for generated credentials. If not set or set to 0, will use system default.",
				},
				"max_ttl": {
					Type:        framework.TypeDurationSecond,
					Description: "Maximum time for role. If not set or set to 0, will use system default.",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathRolesRead,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathRolesWrite,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathRolesWrite,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathRolesDelete,
				},
			},
			HelpSynopsis:    pathRoleHelpSynopsis,
			HelpDescription: pathRoleHelpDescription,
		},
		{
			Pattern: "roles/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathRolesList,
				},
			},
			HelpSynopsis:    pathRoleListHelpSynopsis,
			HelpDescription: pathRoleListHelpDescription,
		},
	}
}

// pathRolesList makes a request to Vault storage to retrieve a list of roles for the backend
func (b *kumaBackend) pathRolesList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entries, err := req.Storage.List(ctx, "role/")
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(entries), nil
}

// pathRolesRead makes a request to Vault storage to read a role and return response data
func (b *kumaBackend) pathRolesRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := b.getRole(ctx, req.Storage, d.Get("name").(string))
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	return &logical.Response{
		Data: entry.toResponseData(),
	}, nil
}

// pathRolesWrite makes a request to Vault storage to update a role based on the attributes passed to the role configuration
func (b *kumaBackend) pathRolesWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	name, ok := d.GetOk("name")
	if !ok {
		return logical.ErrorResponse("missing role name"), nil
	}

	roleEntry, err := b.getRole(ctx, req.Storage, name.(string))
	if err != nil {
		return nil, err
	}

	if roleEntry == nil {
		roleEntry = &kumaRoleEntry{}
	}

	createOperation := (req.Operation == logical.CreateOperation)

	if tags, ok := d.GetOk("tags"); ok {
		parsedTags := make(map[string]string, 0) // non-nil to avoid a "missing tags" error later

		err := jsonutil.DecodeJSON([]byte(tags.(string)), &parsedTags)
		if err != nil {
			return logical.ErrorResponse("error parsing Tags '%s': %s", tags.(string), err.Error()), nil
		}
		roleEntry.Tags = parsedTags
	} else if !ok && createOperation {
		return nil, fmt.Errorf("missing permissions in role")
	}

	if ttlRaw, ok := d.GetOk("ttl"); ok {
		roleEntry.TTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if createOperation {
		roleEntry.TTL = time.Duration(d.Get("ttl").(int)) * time.Second
	}

	if maxTTLRaw, ok := d.GetOk("max_ttl"); ok {
		roleEntry.MaxTTL = time.Duration(maxTTLRaw.(int)) * time.Second
	} else if createOperation {
		roleEntry.MaxTTL = time.Duration(d.Get("max_ttl").(int)) * time.Second
	}

	if roleEntry.MaxTTL != 0 && roleEntry.TTL > roleEntry.MaxTTL {
		return logical.ErrorResponse("ttl cannot be greater than max_ttl"), nil
	}

	if err := setRole(ctx, req.Storage, name.(string), roleEntry); err != nil {
		return nil, err
	}

	return nil, nil
}

// pathRolesDelete makes a request to Vault storage to delete a role
func (b *kumaBackend) pathRolesDelete(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	err := req.Storage.Delete(ctx, "role/"+d.Get("name").(string))
	if err != nil {
		return nil, fmt.Errorf("error deleting harbor role: %w", err)
	}

	return nil, nil
}

// setRole adds the role to the Vault storage API
func setRole(ctx context.Context, s logical.Storage, name string, roleEntry *kumaRoleEntry) error {
	entry, err := logical.StorageEntryJSON("role/"+name, roleEntry)
	if err != nil {
		return err
	}

	if entry == nil {
		return fmt.Errorf("failed to create storage entry for role")
	}

	if err := s.Put(ctx, entry); err != nil {
		return err
	}

	return nil
}

// getRole gets the role from the Vault storage API
func (b *kumaBackend) getRole(ctx context.Context, s logical.Storage, name string) (*kumaRoleEntry, error) {
	if name == "" {
		return nil, fmt.Errorf("missing role name")
	}

	entry, err := s.Get(ctx, "role/"+name)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil
	}

	var role kumaRoleEntry

	if err := entry.DecodeJSON(&role); err != nil {
		return nil, err
	}
	return &role, nil
}
