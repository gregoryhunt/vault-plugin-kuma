module "vault" {
  source = "github.com/shipyard-run/blueprints?ref=81fa351a4bd62cba284f0b5cb78e6ac8844e2ecd/modules//vault-dev"
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
