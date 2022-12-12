terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
  }
}

module "kind" {

  for_each = {
    for cluster in var.clusters.clusters_list :
    cluster.name => cluster
  }

  source = "./module/kind"

  cluster      = each.value
  kind_version = var.kind_version

}