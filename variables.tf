variable "kind_version" {

  type = string

  default = "v1.23.6"

  description = "The kind version to be used."

}

variable "clusters" {

  type = list(object({

    location = string
    name     = string
    networking = object({
      service_subnet = string
      pod_subnet     = string
    })
    peering = number

  }))

}
