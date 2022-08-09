package kuma

import (
	"context"
	"fmt"
	"time"

	"github.com/gobwas/glob"
	"github.com/hashicorp/vault/sdk/framework"
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
	DataplaneName string        `json:"dataplane_name"`
	Mesh          string        `json:"mesh"`
	Tags          tagsMap       `json:"tags"`
	Group         []string      `json:"groups"`
	TTL           time.Duration `json:"ttl"`
	MaxTTL        time.Duration `json:"max_ttl"`
}

// toResponseData returns response data for a role
func (r *kumaRoleEntry) toResponseData() map[string]interface{} {
	t := r.Tags.ToString()

	respData := map[string]interface{}{
		"dataplane_name": r.DataplaneName,
		"mesh":           r.Mesh,
		"tags":           t,
		"ttl":            r.TTL.String(),
		"max_ttl":        r.MaxTTL.String(),
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
				"dataplane_name": {
					Type:        framework.TypeLowerCaseString,
					Description: "Name of the dataplane, can contain globbed matches i.e backend-*",
					Required:    true,
				},
				"mesh": {
					Type:        framework.TypeString,
					Description: "The Mesh to provision token in",
				},
				"tags": {
					Type:        framework.TypeString,
					Description: "The tags for the dataplane token, specified as comma separated key value pairs",
					Required:    true,
				},
				"group": {
					Type:        framework.TypeString,
					Description: "The groups for the user token",
					Required:    false,
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

	if name, ok := d.GetOk("dataplane_name"); ok {
		roleEntry.DataplaneName = name.(string)

		// check if the name contains a globbed pattern that it compiles
		_, err := glob.Compile(roleEntry.DataplaneName)
		if err != nil {
			return logical.ErrorResponse("dataplane_name %s contains an invalid pattern: %s", roleEntry.DataplaneName, err), logical.ErrInvalidRequest
		}

	} else {
		return logical.ErrorResponse("missing dataplane_name in role"), logical.ErrInvalidRequest
	}

	if mesh, ok := d.GetOk("mesh"); ok {
		roleEntry.Mesh = mesh.(string)
	} else if createOperation {
		roleEntry.Mesh = "default"
	}

	if t, ok := d.GetOk("tags"); ok {
		parsedTags, err := tagsString(t.(string)).ToMap()
		if err != nil {
			return logical.ErrorResponse("unable to parse tags", "tags", t), logical.ErrInvalidRequest
		}

		roleEntry.Tags = parsedTags
	} else {
		return logical.ErrorResponse("missing permissions in role"), logical.ErrInvalidRequest
	}

	if ttlRaw, ok := d.GetOk("ttl"); ok {
		roleEntry.TTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if createOperation {
		// if we do not pass a value and are doing a create, set the default to 24hrs
		roleEntry.TTL = 24 * time.Hour
	}

	if ttlRaw, ok := d.GetOk("max_ttl"); ok {
		roleEntry.MaxTTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if createOperation {
		// if we do not pass a value and are doing a create, set the default to 24hrs
		roleEntry.MaxTTL = 24 * time.Hour
	}

	if roleEntry.MaxTTL != 0 && roleEntry.TTL > roleEntry.MaxTTL {
		return logical.ErrorResponse("ttl cannot be greater than max_ttl"), logical.ErrInvalidRequest
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
