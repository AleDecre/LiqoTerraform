terraform {
  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "0.0.15"
    }
  }
}

provider "kind" {
}

module "kind" {

  for_each = {
    for cluster in var.clusters :
    cluster.name => cluster
  }

  source = "./module/kind"

  cluster      = each.value
  kind_version = var.kind_version

}

/*
  resource "null_resource" "cluster_peering" {

    for_each = {
      for cluster in var.clusters :
      cluster.name => cluster if cluster.remote
    }

    provisioner "local-exec" {

      command = "$(liqoctl generate peer-command --kubeconfig \"$KUBECONFIG_REMOTE\" | tail -n 1)"

      environment = {
        KUBECONFIG       = "${kind_cluster.default["rome"].kubeconfig_path}"
        KUBECONFIG_REMOTE = "${kind_cluster.default[var.cluster.name].kubeconfig_path}"
      }

    }

  }
*/
