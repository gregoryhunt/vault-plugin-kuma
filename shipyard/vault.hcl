module "vault" {
  source = "github.com/shipyard-run/blueprints?ref=f235847a73c5bb81943aaed8f0c526edee693d75/modules//vault-dev"
}

variable "vault_network" {
  default = "dc1"
}

