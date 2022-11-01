kind_version = "v1.23.6"

clusters = [
  {
    location = "local"
    name     = "rome"
    networking = {
      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    peering = 0
  },
  {
    location = "remote"
    name     = "milan"
    networking = {
      service_subnet = "10.90.0.0/12"
      pod_subnet     = "10.200.0.0/16"
    }
    peering = 0
  }
]
