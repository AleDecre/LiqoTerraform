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

provider "liqo" {
  alias = "rome"
  kubernetes = {
    config_path = module.kind["rome"].kubeconfig_path
  }
}

provider "liqo" {
  alias = "milan"
  kubernetes = {
    config_path = module.kind["milan"].kubeconfig_path
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

resource "liqo_generate" "gen1" {
  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.milan

}

resource "liqo_peering" "peer1" {
  
  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.rome

  cluster_id      = liqo_generate.gen1.cluster_id
  cluster_name    = liqo_generate.gen1.cluster_name
  cluster_authurl = liqo_generate.gen1.auth_ep
  cluster_token   = liqo_generate.gen1.local_token

}
/*
  locals {
    providers = {
      rome  = liqo.rome
      milan = liqo.milan
    }
  }
*/
