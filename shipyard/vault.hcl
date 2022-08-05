module "vault" {
  source = "github.com/shipyard-run/blueprints?ref=144a4b75e44a8471d1f9b30d6f8a30c8d9e05e7e/modules//vault-dev"
}

variable "vault_network" {
  default = "local"
}

variable "vault_plugin_folder" {
  default     = "${file_dir()}/../vault/plugins"
  description = "Folder where vault will load custom plugins"
}

variable "vault_bootstrap_script" {
  default = <<-EOF
  #/bin/sh -e
  vault status

  EOF
}
