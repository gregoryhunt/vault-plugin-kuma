package kuma

import (
	"errors"
	"fmt"
	"time"

	"github.com/kumahq/kuma/app/kumactl/pkg/client"
	"github.com/kumahq/kuma/app/kumactl/pkg/tokens"
	config_proto "github.com/kumahq/kuma/pkg/config/app/kumactl/v1alpha1"
	util_http "github.com/kumahq/kuma/pkg/util/http"
)

// harborClient creates an object storing
// the client.
type kumaClient struct {
	client tokens.DataplaneTokenClient
}

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

	apiclient, err := baseAPIServerClient(config.URL)
	if err != nil {
		return nil, fmt.Errorf("unable to create base API client: %s", err)
	}

	c := tokens.NewDataplaneTokenClient(apiclient)

	return &kumaClient{c}, nil
}

func baseAPIServerClient(url string) (util_http.Client, error) {

	conf := &config_proto.ControlPlaneCoordinates_ApiServer{}
	conf.Url = url

	c, err := client.ApiServerClient(conf, 30*time.Second)
	if err != nil {
		return nil, err
	}

	return c, nil
}
