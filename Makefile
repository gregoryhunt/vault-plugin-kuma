ARCH = $(shell uname -m)
UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

ifndef GOARCH
	ifeq ($(ARCH), aarch64)
		GOARCH = arm64
	else ifeq ($(ARCH), arm64)
		GOARCH = arm64
	else ifeq ($(ARCH), x86_64)
		GOARCH = amd64
	else
		GOARCH = $(ARCH)
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

build:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-kuma cmd/vault-plugin-kuma/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins

start_shipyard_env:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -o vault/plugins/vault-plugin-kuma cmd/vault-plugin-kuma/main.go
	shipyard run ./shipyard
	@echo 'Ensure you set the environment variables using eval $(shipyard env) before running any further commands'

restart_vault_shipyard:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -o vault/plugins/vault-plugin-kuma cmd/vault-plugin-kuma/main.go
	shipyard taint container.vault && shipyard run --no-browser ./shipyard

enable:
	vault secrets enable -path=kuma vault-plugin-kuma || true

	vault write kuma/config \
		allowed_roles="kuma-role,kuma-role-globbed" \
		url="http://kuma-cp.container.shipyard.run:5681" \
		token="$(shell cat ~/.shipyard/data/kuma_config/admin.token)"

	# How to differentiate between user token role and dataplane role
	vault write kuma/roles/kuma-role \
		token_name="backend-1" \
    mesh=default \
		tags="kuma.io/service=backend,kuma.io/service=backend-admin" \
    ttl="30s" \
    max_ttl="2m"

	vault write kuma/roles/kuma-role-globbed \
		token_name="backend-*" \
    mesh=default \
		tags="kuma.io/service=backend,kuma.io/service=backend-admin" \
    ttl="5m" \
    max_ttl="24h"

	vault write kuma/roles/kuma-role-user \
		token_name="nic" \
    mesh=default \
		groups="mesh-system:admin" \
    ttl="5m" \
    max_ttl="24h"

test_token_generation:
	vault read kuma/creds/kuma-role token_name=backend-1 || true
	@echo ""
	vault read kuma/creds/kuma-role-globbed || true
	@echo ""
	vault read kuma/creds/kuma-role-globbed token_name=backend-1 || true
	@echo ""
	vault read kuma/creds/kuma-role-user token_name=nic || true

generate:
	vault read kuma/creds/kuma-role -format=json | jq -r .data.token > $(HOME)/.shipyard/data/kuma_dp/dataplane.token
	@echo "Token written to $(HOME)/.shipyard/data/kuma_dp/dataplane.token"

run_dp:
	docker exec -it kuma-dp.container.shipyard.run kuma-dp run --cp-address https://kuma-cp.container.shipyard.run:5678 --dataplane-file /files/dataplane.json --dataplane-token "$(shell cat $(HOME)/.shipyard/data/kuma_dp/dataplane.token)" --ca-cert-file /files/ca.cert

clean:
	rm -f ./vault/plugins/*

tests:
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -o vault/plugins/vault-plugin-kuma cmd/vault-plugin-kuma/main.go
	cd functional_tests && go run main.go

fmt:
	go fmt $$(go list ./...)

.PHONY: build clean fmt start enable
