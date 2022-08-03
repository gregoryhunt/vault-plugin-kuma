module "kuma_cp" {
  source = "github.com/gregoryhunt/kuma-blueprint?ref=c3bced6ca949da16367aa08887b17c6a074eb61f"
}

variable "kuma_cp_network" {
  default     = "local"
  description = "Network name that the Kuma control panel is connected to"
}
