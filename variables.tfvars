kind_version = "v1.23.6"

clusters = [
  {
    remote = false
    name     = "rome"
    networking = {
      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    peering = 0
  },
  {
    remote = true
    name     = "milan"
    networking = {
      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    peering = 0
  }
]
