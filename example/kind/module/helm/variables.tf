variable "liqo_provider" {

  type = string

  default = "kind"

  description = "The provider where install liqo."

}

variable "clusters_list" {
  
  type = list(object({
    name = string
    networking = object({
      service_subnet = string
      pod_subnet     = string
    })
  }))

}
