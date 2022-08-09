module "kuma_cp" {
  source = "github.com/gregoryhunt/kuma-blueprint?ref=753f6d9422a2e186d5d4411aca73d0f894bc05a0"
}

variable "kuma_cp_network" {
  default     = "local"
  description = "Network name that the Kuma control panel is connected to"
}

copy "files" {
  source      = "./files/dataplane.json"
  destination = "${data("kuma_dp")}/dataplane.json"
}

copy "ca" {
  depends_on  = ["module.kuma_cp"]
  source      = "${data("kuma_config")}/kuma_cp_ca.cert"
  destination = "${data("kuma_dp")}/ca.cert"
}

container "kuma_dp" {
  image {
    name = "kumahq/kuma-dp:1.7.1"
  }

  entrypoint = [""]

  command = [
    "tail",
    "-f",
    "/dev/null"
  ]

  volume {
    destination = "/files"
    source      = data("kuma_dp")
  }

  network {
    name = "network.local"
  }
}
