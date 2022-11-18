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

resource "liqo_generate" "generate" {
  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.milan

}

resource "liqo_peering" "peering" {

  /*
    for_each = {
      for index, cluster in var.clusters.clusters_list :
      cluster.name => cluster if index != 0 && var.clusters.peering
    }
  */

  provider = liqo.rome

  cluster_id      = liqo_generate.generate.cluster_id
  cluster_name    = liqo_generate.generate.cluster_name
  cluster_authurl = liqo_generate.generate.auth_ep
  cluster_token   = liqo_generate.generate.local_token

}

resource "null_resource" "create_namespace" {

  provisioner "local-exec" {
    command = "kubectl create namespace liqo-demo"
    environment = {
      KUBECONFIG = module.kind["rome"].kubeconfig_path
    }
  }

}

resource "liqo_offload" "offload" {
  depends_on = [
    null_resource.create_namespace,
    liqo_peering.peering
  ]

  provider = liqo.rome

  namespace = "liqo-demo"
}
