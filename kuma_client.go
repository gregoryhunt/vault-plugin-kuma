package kuma

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kumahq/kuma/app/kumactl/pkg/client"
	"github.com/kumahq/kuma/app/kumactl/pkg/tokens"
	config_proto "github.com/kumahq/kuma/pkg/config/app/kumactl/v1alpha1"
	userclient "github.com/kumahq/kuma/pkg/plugins/authn/api-server/tokens/ws/client"
	util_http "github.com/kumahq/kuma/pkg/util/http"
)

// harborClient creates an object storing
// the client.
type kumaClient struct {
	dpTokenClient      tokens.DataplaneTokenClient
	userTokenClient    userclient.UserTokenClient
	globalSecretClient *GlobalSecretsClient
	meshSecretClient   *SecretsClient
}

type ProxyType string

const (
	ProxyTypeDataplane = "dataplane"
	ProxyTypeIngress   = "ingress"
)

// newClient creates a new client to access harbor
// and exposes it for any secrets or roles to use.
func newClient(config *kumaConfig) (*kumaClient, error) {
	if config == nil {
		return nil, errors.New("client configuration was nil")
	}

	if config.Token == "" {
		return nil, errors.New("client token is not defined")
	}

	if config.URL == "" {
		return nil, errors.New("client API server URL is not defined")
	}

	apiclient, err := baseAPIServerClient(config.URL, config.Token)
	if err != nil {
		return nil, fmt.Errorf("unable to create base API client: %s", err)
	}

	dpc := tokens.NewDataplaneTokenClient(apiclient)
	uc := userclient.NewHTTPUserTokenClient(apiclient)
	gs := NewGlobalSecretsClient(apiclient)
	ms := NewSecretsClient(apiclient)

	return &kumaClient{dpc, uc, gs, ms}, nil
}

func baseAPIServerClient(url, token string) (util_http.Client, error) {

	conf := &config_proto.ControlPlaneCoordinates_ApiServer{}
	conf.Url = url
	conf.Headers = []*config_proto.ControlPlaneCoordinates_Headers{
		&config_proto.ControlPlaneCoordinates_Headers{
			Key:   "authorization",
			Value: "Bearer " + token,
		},
		&config_proto.ControlPlaneCoordinates_Headers{
			Key:   "content-type",
			Value: "application/json",
		},
	}

	c, err := client.ApiServerClient(conf, 30*time.Second)
	if err != nil {
		return nil, err
	}

	return c, nil
}

type GlobalSecretsClient struct {
	client util_http.Client
}

func NewGlobalSecretsClient(c util_http.Client) *GlobalSecretsClient {
	return &GlobalSecretsClient{c}
}

var GlobalSecretNotFound = fmt.Errorf("Global Secret Not Found")

type GlobalSecret struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
}

// Get a Global Secret store and return the base64encoded data
func (sc *GlobalSecretsClient) Get(name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "/global-secrets/"+name, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request for global secrets: %s", err)
	}

	resp, err := sc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to execute request for global secrets: %s", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", GlobalSecretNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected statuscode %d, got %d", http.StatusOK, resp.StatusCode)
	}

	defer resp.Body.Close()

	data := &GlobalSecret{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("unable to decode response: %s", err)
	}

	return data.Data, nil
}

func (sc *GlobalSecretsClient) Put(name, data string) error {
	path := "/global-secrets/" + name

	body := &GlobalSecret{
		Type: "GlobalSecret",
		Name: name,
		Data: data,
	}

	d, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPut, path, bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("unable to create request for global secrets: %s", err)
	}

	req.Header["content-type"] = []string{"application/json"}

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to execute request for global secrets: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	d, _ = ioutil.ReadAll(resp.Body)
	return fmt.Errorf("expected statuscode %d, got %d, path: %s, body: %s", http.StatusOK, resp.StatusCode, path, string(d))
}

type SecretsClient struct {
	client util_http.Client
}

func NewSecretsClient(c util_http.Client) *SecretsClient {
	return &SecretsClient{c}
}

var SecretNotFound = fmt.Errorf("Secret Not Found")

type Secret struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Mesh string `json:"mesh"`
	Data string `json:"data"`
}

// Get a Secret store and return the base64encoded data
func (sc *SecretsClient) Get(mesh, name string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/meshes/%s/secrets/%s", mesh, name), nil)
	if err != nil {
		return "", fmt.Errorf("unable to create request for secrets: %s", err)
	}

	resp, err := sc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to execute request for secrets: %s", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return "", SecretNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected statuscode %d, got %d", http.StatusOK, resp.StatusCode)
	}

	defer resp.Body.Close()

	data := &GlobalSecret{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("unable to decode response: %s", err)
	}

	return data.Data, nil
}

func (sc *SecretsClient) Put(mesh, name, data string) error {
	path := fmt.Sprintf("/meshes/%s/secrets/%s", mesh, name)

	body := &Secret{
		Type: "Secret",
		Name: name,
		Mesh: mesh,
		Data: data,
	}

	d, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPut, path, bytes.NewReader(d))
	if err != nil {
		return fmt.Errorf("unable to create request for secrets: %s", err)
	}

	req.Header["content-type"] = []string{"application/json"}

	resp, err := sc.client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to execute request for secrets: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return nil
	}

	d, _ = ioutil.ReadAll(resp.Body)
	return fmt.Errorf("expected statuscode %d, got %d, path: %s, body: %s", http.StatusOK, resp.StatusCode, path, string(d))
}
