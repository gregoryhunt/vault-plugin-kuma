package kuma

import (
	"errors"
	"fmt"
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
	dpTokenClient   tokens.DataplaneTokenClient
	userTokenClient userclient.UserTokenClient
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

	return &kumaClient{dpc, uc}, nil
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
