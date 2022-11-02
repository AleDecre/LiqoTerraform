variable "kind_version" {

  type = string

  default = "v1.23.6"

  description = "The kind version to be used."

}

variable "cluster" {

  type = object({

    remote = bool
    name     = string
    networking = object({
      service_subnet = string
      pod_subnet     = string
    })
    peering = number

  })

}
