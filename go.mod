module github.com/gregoryhunt/vault-plugin-kuma

go 1.16

require (
	github.com/cucumber/godog v0.12.5
	github.com/gobwas/glob v0.2.3
	github.com/hashicorp/go-hclog v1.2.0
	github.com/hashicorp/vault/api v1.7.2
	github.com/hashicorp/vault/sdk v0.5.1
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/kumahq/kuma v0.0.0-20220713112241-38560646a86c
	github.com/mitchellh/go-testing-interface v1.0.4 // indirect
	github.com/stretchr/testify v1.7.1
)

replace github.com/prometheus/prometheus => ./vendored/github.com/prometheus/prometheus
