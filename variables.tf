variable "kind_version" {

  type = string

  default = "kindest/node:v1.23.6"

  description = "The kind version to be used."

}

variable "networking_local" {

  type = object({
    service_subnet = string
    pod_subnet     = string
  })

  default = {
    service_subnet = "10.90.0.0/12"
    pod_subnet     = "10.200.0.0/16"
  }

  description = "The local cluster pod/service CIDR."

}

variable "networking_remote" {

  type = object({
    service_subnet = string
    pod_subnet     = string
  })

  default = {
    service_subnet = "10.90.0.0/12"
    pod_subnet     = "10.200.0.0/16"
  }

  description = "The remote cluster pod/service CIDR."

}

variable "peering" {

  type = number

  default = 1

  validation {
    condition     = var.peering >= 0 && var.peering <= 1
    error_message = "The peering value must be 0 or 1 (default 1)."
  }

  description = "Choice to execute peering at startup"

}
