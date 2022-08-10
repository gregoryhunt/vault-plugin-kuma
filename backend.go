package kuma

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const backendHelp = `
The kuma secrets backend dynamically generates user and service tokens.
After mounting this backend, credentials to manage harbor user and service tokens
must be configured with the "config/" endpoints.
`

// Factory configures and returns Kuma secrets backends.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

// kumaBackend defines an object that
// extends the Vault backend and stores the
// target API's client.
type kumaBackend struct {
	*framework.Backend
	lock   sync.RWMutex
	client *kumaClient
}

// backend defines the target API backend
// for Vault. It must include each path
// and the secrets it will store.
func backend() *kumaBackend {
	var b = kumaBackend{}

	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),
		PathsSpecial: &logical.Paths{
			LocalStorage: []string{},
			SealWrapStorage: []string{
				"config",
				"roles/*",
			},
		},
		Paths: framework.PathAppend(
			pathRoles(&b),
			[]*framework.Path{
				pathConfig(&b),
				pathCreds(&b),
			},
		),
		Secrets: []*framework.Secret{
			b.kumaToken(),
		},
		BackendType: logical.TypeLogical,
		Invalidate:  b.invalidate,
	}
	return &b
}

// invalidate clears an existing client configuration in
// the backend
func (b *kumaBackend) invalidate(ctx context.Context, key string) {
	if key == "config" {
		b.reset()
	}
}

func (b *kumaBackend) reset() {
	b.lock.RLock()
	unlockFunc := b.lock.RUnlock

	// nolint:gocritic
	defer func() { unlockFunc() }()

	b.client = nil
}

// setupClient locks the backend as it configures and creates a
// a new client for the target API
func (b *kumaBackend) getClient(ctx context.Context, s logical.Storage) (*kumaClient, error) {
	b.lock.RLock()
	unlockFunc := b.lock.RUnlock

	defer unlockFunc()

	if b.client != nil {
		return b.client, nil
	}

	config, err := getConfig(ctx, s)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = new(kumaConfig)
	}

	b.client, err = newClient(config)
	if err != nil {
		return nil, err
	}

	return b.client, nil
}
