package main

import (
	"log"
	"os"

	kuma "github.com/gregoryhunt/vault-plugin-kuma"
	dbplugin "github.com/hashicorp/vault/sdk/database/dbplugin/v5"
)

func main() {
	dbType, err := kuma.New()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	dbplugin.Serve(dbType.(dbplugin.Database))
}
