package kuma

import (
	"context"
	"fmt"
	"sync"

	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
	"github.com/hashicorp/vault/sdk/database/helper/connutil"
)

// kumaConnectionProducer implements ConnectionProducer and provides an
// interface for kuma databases to make connections.
type kumaConnectionProducer struct {
	Username      string `json:"username" structs:"username"`
	Password      string `json:"password" structs:"password"`
	ConnectionURL string `json:"connection_url" structs:"connection_url"`

	rawConfig map[string]interface{}

	Initialized bool
	Type        string
	client      *KumaClient
	sync.Mutex
}

func (p *kumaConnectionProducer) secretValues() map[string]string {
	return map[string]string{
		p.Username: "[username]",
		p.Password: "[password]",
	}
}

func (p *kumaConnectionProducer) Initialize(ctx context.Context, req dbplugin.InitializeRequest) (dbplugin.InitializeResponse, error) {
	p.Lock()
	defer p.Unlock()

	p.rawConfig = req.Config

	// Set initialized to true at this point since all fields are set,
	// and the connection can be established at a later time.
	p.Initialized = true

	if req.VerifyConnection {
		if _, err := p.Connection(ctx); err != nil {
			return dbplugin.InitializeResponse{}, fmt.Errorf("error verifying connection: %w", err)
		}
	}

	resp := dbplugin.InitializeResponse{
		Config: req.Config,
	}

	return resp, nil
}

// Connection creates a database connection
func (m *kumaConnectionProducer) Connection(ctx context.Context) (*KumaClient, error) {
	if !m.Initialized {
		return nil, connutil.ErrNotInitialized
	}

	// If we already have a DB, return it
	if m.client != nil {
		return m.client, nil
	}

	client, err := m.createClient()
	if err != nil {
		return nil, err
	}

	// Store the client for later use
	m.client = client

	return m.client, nil
}

func (m *kumaConnectionProducer) createClient() (*KumaClient, error) {
	c, err := NewKumaClient(m.ConnectionURL, m.Username, m.Password)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Close terminates the database connection.
func (m *kumaConnectionProducer) Close() error {
	m.Lock()
	defer m.Unlock()

	m.client = nil

	return nil
}
