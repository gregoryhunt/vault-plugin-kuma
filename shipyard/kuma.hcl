module "kuma_cp" {
  source = "github.com/gregoryhunt/kuma-blueprint?ref=499db086de2b4791975f1d732cd28e481dc2fa4f"
}

variable "kuma_cp_network" {
  default     = "local"
  description = "Network name that the Kuma control panel is connected to"
}
