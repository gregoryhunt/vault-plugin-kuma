module "vault" {
  source = "github.com/shipyard-run/blueprints?ref=46ff6abb218b75386af4c71e2d700279b540343b/modules//vault-dev"
}

variable "vault_network" {
  default = "local"
}

variable "vault_plugin_folder" {
  default = "${file_dir()}/../vault/plugins"
}

