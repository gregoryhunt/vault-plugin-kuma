GOARCH = amd64

UNAME = $(shell uname -s)

ifndef OS
	ifeq ($(UNAME), Linux)
		OS = linux
	else ifeq ($(UNAME), Darwin)
		OS = darwin
	endif
endif

.DEFAULT_GOAL := all

all: fmt build start

build:
	GOOS=$(OS) GOARCH="$(GOARCH)" go build -o vault/plugins/vault-plugin-database-kuma cmd/vault-plugin-kuma/main.go

start:
	vault server -dev -dev-root-token-id=root -dev-plugin-dir=./vault/plugins

enable:
	vault secrets enable database

	vault write database/config/kuma \
    plugin_name=vault-plugin-kuma \
		allowed_roles="kuma-role" \
    username="vault" \
    password="vault" \
    connection_url="kuma.local:1234"

	vault write database/roles/kuma-role \
    db_name=kuma \
    default_ttl="5m" \
    max_ttl="24h"
clean:
	rm -f ./vault/plugins/vault-plugin-kuma

fmt:
	go fmt $$(go list ./...)

.PHONY: build clean fmt start enable
