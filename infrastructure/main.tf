terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
    liqo = {
      source = "liqo-provider/liqo/test"
    }
  }
}

provider "kind" {
}

provider "liqo" {
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

resource "liqo_generate" "gen" {

  for_each = {
    for index, cluster in var.clusters.clusters_list :
    cluster.name => cluster if index != 0 && var.clusters.peering
  }

  kubeconfig_path = module.kind[each.value.name].kubeconfig_path

}

resource "liqo_peering" "peer1" {

  for_each = {
    for index, cluster in var.clusters.clusters_list :
    cluster.name => cluster if index != 0 && var.clusters.peering
  }

  kubeconfig_path = module.kind[var.clusters.clusters_list[0].name].kubeconfig_path
  cluster_id      = liqo_generate.gen[each.value.name].cluster_id
  cluster_name      = liqo_generate.gen[each.value.name].cluster_name
  cluster_authurl = liqo_generate.gen[each.value.name].auth_ep
  cluster_token   = liqo_generate.gen[each.value.name].local_token

}